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
	thisAddr             string
	validators           []*ValidatorNode
	hostsExceptMe        []string // массив адресов вида 1.1.1.1:1337
	prevBlockHash        [HASH_SIZE]byte
	prevBlockHashes      [MAX_PREV_BLOCK_HASHES][HASH_SIZE]byte
	currentLeader        [PKEY_SIZE]byte
	currentBock          *BlocAndkHash
	unrecordedTrans      []TransAndHash
	nextTickTime         time.Time
	blockAppendTime      time.Duration
	blockVotingTime      time.Duration
	justWaitingTime      time.Duration
	startupDelay         time.Duration
	chainSize            uint64
	blockVoting          map[[PKEY_SIZE]byte]int
	kickVoting           map[[PKEY_SIZE]byte]int
	suspiciousValidators map[[PKEY_SIZE]byte]int
	tickPreparation      chan bool // каналы, которые задают цикл работы ноды
	tickThisLeader       chan bool
	tickVoting           chan bool
	tickVotingProcessing chan bool
	done                 chan bool // канал, сообщение в котором заставляет завершиться прогу
	network              *Network
	chs                  *NetworkChannels
	expectBlocks         bool
}

func (bc *Blockchain) Setup(thisPrv []byte, thisAddr string, validators []*ValidatorNode,
	blockAppendTime time.Duration, blockVotingTime time.Duration,
	justWaitingTime time.Duration, startupDelay time.Duration,
	startBlockHash [HASH_SIZE]byte) {
	//зачатки констуруктора q
	var k CryptoKeysData
	k.SetupKeys(thisPrv)
	bc.thisKey = &k
	bc.thisAddr = thisAddr

	bc.validators = validators

	bc.blockAppendTime = blockAppendTime
	bc.blockVotingTime = blockVotingTime
	bc.justWaitingTime = justWaitingTime
	bc.startupDelay = startupDelay
	bc.nextTickTime = time.Now().Add(blockAppendTime).Add(blockVotingTime).Add(justWaitingTime)

	bc.hostsExceptMe = make([]string, 0)

	bc.blockVoting = make(map[[PKEY_SIZE]byte]int)
	bc.kickVoting = make(map[[PKEY_SIZE]byte]int)
	bc.suspiciousValidators = make(map[[PKEY_SIZE]byte]int)

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
	bc.prevBlockHash = startBlockHash

	bc.chainSize = 0
	bc.tickPreparation = make(chan bool, 1)
	bc.tickThisLeader = make(chan bool, 1)
	bc.tickVoting = make(chan bool, 1)
	bc.tickVotingProcessing = make(chan bool, 1)
	bc.done = make(chan bool, 1)
	bc.network = new(Network)
	bc.chs = bc.network.Init()
	bc.expectBlocks = false
}

func (bc *Blockchain) Start() {
	bc.tickPreparation <- true
	go bc.network.Serve(bc.thisAddr) // запускаем сеть в отдельной горутине, не блокируем текущий поток
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
		case <-bc.tickPreparation:
			fmt.Println("Do tick")
			bc.doTickPreparation() // запускаем тик в фоне, чтобы он не стопил основной цикл
			// потом эта функция положит сигнал в канал bc.tickThisLeader
		case <-bc.tickThisLeader:
			bc.doTickThisLeader() // потом эта функция положит сигнал в bc.tickVoting
		case <-bc.tickVoting:
			bc.doTickVoting() // эта положит в bc.tickVotingProcessing
		case <-bc.tickVotingProcessing:
			bc.doTickVotingProcessing() // эта положит в bc.tickPreparation
		case msg := <-bc.chs.blocks:
			// нужно обработчики блоков вынести в отдельные горутины
			fmt.Println("got new block")
			bc.onBlockReceive(msg.data, msg.response)
		case msg := <-bc.chs.blockVotes:
			fmt.Println("got block vote")
			bc.onBlockVote(msg.data, msg.response)
		case msg := <-bc.chs.kickValidatorVote:
			fmt.Println("got kick validator vote")
			bc.onKickValidatorVote(msg.data, msg.response)
		case msg := <-bc.chs.txsValidator:
			// транза от валидатора
			fmt.Println("transaction from validator", msg)
		case msg := <-bc.chs.txsClient:
			// транза от приложения-клиента
			fmt.Println("transaction from client", msg)
		}
	}
}

func (bc *Blockchain) getTimeOfNextTick(lastBlockTime time.Time) time.Time {
	return lastBlockTime.Add(bc.blockAppendTime).Add(bc.blockVotingTime).Add(bc.justWaitingTime)
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
	fmt.Println("block len", blockLen)
	if blockLen == ERR_BLOCK_CREATOR {
		response <- ResponseMsg{ok: true}
		return
	}

	if blockLen != len(data) {
		bc.blockVoting[bc.thisKey.pubKeyByte] = 2
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect block",
		}
		return
	}

	bc.nextTickTime = bc.getTimeOfNextTick(time.Unix(0, int64(b.timestamp)))
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	bc.blockVoting[bc.thisKey.pubKeyByte] = 1
	// голосование за/против блока в doTickVoting
	response <- ResponseMsg{ok: true}
}

func (bc *Blockchain) onBlockVote(data []byte, response chan ResponseMsg) {
	if len(data) != HASH_SIZE+PKEY_SIZE+1+SIG_SIZE {
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect msg length",
		}
		return
	}
	var hash [HASH_SIZE]byte
	var pkey [PKEY_SIZE]byte
	var vote [1]byte
	var sig [SIG_SIZE]byte
	copy(hash[:], data[:HASH_SIZE])
	copy(pkey[:], data[HASH_SIZE:HASH_SIZE+PKEY_SIZE])
	copy(vote[:], data[HASH_SIZE+PKEY_SIZE:HASH_SIZE+PKEY_SIZE+1])
	copy(sig[:], data[HASH_SIZE+PKEY_SIZE+1:])
	_, ok := bc.blockVoting[pkey]
	if !ok || !VerifyData(data[:HASH_SIZE+PKEY_SIZE+1], sig[:], pkey) {
		response <- ResponseMsg{
			ok:    false,
			error: "unknown block vote sender",
		}
		return
	}

	if hash == bc.prevBlockHash && (vote[0] == 0x01 || vote[0] == 0x02) {
		bc.blockVoting[pkey] = int(vote[0])
	} else {
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect block vote data",
		}
		return
	}
	response <- ResponseMsg{ok: true}
}

func (bc *Blockchain) onKickValidatorVote(data []byte, response chan ResponseMsg) {
	if len(data) != PKEY_SIZE*2+SIG_SIZE {
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect vote length",
		}
		return
	}
	var kickPkey [PKEY_SIZE]byte
	var senderPkey [PKEY_SIZE]byte
	var sig [SIG_SIZE]byte
	copy(kickPkey[:], data[:PKEY_SIZE])
	copy(senderPkey[:], data[PKEY_SIZE:PKEY_SIZE*2])
	copy(sig[:], data[PKEY_SIZE*2:])
	_, ok := bc.kickVoting[senderPkey]
	if !ok || !VerifyData(data[:PKEY_SIZE*2], sig[:], senderPkey) {
		response <- ResponseMsg{
			ok:    false,
			error: "unknown kick vote sender",
		}
		return
	}
	if kickPkey == bc.thisKey.pubKeyByte {
		response <- ResponseMsg{ok: true}
		return
	}
	bc.kickVoting[kickPkey] += 1
	response <- ResponseMsg{ok: true}
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
	var clearedSuspiciousValidators = make(map[[PKEY_SIZE]byte]int)
	for _, validator := range bc.validators {
		bc.kickVoting[validator.pkey] = 0
		clearedSuspiciousValidators[validator.pkey] = bc.suspiciousValidators[validator.pkey]
	}
	bc.suspiciousValidators = clearedSuspiciousValidators
}

func (bc *Blockchain) ClearBlockVoting() {
	bc.blockVoting = make(map[[PKEY_SIZE]byte]int, 0)
	for _, validator := range bc.validators {
		bc.blockVoting[validator.pkey] = 0
	}
}

// Функция doTick() поделена на отдельные этапы, между которыми было ожидание
// Теперь все эти этапы выполняются в основном потоке, а во время ожидания могу обрабатываться входящие блоки и транзы
func (bc *Blockchain) doTickPreparation() {
	for k, v := range bc.suspiciousValidators {
		if v > 0 {
			fmt.Println("suspicious validator", k, v)
		}
	}
	fmt.Println("process kick")
	bc.processKick()

	bc.currentLeader = bc.validators[bc.chainSize%uint64(len(bc.validators))].pkey
	bc.nextTickTime = bc.getTimeOfNextTick(time.Now())
	if bc.expectBlocks == false {
		bc.expectBlocks = true
		bc.nextTickTime = bc.nextTickTime.Add(bc.startupDelay)
		go func() {
			time.Sleep(bc.startupDelay)
			bc.tickThisLeader <- true
		}()
	} else {
		bc.tickThisLeader <- true
	}
}

func (bc *Blockchain) doTickThisLeader() {
	bc.currentBock = nil
	if bc.thisKey.pubKeyByte == bc.currentLeader {
		fmt.Println("this == leader")
		bc.expectBlocks = false
		bc.onThisCreateBlock()
	}
	timeWhileBlockIsReceived := bc.nextTickTime.Add(-bc.blockVotingTime - bc.justWaitingTime).Sub(time.Now())
	fmt.Println("sleeping for ", timeWhileBlockIsReceived)
	go func() {
		time.Sleep(timeWhileBlockIsReceived) // sleep for blockAppendTime
		bc.tickVoting <- true
	}()
}

// Фаза голосования, если блока не было, то голоса отправляются против
func (bc *Blockchain) doTickVoting() {
	fmt.Println("block voting time")
	var voteData [HASH_SIZE + PKEY_SIZE + 1 + SIG_SIZE]byte
	copy(voteData[:HASH_SIZE], bc.prevBlockHash[:])
	copy(voteData[HASH_SIZE:HASH_SIZE+PKEY_SIZE], bc.thisKey.pubKeyByte[:])
	var vote [1]byte
	if bc.currentBock != nil {
		vote[0] = 0x01
	} else {
		vote[0] = 0x02
	}
	copy(voteData[HASH_SIZE+PKEY_SIZE:HASH_SIZE+PKEY_SIZE+1], vote[:])
	copy(voteData[HASH_SIZE+PKEY_SIZE+1:], ZERO_ARRAY_SIG[:])
	copy(voteData[HASH_SIZE+PKEY_SIZE+1:], bc.thisKey.Sign(voteData[:]))
	go bc.network.SendVoteToAll(bc.hostsExceptMe, voteData[:])
	var timeWhileVotesAreReceived = bc.nextTickTime.Add(-bc.justWaitingTime).Sub(time.Now())
	fmt.Println("sleeping for ", timeWhileVotesAreReceived)
	go func() {
		time.Sleep(timeWhileVotesAreReceived) // sleep from blockVotingTime
		bc.tickVotingProcessing <- true
	}()
}

func (bc *Blockchain) doTickVotingProcessing() {
	fmt.Println("process voting")
	yesVote, noVote := 0, 0
	for pkey, vote := range bc.blockVoting {
		if pkey != bc.currentLeader {
			if vote == 0x01 {
				yesVote += 1
				bc.suspiciousValidators[pkey] = 0
			} else if vote == 0x02 {
				noVote += 1
			} else {
				bc.suspiciousValidators[pkey] += 1
			}
		}
	}

	if noVote < yesVote {
		fmt.Println("block accepted")
		bc.updatePrevHashBlock()
		//запись блока в БД
		bc.updateUnrecordedTrans()
		bc.chainSize += 1
		bc.suspiciousValidators[bc.currentLeader] = 0
	} else if bc.currentLeader != bc.thisKey.pubKeyByte {
		fmt.Println("block rejected")
		bc.suspiciousValidators[bc.currentLeader] += 1
	}
	fmt.Println("vote kick check")
	bc.tryKickValidator()

	fmt.Println("clear block voting")
	bc.ClearBlockVoting()  // чистим голоса за блок до начала получения новых блоков
	bc.expectBlocks = true // меняем флаг заранее, чтобы не пропустить блок
	timeBeforeNextTick := bc.nextTickTime.Sub(time.Now())
	fmt.Println("time before next tick", timeBeforeNextTick)
	go func() {
		time.Sleep(timeBeforeNextTick) // sleep for justWaitingTime
		bc.tickPreparation <- true
	}()
}

func (bc *Blockchain) onThisCreateBlock() {
	var b Block
	transSize := len(bc.unrecordedTrans)
	if transSize > MAX_TRANS_SIZE {
		transSize = MAX_TRANS_SIZE
	}
	b.CreateBlock(bc.unrecordedTrans[:transSize], bc.prevBlockHash, bc.thisKey)
	blockBytes := b.ToBytes()
	hash := b.HashBlock(blockBytes)
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	go bc.network.SendBlockToAll(bc.hostsExceptMe, blockBytes)
}

func (bc *Blockchain) tryKickValidator() {
	for pkey, v := range bc.suspiciousValidators {
		if v > 1 {
			bc.kickVoting[pkey] += 1
			var data [PKEY_SIZE*2 + SIG_SIZE]byte
			copy(data[:PKEY_SIZE], pkey[:])
			copy(data[PKEY_SIZE:PKEY_SIZE*2], bc.thisKey.pubKeyByte[:])
			copy(data[PKEY_SIZE*2:], ZERO_ARRAY_SIG[:])
			copy(data[PKEY_SIZE*2:], bc.thisKey.Sign(data[:]))
			go bc.network.SendKickMsgToAll(bc.hostsExceptMe, data[:])
		}
	}
}
