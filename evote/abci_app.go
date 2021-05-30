package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

type ValidatorNode struct {
	Pkey           [PkeySize]byte
	IpAndPort      string // адрес вида 1.1.1.1:1337
	TendermintAddr [TmAddrSize]byte
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
	pkeyToValidator           map[[PkeySize]byte]*ValidatorNode
	tendermintAddrToValidator map[[TmAddrSize]byte]*ValidatorNode // map form consensus keys into validators
	appBlockHash              []byte                              // hash of the last committed block
	appHeight                 int64                               // number of the last committed block
	checkTxState              *TxExecutor                         // TODO: move transaction execution logic into separate struct
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
	bc.pkeyToValidator = make(map[[PkeySize]byte]*ValidatorNode)
	bc.tendermintAddrToValidator = make(map[[TmAddrSize]byte]*ValidatorNode)
	bc.validators = validators
	bc.version = version
	bc.appVersion = appVersion

	for _, v := range bc.validators {
		if bc.thisKey.PkeyByte == v.Pkey {
			bc.thisValidator = v
		}
		bc.pkeyToValidator[v.Pkey] = v
		bc.addrToValidator[v.IpAndPort] = v
		bc.tendermintAddrToValidator[v.TendermintAddr] = v
	}
	if bc.thisValidator == nil {
		panic(fmt.Errorf("no validator with pkey %v", bc.thisKey.PkeyByte))
	}

	//load prev from DB
	bc.appBlockHash = nil

	bc.appHeight = 0

	bc.db = new(Database)
	err := bc.db.Init(DbName, DbUser, DbPassword, DbHost, dbPort)
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
	var tmAddr [TmAddrSize]byte
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
	//fmt.Println("begin block", req.Hash)
	proposer := bc.getValidator(req.Header.ProposerAddress)
	bc.deliverTxState.BeginBlock(req.Header.Time, proposer.Pkey)
	return abcitypes.ResponseBeginBlock{}
}

func (bc *BlockchainApp) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	//fmt.Println("deliver tx")
	code := bc.deliverTxState.AppendTx(req.Tx, false)
	return abcitypes.ResponseDeliverTx{
		Code: code,
	}
}

func (bc *BlockchainApp) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	// no changes to validator set during runtime
	//fmt.Println("end block")
	return abcitypes.ResponseEndBlock{}
}

// BroadcastTxUntilSuccess function blocks thread, until success or error broadcast
func BroadcastTxUntilSuccess(nw *Network, tx []byte) {
	maxErrors := len(nw.allHosts)
	errors := 0
	// Network has internal state, so for thread safety copy it
	for {
		_, err := nw.BroadcastTxSync(tx)
		if err == nil {
			//fmt.Println("reward tx broadcast success")
			return
		}
		nw.SelectNextHost()
		errors++
		if errors >= maxErrors {
			panic(
				fmt.Sprintf(
					"impossible to broadcast reward tx, err %v",
					err.Error(),
				),
			)
		}
	}
}

// function blocks thread
func (bc *BlockchainApp) broadcastRewardForMe(blockHash []byte) {
	t, err := CreateMiningReward(bc.thisKey, blockHash)
	if err != nil {
		panic(err)
	}
	txBytes, err := proto.Marshal(t)
	if err != nil {
		panic(err)
	}
	BroadcastTxUntilSuccess(bc.nw.Copy(), txBytes)
}

func (bc *BlockchainApp) Commit() abcitypes.ResponseCommit {
	//fmt.Println("commit")
	b, err := CreateBlock(
		bc.deliverTxState.Transactions,
		bc.appBlockHash,
		bc.deliverTxState.Timestamp,
		bc.deliverTxState.BlockProposer,
	)
	if err != nil {
		panic(err)
	}
	if bc.deliverTxState.BlockProposer == bc.thisValidator.Pkey {
		// this validator is proposer of the block, reward tx will be added in some of the next blocks
		go bc.broadcastRewardForMe(b.Hash)
	}
	err = bc.db.SaveNextBlock(b)
	if err != nil {
		panic(err)
	}
	bc.appBlockHash = b.Hash
	bc.appHeight++
	bc.checkTxState.Reset()
	bc.deliverTxState.Reset()
	fmt.Println("block committed", hex.EncodeToString(b.Hash), "txCount", len(b.Transactions))
	return abcitypes.ResponseCommit{
		Data: bc.appBlockHash,
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

func respondAbciQuery(code uint32, err error, resp *golosovaniepb.Response) abcitypes.ResponseQuery {
	if err != nil {
		return abcitypes.ResponseQuery{
			Code: code,
			Log:  err.Error(),
		}
	} else {
		value, err := proto.Marshal(resp)
		if err != nil {
			return abcitypes.ResponseQuery{
				Code: CodeSerializeErr,
				Log:  "error during final serialization of the response",
			}
		}
		return abcitypes.ResponseQuery{
			Code:  code,
			Value: value,
		}
	}
}

func (bc *BlockchainApp) Query(reqQuery abcitypes.RequestQuery) abcitypes.ResponseQuery {
	fmt.Println("query", reqQuery.Path, reqQuery.Data)
	var req golosovaniepb.Request
	err := proto.Unmarshal(reqQuery.Data, &req)
	if err != nil {
		return respondAbciQuery(
			CodeParseErr,
			err,
			nil,
		)
	}
	switch reqQuery.Path {
	case "getTxs":
		return respondAbciQuery(
			OnGetTxsByHashes(bc.db, req.GetTxsByHashes()),
		)
	case "getTxsByPubKey":
		return respondAbciQuery(
			OnGetTxsByPkey(bc.db, req.GetTxsByPkey()),
		)
	case "getUtxosByPubKey":
		return respondAbciQuery(
			OnGetUtxosByPkey(bc.db, req.GetUtxosByPkey()),
		)
	case "faucet":
		return respondAbciQuery(
			OnFaucet(bc.db, bc.nw, bc.thisKey, req.GetFaucet()),
		)
	case "getVoteResult":
		return respondAbciQuery(
			OnGetVoteResult(bc.db, req.GetVoteResult()),
		)
	}

	return abcitypes.ResponseQuery{
		Code: CodeUnknownPath,
		Log:  "no such path, check in request is correct",
	}
}

func (bc *BlockchainApp) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	bc.appHeight = req.InitialHeight
	fmt.Println("init chain, appStateBytes", req.AppStateBytes)
	go bc.initNetwork() // init in background, to not to block response
	return abcitypes.ResponseInitChain{
		AppHash: bc.appBlockHash[:],
	}
}
