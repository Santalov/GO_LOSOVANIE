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
	pkey [PKEY_SIZE]byte
	addr string // адрес вида 1.1.1.1:1337
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
		if bc.thisKey.pubKeyByte == v.pkey {
			bc.thisValidator = v
		}
		bc.pkeyToValidator[v.pkey] = v
		bc.addrToValidator[v.addr] = v
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
	go bc.network.Serve(bc.thisValidator.addr) // запускаем сеть в отдельной горутине, не блокируем текущий поток
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
			fmt.Println("transaction from validator", msg)
		case msg := <-bc.chs.txsClient:
			// транза от приложения-клиента
			fmt.Println("transaction from client", msg)
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
		}
	}
}

func (bc *Blockchain) getTimeOfNextTick(lastBlockTime time.Time) time.Time {
	return lastBlockTime.Add(bc.blockAppendTime).Add(bc.blockVotingTime).Add(bc.justWaitingTime)
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
	hash, blockLen := b.Verify(data, bc.prevBlockHash, bc.currentLeader.pkey, bc.db)
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

	bc.nextTickTime = bc.getTimeOfNextTick(time.Unix(0, int64(b.timestamp)))
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

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
	bc.nextTickTime = bc.getTimeOfNextTick(time.Unix(0, int64(b.timestamp)))

	if blockLen != len(data) {
		addr := bc.activeValidators[(bc.chainSize+1)%uint64(len(bc.activeValidators))].addr
		// опасно начинать общаться с другими участниками сети до отсылки ответа текущему
		// т.е опасно делать response <- resp после bc.getMissingBlock(addr)
		bc.getMissingBlock(addr)
		response <- resp
		return
	}

	// установку nextTimeTick возможно надо перенести наверх, до вызова bc.getMissingBlock
	// а из getMissingBlock установку nextTickTime полностью убрать
	bc.currentBock = new(BlocAndkHash)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	if len(bc.chs.blocks) == 0 {
		bc.voteAppendValidator()
	}
	response <- resp
}

func (bc *Blockchain) voteAppendValidator() {
	fmt.Println("create append req")
	var data = make([]byte, INT_32_SIZE*2+HASH_SIZE+PKEY_SIZE)
	binary.LittleEndian.PutUint64(data[:INT_32_SIZE*2], bc.chainSize)
	copy(data[INT_32_SIZE*2:INT_32_SIZE*2+HASH_SIZE], bc.currentBock.hash[:])
	copy(data[INT_32_SIZE*2+HASH_SIZE:], bc.thisValidator.pkey[:])
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
	if bc.currentBock != nil && bc.currentBock.hash == hash && bc.chainSize == size {
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
	bc.prevBlockHashes[0] = bc.currentBock.hash
	bc.prevBlockHash = bc.currentBock.hash
}

func (bc *Blockchain) updateUnrecordedTrans() {
	var newUnrecorded []TransAndHash
	for _, t := range bc.unrecordedTrans {
		if !containsTransInBlock(bc.currentBock.b, t.hash) {
			newUnrecorded = append(newUnrecorded, t)
		}
	}
}

func (bc *Blockchain) processKick() {
	for k, v := range bc.kickVoting {
		if float32(v)/float32(len(bc.activeValidators)) > 0.5 {
			bc.activeValidators = removePkey(bc.activeValidators, k.pkey)
			bc.activeHostsExceptMe = removeAddr(bc.activeHostsExceptMe, k.addr)
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
	} else {
		bc.getMissingBlock(bc.activeValidators[(bc.chainSize+1)%uint64(len(bc.activeValidators))].addr)
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
		copy(voteData[HASH_SIZE:HASH_SIZE+PKEY_SIZE], bc.thisValidator.pkey[:])
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
		bc.getMissingBlock(bc.activeValidators[(bc.chainSize+1)%uint64(len(bc.activeValidators))].addr)
	}

	bc.currentBock = nil   // очищаем инфу о старом блоке, чтобы быть готовым принимать новые
	bc.expectBlocks = true // меняем флаг заранее, чтобы не пропустить блок
	// вместе с обнулением блока необходимо обнулять и все хранилища голосов, привязанные к блоку (blockVoting, appendVoting)
	bc.blockVoting = make(map[*ValidatorNode]int)
	bc.appendVoting = make(map[*ValidatorNode]int)
	bc.appendVotingMe = make(map[*ValidatorNode]int)
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
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b
	bc.blockVoting[bc.thisValidator] = 0x01
	go bc.network.SendBlockToAll(bc.activeHostsExceptMe, blockBytes)
}

func (bc *Blockchain) tryKickValidator() {
	for valid, v := range bc.suspiciousValidators {
		if v > 1 {
			bc.kickVoting[valid] += 1
			data := make([]byte, PKEY_SIZE*2)
			copy(data[:PKEY_SIZE], valid.pkey[:])
			copy(data[PKEY_SIZE:PKEY_SIZE*2], bc.thisValidator.pkey[:])
			data = bc.thisKey.AppendSign(data)
			go bc.network.SendKickMsgToAll(bc.activeHostsExceptMe, data)
		}
	}
}

func (bc *Blockchain) doAppendValidator() {
	fmt.Println("do append new validator")
	for valid, val := range bc.appendVoting {
		fmt.Println(val)
		if float32(val)/float32(len(bc.activeValidators)) > 0.5 {
			fmt.Println(valid.addr + " is new validator")
			bc.activeValidators = appendValidator(bc.activeValidators,
				bc.allValidators, valid)
			bc.activeHostsExceptMe = remakeActiveHostsExceptMe(bc.activeHostsExceptMe,
				bc.activeValidators, bc.thisValidator)
			if bc.thisValidator.pkey == valid.pkey {
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
			copy(data[:PKEY_SIZE], valid.pkey[:])
			copy(data[PKEY_SIZE:PKEY_SIZE*2], bc.thisValidator.pkey[:])
			data = bc.thisKey.AppendSign(data)
			go bc.network.SendVoteAppendValidatorMsgToAll(bc.activeHostsExceptMe, data)
		}
	}
}

//блок функций подготовки валидатора к генерации блоков
//----------------------------------------------------
//методы отправителей

func (bc *Blockchain) prepare() {
	hosts := hostsExceptGiven(bc.allValidators, bc.thisValidator.pkey)
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
	data := bc.thisKey.AppendSign(bc.thisValidator.pkey[:])
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
				bc.activeHostsExceptMe = append(bc.activeHostsExceptMe, valid.addr)
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
		copy(bc.currentBock.hash[:], hash)
		bc.currentBock.b = &b
		bc.updatePrevHashBlock()
		err := bc.db.SaveNextBlock(bc.currentBock)
		if err != nil {
			panic(err)
		}
		bc.chainSize += 1
		fmt.Println("current chain size is ", bc.chainSize)
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
			error: "err pkey",
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
			error: "err pkey",
		}
		return
	}

	respData := bc.thisKey.AppendSign(bc.thisValidator.pkey[:])
	response <- ByteResponse{
		ok:   true,
		data: respData,
	}
	bc.activeHostsExceptMe = append(bc.activeHostsExceptMe, bc.pkeyToValidator[pkey].addr)
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
	blockBytes := b.b.ToBytes()
	response <- ByteResponse{
		ok:   true,
		data: blockBytes,
	}

}
