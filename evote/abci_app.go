package evote

import (
	"fmt"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

type ValidatorNode struct {
	Pkey           [PKEY_SIZE]byte
	IpAndPort      string // адрес вида 1.1.1.1:1337
	TendermintAddr [TM_ADDR_SIZE]byte
}

type BlockchainApp struct {
	db *Database
	nw *Network // not thread safe
	// validator key for sending and receiving transactions is different from consensus key
	thisKey       *CryptoKeysData // key for sending transactions
	validators    []*ValidatorNode
	thisValidator *ValidatorNode // contains key for consensus
	//map для удобного получения валидаторов
	addrToValidator           map[string]*ValidatorNode
	pkeyToValidator           map[[PKEY_SIZE]byte]*ValidatorNode
	tendermintAddrToValidator map[[TM_ADDR_SIZE]byte]*ValidatorNode // map form consensus keys into validators
	appBlockHash              [HASH_SIZE]byte                       // hash of the last committed block
	appHeight                 int64                                 // number of the last committed block
	checkTxState              *TxExecutor                           // TODO: move transaction execution logic into separate struct
	deliverTxState            *TxExecutor

	version    string
	appVersion uint64
}

var _ abcitypes.Application = (*BlockchainApp)(nil)

func NewBlockchainApp(
	thisPrv []byte,
	validators []*ValidatorNode,
	dbPort int,
	version string, // application software semantic version
	appVersion uint64, // application protocol version, included in every block
) *BlockchainApp {
	bc := &BlockchainApp{}
	bc.setup(thisPrv, validators, dbPort, version, appVersion)
	return bc
}

func (bc *BlockchainApp) setup(
	thisPrv []byte,
	validators []*ValidatorNode,
	dbPort int,
	version string,
	appVersion uint64,
) {
	var k CryptoKeysData
	k.SetupKeys(thisPrv)
	bc.thisKey = &k

	bc.addrToValidator = make(map[string]*ValidatorNode)
	bc.pkeyToValidator = make(map[[PKEY_SIZE]byte]*ValidatorNode)
	bc.tendermintAddrToValidator = make(map[[TM_ADDR_SIZE]byte]*ValidatorNode)
	bc.validators = validators
	bc.version = version
	bc.appVersion = appVersion

	for _, v := range bc.validators {
		if bc.thisKey.PubkeyByte == v.Pkey {
			bc.thisValidator = v
		}
		bc.pkeyToValidator[v.Pkey] = v
		bc.addrToValidator[v.IpAndPort] = v
		bc.tendermintAddrToValidator[v.TendermintAddr] = v
	}
	if bc.thisValidator == nil {
		panic(fmt.Errorf("no validator with pkey %v", bc.thisKey.PubkeyByte))
	}

	//load prev from DB
	bc.appBlockHash = ZERO_ARRAY_HASH

	bc.appHeight = 0

	bc.db = new(Database)
	err := bc.db.Init(DBNAME, DBUSER, DBPASSWORD, DBHOST, dbPort)
	if err != nil {
		panic(err)
	}
	bc.checkTxState = NewTxExecutor(bc.db)
	bc.deliverTxState = NewTxExecutor(bc.db)
}

func (bc *BlockchainApp) initNetwork() {
	bc.nw = new(Network)
	var allHosts []string
	// including self into available hosts is dangerous, but cannot be avoided, cos we need to work even if we are alone
	for _, v := range bc.validators {
		allHosts = append(allHosts, v.IpAndPort)
	}
	bc.nw.Init(allHosts)
}

func (bc *BlockchainApp) getValidator(tendermintAddr []byte) *ValidatorNode {
	var tmAddr [TM_ADDR_SIZE]byte
	copy(tmAddr[:], tendermintAddr)
	return bc.tendermintAddrToValidator[tmAddr]
}

func (bc *BlockchainApp) SetOption(option abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{}
}

func (bc *BlockchainApp) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{
		Data:             "e-voting blockchain validator",
		Version:          bc.version,
		AppVersion:       bc.appVersion,
		LastBlockHeight:  bc.appHeight,
		LastBlockAppHash: bc.appBlockHash[:],
	}
}

func (bc *BlockchainApp) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	code := bc.checkTxState.AppendTx(req.Tx, true)
	return abcitypes.ResponseCheckTx{
		Code: code,
	}
}

func (bc *BlockchainApp) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	fmt.Println("begin block", req.Hash)
	proposer := bc.getValidator(req.Header.ProposerAddress)
	bc.deliverTxState.BeginBlock(req.Header.Time, proposer.Pkey)
	return abcitypes.ResponseBeginBlock{}
}

func (bc *BlockchainApp) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	fmt.Println("deliver tx")
	code := bc.deliverTxState.AppendTx(req.Tx, false)
	return abcitypes.ResponseDeliverTx{
		Code: code,
	}
}

func (bc *BlockchainApp) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	// no changes to validator set during runtime
	fmt.Println("end block")
	return abcitypes.ResponseEndBlock{}
}

// function blocks thread
func (bc *BlockchainApp) broadcastTxUntilSuccess(tx []byte) {
	maxErrors := len(bc.validators)
	errors := 0
	// Network has internal state, so for thread safety copy it
	nw := bc.nw.Copy()
	for {
		resp, err := nw.BroadcastTxSync(tx)
		if err == nil && resp.Error == nil {
			fmt.Println("reward tx broadcast success, result", resp.Result)
			return
		}
		nw.SelectNextHost()
		errors++
		if errors >= maxErrors {
			panic(
				fmt.Sprintf(
					"impossible to broadcast reward tx, err %v, rpc err %v\n",
					err.Error(),
					resp.Error.Error(),
				),
			)
		}
	}
}

// function blocks thread
func (bc *BlockchainApp) broadcastRewardForMe(blockHash [HASH_SIZE]byte) {
	var t Transaction
	t.CreateMiningReward(bc.thisKey, blockHash)
	bc.broadcastTxUntilSuccess(t.ToBytes())
}

func (bc *BlockchainApp) Commit() abcitypes.ResponseCommit {
	fmt.Println("commit")
	var b Block
	b.CreateBlock(
		bc.deliverTxState.Transactions,
		bc.appBlockHash,
		bc.deliverTxState.Timestamp,
		bc.deliverTxState.BlockProposer,
	)
	var blockAndHash BlocAndkHash
	blockAndHash.B = &b
	hash := b.HashBlock(b.ToBytes())
	copy(blockAndHash.Hash[:], hash)
	if bc.deliverTxState.BlockProposer == bc.thisValidator.Pkey {
		// this validator is proposer of the block, reward tx will be added in some of the next blocks
		go bc.broadcastRewardForMe(blockAndHash.Hash)
	}
	err := bc.db.SaveNextBlock(&blockAndHash)
	if err != nil {
		panic(err)
	}
	bc.appBlockHash = blockAndHash.Hash
	bc.appHeight++
	bc.checkTxState.Reset()
	bc.deliverTxState.Reset()
	fmt.Println("block committed", blockAndHash.Hash)
	return abcitypes.ResponseCommit{
		Data: bc.appBlockHash[:],
	}
}

func (bc *BlockchainApp) ListSnapshots(req abcitypes.RequestListSnapshots) abcitypes.ResponseListSnapshots {
	return abcitypes.ResponseListSnapshots{}
}

func (bc *BlockchainApp) OfferSnapshot(snapshot abcitypes.RequestOfferSnapshot) abcitypes.ResponseOfferSnapshot {
	return abcitypes.ResponseOfferSnapshot{}
}

func (bc *BlockchainApp) LoadSnapshotChunk(req abcitypes.RequestLoadSnapshotChunk) abcitypes.ResponseLoadSnapshotChunk {
	return abcitypes.ResponseLoadSnapshotChunk{}
}

func (bc *BlockchainApp) ApplySnapshotChunk(chunk abcitypes.RequestApplySnapshotChunk) abcitypes.ResponseApplySnapshotChunk {
	return abcitypes.ResponseApplySnapshotChunk{}
}

func (bc *BlockchainApp) Query(reqQuery abcitypes.RequestQuery) abcitypes.ResponseQuery {
	fmt.Println("query", reqQuery.Path, reqQuery.Data)
	return abcitypes.ResponseQuery{}
}

func (bc *BlockchainApp) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	bc.appHeight = req.InitialHeight
	fmt.Println("init chain, appStateBytes", req.AppStateBytes)
	go bc.initNetwork() // init in background, to not to block response
	return abcitypes.ResponseInitChain{
		AppHash: bc.appBlockHash[:],
	}
}
