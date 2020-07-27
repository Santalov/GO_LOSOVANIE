package evote

import (
	"encoding/binary"
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

const (
	INACTIVE  = 0
	VIEWER    = 1
	VALIDATOR = 2
)

type ValidatorNode struct {
	Pkey [PKEY_SIZE]byte
	Addr string // адрес вида 1.1.1.1:1337
}

type Blockchain struct {
	thisKey              *CryptoKeysData
	thisValidator        *ValidatorNode
	activeValidators     []*ValidatorNode
	allValidators        []*ValidatorNode
	activeHostsExceptMe  []string // массив адресов вида 1.1.1.1:1337
	prevBlockHash        [HASH_SIZE]byte
	prevBlockHashes      [MAX_PREV_BLOCK_HASHES][HASH_SIZE]byte
	currentLeader        *ValidatorNode
	currentBock          *BlocAndkHash
	unrecordedTrans      []TransAndHash
	nextTickTime         time.Time
	blockAppendTime      time.Duration
	blockVotingTime      time.Duration
	justWaitingTime      time.Duration
	startupDelay         time.Duration
	chainSize            uint64
	genBlocksCount       uint64
	blockVoting          map[*ValidatorNode]int
	kickVoting           map[*ValidatorNode]int
	appendVoting         map[*ValidatorNode]int
	appendVotingMe       map[*ValidatorNode]int
	suspiciousValidators map[*ValidatorNode]int
	tickPreparation      chan bool // каналы, которые задают цикл работы ноды
	tickThisLeader       chan bool
	tickVoting           chan bool
	tickVotingProcessing chan bool
	done                 chan bool // канал, сообщение в котором заставляет завершиться прогу
	network              *Network
	chs                  *NetworkChannels
	expectBlocks         bool
	validatorStatus      int
	db                   *Database

	//map для удобного получения валидаторов
	addrToValidator map[string]*ValidatorNode
	pkeyToValidator map[[PKEY_SIZE]byte]*ValidatorNode
}

func (bc *Blockchain) Setup(thisPrv []byte, thisAddr string, validators []*ValidatorNode,
	blockAppendTime time.Duration, blockVotingTime time.Duration,
	justWaitingTime time.Duration, startupDelay time.Duration,
	startBlockHash [HASH_SIZE]byte, dbPort int) {
	//зачатки констуруктора q
	var k CryptoKeysData
	k.SetupKeys(thisPrv)
	bc.thisKey = &k

	bc.blockVoting = make(map[*ValidatorNode]int)
	bc.kickVoting = make(map[*ValidatorNode]int)
	bc.suspiciousValidators = make(map[*ValidatorNode]int)
	bc.appendVoting = make(map[*ValidatorNode]int)
	bc.appendVotingMe = make(map[*ValidatorNode]int)
	bc.addrToValidator = make(map[string]*ValidatorNode)
	bc.pkeyToValidator = make(map[[PKEY_SIZE]byte]*ValidatorNode)
	bc.activeHostsExceptMe = make([]string, 0)

	bc.allValidators = validators
	for _, v := range bc.allValidators {
		if bc.thisKey.PubkeyByte == v.Pkey {
			bc.thisValidator = v
		}
		bc.pkeyToValidator[v.Pkey] = v
		bc.addrToValidator[v.Addr] = v
	}

	bc.blockAppendTime = blockAppendTime
	bc.blockVotingTime = blockVotingTime
	bc.justWaitingTime = justWaitingTime
	bc.nextTickTime = bc.getTimeOfNextTick(time.Now())

	//load prev from DB
	bc.prevBlockHash = startBlockHash

	bc.chainSize = 0
	bc.genBlocksCount = 0
	bc.tickPreparation = make(chan bool, 1)
	bc.tickThisLeader = make(chan bool, 1)
	bc.tickVoting = make(chan bool, 1)
	bc.tickVotingProcessing = make(chan bool, 1)
	bc.done = make(chan bool, 1)
	bc.network = new(Network)
	bc.chs = bc.network.Init()
	bc.expectBlocks = true
	bc.validatorStatus = INACTIVE

	bc.db = new(Database)
	err := bc.db.Init(DBNAME, DBUSER, DBPASSWORD, DBHOST, dbPort)
	if err != nil {
		panic(err)
	}
}

func (bc *Blockchain) Start() {
	go bc.network.Serve(bc.thisValidator.Addr) // запускаем сеть в отдельной горутине, не блокируем текущий поток
	bc.prepare()
	bc.tickPreparation <- true
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
			fmt.Println("Transaction from validator", msg)
			bc.onTransReceive(msg.data, msg.response)
		case msg := <-bc.chs.txsClient:
			// транза от приложения-клиента
			fmt.Println("Transaction from client", msg)
			if bc.onTransReceive(msg.data, msg.response) {
				go bc.network.SendTxToAll(bc.activeHostsExceptMe, msg.data)
			}
		case msg := <-bc.chs.blockAfter:
			//запрос на блок от INACTIVE или VIEWER
			bc.onGetBlockAfter(msg.data, msg.response)
		case msg := <-bc.chs.appendViewer:
			fmt.Println("on append viewer")
			//запрос добавление VIEWER
			bc.onAppendViewer(msg.data, msg.response)
		case msg := <-bc.chs.appendValidatorVote:
			//голосование за добавление валидатора
			fmt.Println("on append vote incoming")
			bc.onAppendVote(msg.data, msg.response)
		case msg := <-bc.chs.getTxsByHashes:
			fmt.Println("get txs by hashes request")
			bc.onGetTxsByHashes(msg.data, msg.response)
		case msg := <-bc.chs.getTxsByPkey:
			fmt.Println("get txs by pkey request")
			bc.onGetTxsByPkey(msg.data, msg.response)
		case msg := <-bc.chs.getUtxosByPkey:
			fmt.Println("get utxos by pkey request")
			bc.onGetUtxosByPkey(msg.data, msg.response)
		case msg := <-bc.chs.faucet:
			fmt.Println("get money by pkey request")
			bc.onGetMoneyRequest(msg.data, msg.response)
		case msg := <-bc.chs.getVoteResult:
			fmt.Println("get money by pkey request")
			bc.onGetVoteResult(msg.data, msg.response)
		}
	}
}

func (bc *Blockchain) getTimeOfNextTick(lastBlockTime time.Time) time.Time {
	return lastBlockTime.Add(bc.blockAppendTime).Add(bc.blockVotingTime).Add(bc.justWaitingTime)
}

func (bc *Blockchain) appendUnrecordedTrans(t *Transaction, hash []byte) {
	var transHash TransAndHash
	transHash.Transaction = t
	copy(transHash.Hash[:], hash)
	bc.unrecordedTrans = append(bc.unrecordedTrans, transHash)
}

func (bc *Blockchain) onTransReceive(data []byte, response chan ResponseMsg) bool {
	if bc.validatorStatus != VALIDATOR {
		response <- ResponseMsg{
			ok:    false,
			error: "i'm not validator",
		}
		return false
	}
	var t Transaction
	hash, transLen := t.Verify(data, bc.db)
	fmt.Println("trans len", transLen)
	if transLen != len(data) {
		response <- ResponseMsg{
			ok:    false,
			error: "bad transaction from client",
		}
		return false
	}
	bc.appendUnrecordedTrans(&t, hash)
	response <- ResponseMsg{
		ok: true,
	}
	return true

}

func (bc *Blockchain) onBlockReceive(data []byte, response chan ResponseMsg) {
	if bc.validatorStatus == VALIDATOR {
		bc.onBlockReceiveValidator(data, response)
	} else {
		bc.onBlockReceiveViewer(data, response)
	}
}

func (bc *Blockchain) onBlockReceiveValidator(data []byte, response chan ResponseMsg) {
	if !bc.expectBlocks {
		response <- ResponseMsg{
			ok:    false,
			error: "unexpected block",
		}
		return
	}
	var b Block
	hash, blockLen := b.Verify(data, bc.prevBlockHash, bc.currentLeader.Pkey, bc.db)
	fmt.Println("block len", blockLen)
	if blockLen == ERR_BLOCK_CREATOR {
		response <- ResponseMsg{ok: true}
		return
	}

	if blockLen != len(data) {
		bc.blockVoting[bc.thisValidator] = 2
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect block",
		}
		return
	}

	bc.nextTickTime = bc.getTimeOfNextTick(time.Unix(0, int64(b.Timestamp)))
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.Hash[:], hash)
	bc.currentBock.B = &b

	bc.blockVoting[bc.thisValidator] = 1
	// голосование за/против блока в doTickVoting
	response <- ResponseMsg{ok: true}
}

func (bc *Blockchain) onBlockReceiveViewer(data []byte, response chan ResponseMsg) {
	resp := ResponseMsg{ok: true}
	var creator [PKEY_SIZE]byte
	var b Block
	hash, blockLen := b.Verify(data, bc.prevBlockHash, creator, bc.db)
	fmt.Println("block len", blockLen)

	if blockLen != len(data) {
		addr := bc.activeValidators[(bc.chainSize+1)%uint64(len(bc.activeValidators))].Addr
		// опасно начинать общаться с другими участниками сети до отсылки ответа текущему
		// т.е опасно делать response <- resp после bc.getMissingBlock(Addr)
		bc.getMissingBlock(addr)
		response <- resp
		return
	}

	// установку nextTimeTick возможно надо перенести наверх, до вызова bc.getMissingBlock
	// а из getMissingBlock установку nextTickTime полностью убрать
	bc.nextTickTime = bc.getTimeOfNextTick(time.Unix(0, int64(b.Timestamp)))
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.Hash[:], hash)
	bc.currentBock.B = &b

	if len(bc.chs.blocks) == 0 {
		bc.voteAppendValidator()
	}
	response <- resp
}

func (bc *Blockchain) voteAppendValidator() {
	fmt.Println("create append req")
	var data = make([]byte, INT_32_SIZE*2+HASH_SIZE+PKEY_SIZE)
	binary.LittleEndian.PutUint64(data[:INT_32_SIZE*2], bc.chainSize)
	copy(data[INT_32_SIZE*2:INT_32_SIZE*2+HASH_SIZE], bc.currentBock.Hash[:])
	copy(data[INT_32_SIZE*2+HASH_SIZE:], bc.thisValidator.Pkey[:])
	data = bc.thisKey.AppendSign(data)
	bc.appendVoting[bc.thisValidator] = 0
	go bc.network.SendVoteAppendValidatorMsgToAll(bc.activeHostsExceptMe, data)
}

func (bc *Blockchain) onAppendVote(data []byte, response chan ResponseMsg) {
	if len(data) == PKEY_SIZE*2+SIG_SIZE {
		bc.onAppendVoteValidator(data, response)
		return
	}
	if len(data) == INT_32_SIZE*2+HASH_SIZE+PKEY_SIZE+SIG_SIZE {
		bc.onAppendVoteViewer(data, response)
		return
	}
	response <- ResponseMsg{
		ok:    false,
		error: "incorrect msg length",
	}

}

func (bc *Blockchain) onAppendVoteViewer(data []byte, response chan ResponseMsg) {
	// по логике эта функция должна рассылать всем голос о том, что валидатор согласен принять вьювера в сеть
	var hash [HASH_SIZE]byte
	var pkey [PKEY_SIZE]byte
	var size uint64
	var sig [SIG_SIZE]byte
	size = binary.LittleEndian.Uint64(data[:INT_32_SIZE*2])
	copy(hash[:], data[INT_32_SIZE*2:INT_32_SIZE*2+HASH_SIZE])
	copy(pkey[:], data[INT_32_SIZE*2+HASH_SIZE:INT_32_SIZE*2+HASH_SIZE+PKEY_SIZE])
	copy(sig[:], data[INT_32_SIZE*2+HASH_SIZE+PKEY_SIZE:])
	sender, ok := bc.pkeyToValidator[pkey]
	if !ok || !VerifyData(data[:INT_32_SIZE*2+HASH_SIZE+PKEY_SIZE], sig[:], pkey) {
		response <- ResponseMsg{
			ok:    false,
			error: "unknown append vote sender",
		}
		return
	}
	if bc.currentBock != nil && bc.currentBock.Hash == hash && bc.chainSize == size {
		bc.appendVoting[sender] = 1
		bc.appendVotingMe[sender] = 1
	} else {
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect append validator data",
		}
		bc.appendVoting[sender] = 0
		bc.appendVotingMe[sender] = 0
		return
	}
	response <- ResponseMsg{ok: true}

}

func (bc *Blockchain) onAppendVoteValidator(data []byte, response chan ResponseMsg) {
	// эта функция мб никогда не вызовется
	var appendPkey [PKEY_SIZE]byte
	var senderPkey [PKEY_SIZE]byte
	var sig [SIG_SIZE]byte
	copy(appendPkey[:], data[:PKEY_SIZE])
	copy(senderPkey[:], data[PKEY_SIZE:PKEY_SIZE*2])
	copy(sig[:], data[PKEY_SIZE*2:])
	_, ok := bc.blockVoting[bc.pkeyToValidator[senderPkey]]
	if !ok || !VerifyData(data[:PKEY_SIZE*2], sig[:], senderPkey) {
		response <- ResponseMsg{
			ok:    false,
			error: "unknown append vote sender",
		}
		return
	}
	// после того, как придет первый голос, цикл в doAppendVoting перестанет посылать голоса
	// мб в посылать голос за добавление валидатора не в doAppendVoting, а в onAppendVoteViewer
	bc.appendVoting[bc.pkeyToValidator[appendPkey]] += 1
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
	sender := bc.pkeyToValidator[pkey]
	_, ok := bc.blockVoting[sender]
	if !ok || !VerifyData(data[:HASH_SIZE+PKEY_SIZE+1], sig[:], pkey) {
		response <- ResponseMsg{
			ok:    false,
			error: "unknown block vote sender",
		}
		return
	}

	if hash == bc.prevBlockHash && (vote[0] == 0x01 || vote[0] == 0x02) {
		bc.blockVoting[sender] = int(vote[0])
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
	_, ok := bc.kickVoting[bc.pkeyToValidator[senderPkey]]
	if !ok || !VerifyData(data[:PKEY_SIZE*2], sig[:], senderPkey) {
		response <- ResponseMsg{
			ok:    false,
			error: "unknown kick vote sender",
		}
		return
	}
	bc.kickVoting[bc.pkeyToValidator[kickPkey]] += 1
	response <- ResponseMsg{ok: true}
}

func (bc *Blockchain) updatePrevHashBlock() {
	for i := MAX_PREV_BLOCK_HASHES - 1; i >= 1; i-- {
		bc.prevBlockHashes[i] = bc.prevBlockHashes[i-1]
	}
	bc.prevBlockHashes[0] = bc.currentBock.Hash
	bc.prevBlockHash = bc.currentBock.Hash
}

func (bc *Blockchain) updateUnrecordedTrans() {
	var newUnrecorded []TransAndHash
	for _, t := range bc.unrecordedTrans {
		if !containsTransInBlock(bc.currentBock.B, t.Hash) {
			newUnrecorded = append(newUnrecorded, t)
		}
	}
	bc.unrecordedTrans = newUnrecorded
}

func (bc *Blockchain) processKick() {
	for k, v := range bc.kickVoting {
		if float32(v)/float32(len(bc.activeValidators)) > 0.5 {
			bc.activeValidators = removePkey(bc.activeValidators, k.Pkey)
			bc.activeHostsExceptMe = removeAddr(bc.activeHostsExceptMe, k.Addr)
		}
	}
	bc.kickVoting = make(map[*ValidatorNode]int)
	var clearedSuspiciousValidators = make(map[*ValidatorNode]int)
	for _, validator := range bc.activeValidators {
		bc.kickVoting[validator] = 0
		bc.blockVoting[validator] = 0
		clearedSuspiciousValidators[validator] = bc.suspiciousValidators[validator]
	}
	bc.suspiciousValidators = clearedSuspiciousValidators
}

// Функция doTick() поделена на отдельные этапы, между которыми было ожидание
// Теперь все эти этапы выполняются в основном потоке, а во время ожидания могу обрабатываться входящие блоки и транзы
func (bc *Blockchain) doTickPreparation() {
	bc.processKick()
	if bc.validatorStatus == VALIDATOR {
		for k, v := range bc.suspiciousValidators {
			if v > 0 {
				fmt.Println("suspicious validator", k, v)
			}
		}

		fmt.Println("process kick")

		bc.currentLeader = bc.activeValidators[bc.genBlocksCount%uint64(len(bc.activeValidators))]
	}
	fmt.Println(bc.chainSize)
	bc.nextTickTime = bc.getTimeOfNextTick(time.Now())
	bc.tickThisLeader <- true
}

func (bc *Blockchain) doTickThisLeader() {
	if bc.validatorStatus == VALIDATOR {
		if bc.thisValidator == bc.currentLeader {
			fmt.Println("this == leader")
			bc.expectBlocks = false
			bc.onThisCreateBlock()
		}
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
	if bc.validatorStatus == VALIDATOR {
		if len(bc.appendVoting) > 0 {
			bc.doAppendVoting()
		}
		fmt.Println("block voting time")
		voteData := make([]byte, HASH_SIZE+PKEY_SIZE+1)
		copy(voteData[:HASH_SIZE], bc.prevBlockHash[:])
		copy(voteData[HASH_SIZE:HASH_SIZE+PKEY_SIZE], bc.thisValidator.Pkey[:])
		var vote [1]byte
		if bc.currentBock != nil {
			vote[0] = 0x01
		} else {
			vote[0] = 0x02
		}
		copy(voteData[HASH_SIZE+PKEY_SIZE:HASH_SIZE+PKEY_SIZE+1], vote[:])
		voteData = bc.thisKey.AppendSign(voteData)
		go bc.network.SendVoteToAll(bc.activeHostsExceptMe, voteData)
	}
	var timeWhileVotesAreReceived = bc.nextTickTime.Add(-bc.justWaitingTime).Sub(time.Now())
	fmt.Println("sleeping for ", timeWhileVotesAreReceived)
	go func() {
		time.Sleep(timeWhileVotesAreReceived) // sleep from blockVotingTime
		bc.tickVotingProcessing <- true
	}()
}

func (bc *Blockchain) doTickVotingProcessing() {
	if len(bc.appendVoting) > 0 {
		bc.doAppendValidator()
	}
	if bc.validatorStatus == VALIDATOR {
		fmt.Println("process voting")
		yesVote, noVote := 0, 0
		for valid, vote := range bc.blockVoting {
			if vote == 0x01 {
				yesVote += 1
				bc.suspiciousValidators[valid] = 0
			} else if vote == 0x02 {
				noVote += 1
			} else if valid != bc.thisValidator {
				bc.suspiciousValidators[valid] += 1
			}
		}

		if noVote < yesVote {
			if bc.currentBock != nil {
				fmt.Println("block accepted")
				bc.updatePrevHashBlock()
				err := bc.db.SaveNextBlock(bc.currentBock)
				if err != nil {
					panic(err)
				}
				bc.updateUnrecordedTrans()
				bc.chainSize += 1
				bc.suspiciousValidators[bc.currentLeader] = 0
			}
		} else if bc.currentLeader != bc.thisValidator {
			fmt.Println("block rejected")
			bc.suspiciousValidators[bc.currentLeader] += 1
		}
		bc.genBlocksCount += 1
		fmt.Println("vote kick check")
		bc.tryKickValidator()
	} else {
		// get missing block надо делать после того, как блок был сгенерирован и сохранен всеми участниками сети
		// иначе возможно получить только максимум предыдущий блок
		// то есть разрыв в один принятый блок получается неустранимым
		// мб его нужно перенести в doTickPreparation
		bc.getMissingBlock(bc.activeValidators[(bc.chainSize+1)%uint64(len(bc.activeValidators))].Addr)
	}

	bc.currentBock = nil   // очищаем инфу о старом блоке, чтобы быть готовым принимать новые
	bc.expectBlocks = true // меняем флаг заранее, чтобы не пропустить блок
	// очистку blockVoting надо делать в момент, когда expectBlocks устанавливается в true
	// так как иначе, если лидер сгенерирует блок чуть раньше, можно затереть свой голос
	bc.blockVoting = make(map[*ValidatorNode]int)
	// с appendVoting аналогично, так как можно затереть отметку, котору ставит onAppendVoteViewer
	bc.appendVoting = make(map[*ValidatorNode]int)
	bc.appendVotingMe = make(map[*ValidatorNode]int)
	// вместе с обнулением блока необходимо обнулять и все хранилища голосов, привязанные к блоку (blockVoting, appendVoting)
	bc.currentLeader = bc.activeValidators[bc.genBlocksCount%uint64(len(bc.activeValidators))]
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
	b.CreateBlock(bc.unrecordedTrans[:transSize], bc.prevBlockHash, bc.thisKey, bc.chainSize)
	blockBytes := b.ToBytes()
	hash := b.HashBlock(blockBytes)
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.Hash[:], hash)
	bc.currentBock.B = &b
	bc.blockVoting[bc.thisValidator] = 0x01
	go bc.network.SendBlockToAll(bc.activeHostsExceptMe, blockBytes)
}

func (bc *Blockchain) tryKickValidator() {
	for valid, v := range bc.suspiciousValidators {
		if v > 1 {
			bc.kickVoting[valid] += 1
			data := make([]byte, PKEY_SIZE*2)
			copy(data[:PKEY_SIZE], valid.Pkey[:])
			copy(data[PKEY_SIZE:PKEY_SIZE*2], bc.thisValidator.Pkey[:])
			data = bc.thisKey.AppendSign(data)
			go bc.network.SendKickMsgToAll(bc.activeHostsExceptMe, data)
		}
	}
}

func (bc *Blockchain) doAppendValidator() {
	fmt.Println("do append new validator")
	var activeLen = float32(len(bc.activeValidators))
	for valid, val := range bc.appendVoting {
		fmt.Println(val)
		if float32(val)/activeLen > 0.5 {
			fmt.Println(valid.Addr + " is new validator")
			bc.activeValidators = appendValidator(bc.activeValidators,
				bc.allValidators, valid)
			bc.activeHostsExceptMe = remakeActiveHostsExceptMe(bc.activeHostsExceptMe,
				bc.activeValidators, bc.thisValidator)
			if bc.thisValidator.Pkey == valid.Pkey {
				bc.validatorStatus = VALIDATOR
			}
			if bc.validatorStatus == VALIDATOR {
				bc.suspiciousValidators[valid] = 0
				bc.genBlocksCount = bc.chainSize
			}
			bc.blockVoting[valid] = 1
			bc.kickVoting[valid] = 0
		}
	}
}

func (bc *Blockchain) doAppendVoting() {
	for valid, val := range bc.appendVotingMe {
		if val == 1 {
			data := make([]byte, PKEY_SIZE*2)
			copy(data[:PKEY_SIZE], valid.Pkey[:])
			copy(data[PKEY_SIZE:PKEY_SIZE*2], bc.thisValidator.Pkey[:])
			data = bc.thisKey.AppendSign(data)
			go bc.network.SendVoteAppendValidatorMsgToAll(bc.activeHostsExceptMe, data)
		}
	}
}

//блок функций подготовки валидатора к генерации блоков
//----------------------------------------------------
//методы отправителей

func (bc *Blockchain) prepare() {
	hosts := hostsExceptGiven(bc.allValidators, bc.thisValidator.Pkey)
	hosts = bc.network.PingHosts(hosts)
	if len(hosts) == 0 {
		bc.activeValidators = append(bc.activeValidators, bc.thisValidator)
		hosts = make([]string, 0)
		bc.validatorStatus = VALIDATOR
		bc.suspiciousValidators[bc.thisValidator] = 0
		return
	}
	i := 0
	addr := hosts[i]
	for {
		oldSize := bc.chainSize
		ok := bc.getMissingBlock(addr)
		if ok && oldSize == bc.chainSize {
			break
		}
		if !ok {
			i++
			if i == len(hosts) {
				i = 0
			}
			addr = hosts[i]
		}
	}
	bc.sendAppendMsg(hosts)
	bc.validatorStatus = VIEWER
	if bc.nextTickTime.Sub(time.Now()) < 0 {
		// все равно nextTickTime может быть меньше текущего времени, если запрос попал в промежуток
		// между созданием нового блока и окончанием голосования за его принятие
		bc.getMissingBlock(addr)
	}
	sleepTime := bc.nextTickTime.Sub(time.Now())
	time.Sleep(sleepTime)
}

func (bc *Blockchain) sendAppendMsg(hosts []string) {
	data := bc.thisKey.AppendSign(bc.thisValidator.Pkey[:])
	bc.activeValidators = make([]*ValidatorNode, 0)
	for _, h := range hosts {
		resp, err := bc.network.SendAppendViewerMsg(h, data)
		if err == nil && len(resp) == PKEY_SIZE+SIG_SIZE {
			var pkey [PKEY_SIZE]byte
			var sig [SIG_SIZE]byte
			copy(pkey[:], resp[:PKEY_SIZE])
			copy(sig[:], resp[PKEY_SIZE:])
			_, ok := bc.pkeyToValidator[pkey]
			if ok && VerifyData(resp[:PKEY_SIZE], sig[:], pkey) {
				valid := bc.pkeyToValidator[pkey]
				bc.activeValidators = appendValidator(bc.activeValidators,
					bc.allValidators, valid)
				bc.activeHostsExceptMe = append(bc.activeHostsExceptMe, valid.Addr)
				bc.suspiciousValidators[valid] = 0
			}
		}
	}
}

func (bc *Blockchain) getMissingBlock(host string) bool {
	data, err := bc.network.GetBlockAfter(host, bc.prevBlockHash)
	if err == nil {
		//data[0] == 0xFF означает, что нода имеет все блоки,
		//которые есть у отправителя в БД
		if len(data) == 1 && data[0] == 0xFF {
			return true
		}
		//0x00 означает, что проверка на creator не происходит
		//т.к. creator[0] в нормальном случае может быть равен только 0x02 или 0x03
		var creator [PKEY_SIZE]byte
		var b Block
		hash, blockLen := b.Verify(data, bc.prevBlockHash, creator, bc.db)
		fmt.Println("block len", blockLen)
		if blockLen != len(data) {
			return false
		}
		bc.currentBock = new(BlocAndkHash)
		copy(bc.currentBock.Hash[:], hash)
		bc.currentBock.B = &b
		bc.updatePrevHashBlock()
		err := bc.db.SaveNextBlock(bc.currentBock)
		if err != nil {
			panic(err)
		}
		bc.chainSize += 1
		fmt.Println("current chain size is ", bc.chainSize)
		// Это одни из предыдущих блоко, в nextTickTime будет время предыдущего (уже прошедшего) тика
		bc.nextTickTime = bc.getTimeOfNextTick(time.Unix(0, int64(b.Timestamp)))
		return true
	}
	return false
}

//--------------------------------------------

//методы для получателей
func (bc *Blockchain) onAppendViewer(data []byte, response chan ByteResponse) {
	if bc.validatorStatus != VALIDATOR {
		response <- ByteResponse{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}
	if len(data) != PKEY_SIZE+SIG_SIZE {
		response <- ByteResponse{
			ok:    false,
			error: "err Pkey",
		}
		return
	}
	var pkey [PKEY_SIZE]byte
	var sig [SIG_SIZE]byte
	copy(pkey[:], data[:PKEY_SIZE])
	copy(sig[:], data[PKEY_SIZE:])
	_, ok := bc.pkeyToValidator[pkey]
	if !ok || !VerifyData(data[:PKEY_SIZE], sig[:], pkey) {
		response <- ByteResponse{
			ok:    false,
			error: "err Pkey",
		}
		return
	}

	respData := bc.thisKey.AppendSign(bc.thisValidator.Pkey[:])
	response <- ByteResponse{
		ok:   true,
		data: respData,
	}
	bc.activeHostsExceptMe = append(bc.activeHostsExceptMe, bc.pkeyToValidator[pkey].Addr)
}

func (bc *Blockchain) onGetBlockAfter(data []byte, response chan ByteResponse) {
	if bc.validatorStatus != VALIDATOR {
		response <- ByteResponse{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}
	var hash [HASH_SIZE]byte
	copy(hash[:], data)
	//ответ когда this.bc.prevHash == data
	if bc.prevBlockHash == hash {
		var oneByte = [1]byte{0xFF}
		response <- ByteResponse{
			ok:   true,
			data: oneByte[:],
		}
		return
	}
	b, err := bc.db.GetBlockAfter(hash)
	if err != nil {
		panic(err)
	}
	if b == nil {
		response <- ByteResponse{
			ok:    false,
			error: "no such block",
		}
		return
	}
	// ответ с блоком
	blockBytes := b.B.ToBytes()
	response <- ByteResponse{
		ok:   true,
		data: blockBytes,
	}

}

func (bc *Blockchain) onGetTxsByHashes(data []byte, response chan ByteResponse) {
	if bc.validatorStatus != VALIDATOR {
		response <- ByteResponse{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}
	if len(data) < INT_32_SIZE {
		response <- ByteResponse{
			ok:    false,
			error: "no hashes sent",
		}
		return
	}
	hashesNum := binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	if int(hashesNum)*HASH_SIZE+INT_32_SIZE != len(data) {
		response <- ByteResponse{
			ok:    false,
			error: "incorrect hashes length",
		}
		return
	}
	offset := INT_32_SIZE
	hashes := make([][HASH_SIZE]byte, 0)
	for offset+HASH_SIZE <= len(data) {
		hash := [HASH_SIZE]byte{}
		copy(hash[:], data[offset:offset+HASH_SIZE])
		hashes = append(hashes, hash)
		offset += HASH_SIZE
	}
	txs, err := bc.db.GetTxsByHashes(hashes)
	if err != nil {
		response <- ByteResponse{
			ok:    false,
			error: "db error" + err.Error(),
		}
		return
	}
	txsPacked := make([]byte, INT_32_SIZE)
	binary.LittleEndian.PutUint32(txsPacked, uint32(len(txs)))
	for _, tx := range txs {
		txsPacked = append(txsPacked, tx.Transaction.ToBytes()...)
	}
	response <- ByteResponse{
		ok:   true,
		data: txsPacked,
	}
	return
}

func (bc *Blockchain) onGetTxsByPkey(data []byte, response chan ByteResponse) {
	if bc.validatorStatus != VALIDATOR {
		response <- ByteResponse{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}
	if len(data) != PKEY_SIZE {
		response <- ByteResponse{
			ok:    false,
			error: "incorrect pkey size",
		}
		return
	}
	pkey := [PKEY_SIZE]byte{}
	copy(pkey[:], data)
	txs, err := bc.db.GetTxsByPubKey(pkey)
	if err != nil {
		response <- ByteResponse{
			ok:    false,
			error: "db error" + err.Error(),
		}
		return
	}
	txsPacked := make([]byte, INT_32_SIZE)
	binary.LittleEndian.PutUint32(txsPacked, uint32(len(txs)))
	for _, tx := range txs {
		txsPacked = append(txsPacked, tx.Transaction.ToBytes()...)
	}
	response <- ByteResponse{
		ok:   true,
		data: txsPacked,
	}
	return
}

func (bc *Blockchain) onGetUtxosByPkey(data []byte, response chan ByteResponse) {
	if bc.validatorStatus != VALIDATOR {
		response <- ByteResponse{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}
	if len(data) != PKEY_SIZE {
		response <- ByteResponse{
			ok:    false,
			error: "incorrect pkey size",
		}
		return
	}
	pkey := [PKEY_SIZE]byte{}
	copy(pkey[:], data)
	utxos, err := bc.db.GetUTXOSByPkey(pkey)
	if err != nil {
		response <- ByteResponse{
			ok:    false,
			error: "db error" + err.Error(),
		}
		return
	}
	utxosPacked := make([]byte, INT_32_SIZE)
	binary.LittleEndian.PutUint32(utxosPacked, uint32(len(utxos)))
	for _, utxo := range utxos {
		utxosPacked = append(utxosPacked, utxo.ToBytes()...)
	}
	response <- ByteResponse{
		ok:   true,
		data: utxosPacked,
	}
}

func (bc *Blockchain) onGetMoneyRequest(data []byte, response chan ResponseMsg) {
	if bc.validatorStatus != VALIDATOR {
		response <- ResponseMsg{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}
	if len(data) != INT_32_SIZE+PKEY_SIZE {
		response <- ResponseMsg{
			ok:    false,
			error: "incorrect msg len",
		}
		return
	}

	var pkey [PKEY_SIZE]byte
	var amount = binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	copy(pkey[:], data[INT_32_SIZE:])
	utxos, err := bc.db.GetUTXOSByPkey(bc.thisValidator.Pkey)
	if err != nil {
		response <- ResponseMsg{
			ok:    false,
			error: "db error: " + err.Error(),
		}
		return
	}
	var t Transaction
	var outputs = make(map[[PKEY_SIZE]byte]uint32, 0)
	outputs[pkey] = amount
	errCreate := t.CreateTrans(utxos, outputs, ZERO_ARRAY_HASH, bc.thisKey, 0, 0, false)
	if errCreate != OK {
		response <- ResponseMsg{
			ok:    false,
			error: "trans create err",
		}
		return
	}
	transBytes := t.ToBytes()
	hash, transLen := t.Verify(transBytes, bc.db)
	if transLen < 0 {
		response <- ResponseMsg{
			ok:    false,
			error: "trans create-verify err",
		}
		return
	}
	bc.appendUnrecordedTrans(&t, hash)
	go bc.network.SendTxToAll(bc.activeHostsExceptMe, transBytes)
	response <- ResponseMsg{
		ok: true,
	}
}

func (bc *Blockchain) onGetVoteResult(data []byte, response chan ByteResponse) {
	if bc.validatorStatus != VALIDATOR {
		response <- ByteResponse{
			ok:    false,
			error: "i'm not validator",
		}
		return
	}

	if len(data) != HASH_SIZE {
		response <- ByteResponse{
			ok:    false,
			error: "incorrect msg len",
		}
		return
	}
	var mainHash [HASH_SIZE]byte
	copy(mainHash[:], data[:])
	t, timeStart, err := bc.db.GetTxAndTimeByHash(mainHash)
	if err != nil {
		response <- ByteResponse{
			ok:    false,
			error: "db error: " + err.Error(),
		}
		return
	}
	endTime := timeStart + uint64(time.Second)*uint64(t.Transaction.Duration)
	utxos, err := bc.db.GetUTXOSByTypeValue(mainHash)
	if err != nil {
		response <- ByteResponse{
			ok:    false,
			error: "db error: " + err.Error(),
		}
		return
	}

	//блок подсчета голосов надо разбить на разные функции, так как
	//в при некоторых случаях один и тот же избиратель может голосовать дважды,
	//может голосовать "за", "против", "воздержался" и т.п.
	//так же может происходит сортировка результатов гослования в зависимости от его типа
	result := make(map[[PKEY_SIZE]byte]int32, 0)
	for _, utxo := range utxos {
		_, contains := result[utxo.PkeyTo]
		if utxo.TypeValue == t.Hash && utxo.Timestamp < endTime {
			if contains {
				result[utxo.PkeyTo] += getVoteValue(utxo.Value, t.Transaction.TypeVote)
			} else {
				result[utxo.PkeyTo] = getVoteValue(utxo.Value, t.Transaction.TypeVote)
			}
		}
	}

	for {
		for _, out := range t.Transaction.Outputs {
			_, contains := result[out.PkeyTo]
			if contains {
				delete(result, out.PkeyTo)
			}
		}
		t, err = bc.db.GetTxByHashLink(t.Hash)
		if err != nil {
			response <- ByteResponse{
				ok:    false,
				error: "db error: " + err.Error(),
			}
			return
		}
		if t == nil {
			break
		}
	}

	//create bytes result
	var resBytes []byte
	var valBytes [INT_32_SIZE]byte
	fmt.Println("result voting with mainHash: ", mainHash)
	for pkey, val := range result {
		fmt.Println(pkey, val)
		binary.LittleEndian.PutUint32(valBytes[:], uint32(val))
		resBytes = append(resBytes, pkey[:]...)
		resBytes = append(resBytes, valBytes[:]...)
	}
	fmt.Println()

	response <- ByteResponse{
		ok:   true,
		data: resBytes,
	}
}
