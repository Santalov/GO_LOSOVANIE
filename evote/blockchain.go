package evote

import (
	"fmt"
	"time"
)

/*
1) on block voting:
vote 0 == no vote
vote 1 == yes
vote 2 == no

2) on suspicious validators:
vote == count no vote/bad blocks
vote can be 0 or 1,
if (vote > 1) then process kick start

3) On kick voting:
0 <= vote <= len(validatePkeys)
*/

type ValidatorNode struct {
	pkey [PKEY_SIZE]byte
	addr string // адрес вида 1.1.1.1:1337
}

type Blockchain struct {
	thisKey              *CryptoKeysData
	validators           []*ValidatorNode
	hostsExceptMe        []string // массив адресов вида 1.1.1.1:1337
	prevBlockHash        [HASH_SIZE]byte
	prevBlockHashes      [MAX_PREV_BLOCK_HASHES][HASH_SIZE]byte
	currentLeader        [PKEY_SIZE]byte
	currentBock          *BlocAndkHash
	unrecordedTrans      []TransAndHash
	nextLeaderVoteTime   time.Time
	nextLeaderPeriod     time.Duration
	blockAppendTime      time.Duration
	chainSize            uint64
	blockVoting          map[[PKEY_SIZE]byte]int
	kickVoting           map[[PKEY_SIZE]byte]int
	suspiciousValidators map[[PKEY_SIZE]byte]int
	ticker               chan bool // канал, который задает цикл работы ноды
	done                 chan bool // канал, сообщение в котором заставляет завершиться прогу
	network              *Network
	chs                  *NetworkChannels
	expectBlocks         bool
}

func (bc *Blockchain) Setup(thisPrv []byte, validators []*ValidatorNode,
	nextVoteTime time.Time, nextPeriod time.Duration, appendTime time.Duration) {
	//зачатки констуруктора
	var k CryptoKeysData
	k.SetupKeys(thisPrv)
	bc.thisKey = &k

	bc.validators = validators

	bc.nextLeaderVoteTime = nextVoteTime
	bc.nextLeaderPeriod = nextPeriod
	bc.blockAppendTime = appendTime
	bc.hostsExceptMe = make([]string, len(bc.validators)-1)

	for _, validator := range bc.validators {
		bc.blockVoting[validator.pkey] = 0
		bc.kickVoting[validator.pkey] = 0
		bc.suspiciousValidators[validator.pkey] = 0
		if validator.pkey != bc.thisKey.pubKeyByte {
			bc.hostsExceptMe = append(bc.hostsExceptMe, validator.addr)
		}
	}

	var blockInCons BlocAndkHash
	bc.currentBock = &blockInCons

	bc.chainSize = 0
	bc.ticker = make(chan bool)
	bc.done = make(chan bool)
	bc.network = new(Network)
	bc.chs = bc.network.Init()
}

func (bc *Blockchain) Start() {
	bc.ticker <- true
	go bc.network.Serve() // запускаем сеть в отдельной горутине, не блокируем текущий поток
	for {
		// бесконечно забираем сообщения из каналов
		select {
		// этот код будет выбирать сообщения из того канала, в котором оно первым появится
		// то есть одновременно будут приходить блоки, транзы и "тики",
		// и они будут последовательно обрабатываться в этом цикле
		// каждом каналу (то есть типу сообщений) отвечает соответсвующая секция
		case <-bc.done: // в этот канал поступает сигнал об остановке
			// механизм завершения не реализован
			fmt.Println("I must stop!")
			return
		case <-bc.ticker:
			go bc.doTick() // запускаем тик в фоне, чтобы он не стопил основной цикл
		// сам тик потом сделает bc.ticker<-true, чтобы цикл продолжился
		case msg := <-bc.chs.blocks:
			// нужно обработчики блоков вынести в отдельные горутины
			bc.onBlockReceive(msg.data, msg.response)
		case msg := <-bc.chs.blockVotes:
			bc.onBlockVote(msg.data)
			// ответ в сеть всегда положительный, голос всегда принимается
			msg.response <- ResponseMsg{ok: true}
		case msg := <-bc.chs.kickValidatorVote:
			bc.onKickValidatorVote(msg.data)
			// ответ в сеть всегда положительный, голос всегда принимается
			msg.response <- ResponseMsg{ok: true}
		case msg := <-bc.chs.txsValidator:
			// транза от валидатора
			fmt.Println("transaction from validator", msg)
		case msg := <-bc.chs.txsClient:
			// транза от приложения-клиента
			fmt.Println("transaction from client", msg)
		}
	}
}

func (bc *Blockchain) onBlockReceive(data []byte, response chan ResponseMsg) {
	if !bc.expectBlocks {
		response <- ResponseMsg{
			ok:    false,
			error: "unexpected block",
		}
		return
	}
	var b Block
	hash, blockLen := b.Verify(data, bc.prevBlockHash, bc.currentLeader)
	if blockLen == ERR_BLOCK_CREATOR {
		response <- ResponseMsg{ok: true}
		return
	}
	if blockLen != len(data) {
		bc.suspiciousValidators[bc.currentLeader] = 1
		bc.blockVoting[bc.thisKey.pubKeyByte] = 2
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect block",
		}
		return
	}

	bc.nextLeaderVoteTime = time.Unix(0, int64(b.timestamp)).Add(bc.nextLeaderPeriod)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	bc.blockVoting[bc.thisKey.pubKeyByte] = 1
	response <- ResponseMsg{ok: true}
}

func (bc *Blockchain) onBlockVote(data []byte) {
	if len(data) != HASH_SIZE+PKEY_SIZE+1 {
		return
	}
	var hash [HASH_SIZE]byte
	var pkey [PKEY_SIZE]byte
	var vote [1]byte
	copy(hash[:], data[:HASH_SIZE])
	copy(pkey[:], data[HASH_SIZE:PKEY_SIZE])
	copy(vote[:], data[HASH_SIZE+PKEY_SIZE:])
	_, ok := bc.blockVoting[pkey]
	if hash == bc.prevBlockHash && ok {
		if vote[0] == 0x01 || vote[0] == 0x02 {
			bc.blockVoting[pkey] = int(vote[0])
		}
	}
}

func (bc *Blockchain) onKickValidatorVote(data []byte) {
	if len(data) != PKEY_SIZE {
		return
	}
	var kickPkey [PKEY_SIZE]byte
	copy(kickPkey[:], data)
	if kickPkey == bc.thisKey.pubKeyByte {
		return
	}
	bc.kickVoting[kickPkey] += 1
}

func (bc *Blockchain) updatePrevHashBlock() {
	for i := MAX_PREV_BLOCK_HASHES - 1; i >= 1; i-- {
		bc.prevBlockHashes[i] = bc.prevBlockHashes[i-1]
	}
	bc.prevBlockHashes[0] = bc.currentBock.hash
}

func (bc *Blockchain) containsTransInBlock(hash [HASH_SIZE]byte) bool {
	for _, val := range bc.currentBock.b.trans {
		if hash == val.hash {
			return true
		}
	}
	return false
}

func (bc *Blockchain) updateUnrecordedTrans() {
	var newUnrecorded []TransAndHash
	for _, t := range bc.unrecordedTrans {
		if !bc.containsTransInBlock(t.hash) {
			newUnrecorded = append(newUnrecorded, t)
		}
	}
}

func (bc *Blockchain) processKick() {
	for k, v := range bc.kickVoting {
		if float32(v)/float32(len(bc.validators)) > 0.5 {
			for i, validator := range bc.validators {
				if validator.pkey == k {
					bc.validators = append(bc.validators[:i], bc.validators[i+1:]...)
					break
				}
			}
		}
	}
	bc.kickVoting = make(map[[PKEY_SIZE]byte]int, 0)
	for _, validator := range bc.validators {
		bc.kickVoting[validator.pkey] = 0
	}
}

func (bc *Blockchain) ClearBlockVoting() {
	bc.blockVoting = make(map[[PKEY_SIZE]byte]int, 0)
	for _, validator := range bc.validators {
		bc.blockVoting[validator.pkey] = 0
	}
}

func (bc *Blockchain) doTick() {
	bc.processKick()

	bc.currentLeader = bc.validators[bc.chainSize%uint64(len(bc.validators))].pkey

	if bc.thisKey.pubKeyByte == bc.currentLeader {
		bc.expectBlocks = true
		bc.nextLeaderVoteTime = time.Now().Add(bc.nextLeaderPeriod)
		bc.onThisCreateBlock()
	}

	time.Sleep(bc.nextLeaderVoteTime.Add(-bc.blockAppendTime).Sub(time.Now()))

	yesVote, noVote := 0, 0
	for pkey, vote := range bc.blockVoting {
		if vote == 0 {
			bc.suspiciousValidators[pkey] += 1
			if bc.suspiciousValidators[pkey] > 1 {
				//vote kick
			}
			noVote += 1
		} else if vote == 1 {
			yesVote += 1
			bc.suspiciousValidators[pkey] = 0
		} else {
			noVote += 1
		}
	}

	if noVote < yesVote {
		bc.updatePrevHashBlock()
		//запись блока в БД
		bc.updateUnrecordedTrans()
		bc.chainSize += 1
	} else if bc.currentLeader != bc.thisKey.pubKeyByte {
		bc.suspiciousValidators[bc.currentLeader] += 1
		if bc.suspiciousValidators[bc.currentLeader] > 1 {
			//vote kick
		}
	}
	bc.ClearBlockVoting()   // чистим голоса за блок до начала получения новых блоков
	bc.expectBlocks = false // меняем флаг заранее, чтобы не пропустить блок
	time.Sleep(bc.nextLeaderVoteTime.Sub(time.Now()))
	bc.ticker <- true
}

func (bc *Blockchain) onThisCreateBlock() {
	var b Block
	b.CreateBlock(bc.unrecordedTrans[:MAX_TRANS_SIZE], bc.prevBlockHash, bc.thisKey)
	blockBytes := b.ToBytes()
	hash := b.HashBlock(blockBytes)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	bc.network.SendBlockToAll(bc.hostsExceptMe, blockBytes)
}

func (bc *Blockchain) voteKickValidator(pkey [PKEY_SIZE]byte) {
	bc.kickVoting[pkey] += 1
}
