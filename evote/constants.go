package evote

//err codes
const (
	OK              = 0
	ErrTransSize    = -1
	ErrTransVerify  = -2
	ErrBlockSize    = -3
	ErrBlockVerify  = -4
	ErrBlockCreator = -5
	ErrUtxoSize     = -6
	ErrCreateTrans  = -7
)

// return codes for CheckTx and DeliveryTx abci methods, for client request handlers
const (
	CodeOk uint32 = iota
	CodeParseErr
	CodeNoOutputs
	CodeHashLinkAndTypeVoteTogether
	CodeDoubleSpending
	CodeMixingTypeValue
	CodeInputsNotMatchOutputs
	CodeInvalidSignature
	CodeCoinbaseTxNoOutput
	CodeCoinbaseNoBlock
	CodeCoinbaseProposerMismatch
	CodeCoinbaseIncorrectReward
	CodeDoubleCoinbaseForSameBlock
	CodeInvalidDataLen
	CodeDatabaseFailed
	CodeBroadcastTxFailed
	CodeUnknownPath
)

//size consts
const (
	Int32Size       = 4
	SigSize         = 64 + 1 // one bit for pkey recovery
	PkeySize        = 33
	TmPkeySize      = 32 // TM is abbr from tendermint
	TmAddrSize      = 20
	HashSize        = 32
	TransOutputSize = Int32Size + PkeySize
	TransInputSize  = HashSize + Int32Size
	MinTransSize    = Int32Size*4 + TransOutputSize + SigSize + HashSize*2
	MinBlockSize    = HashSize*2 + PkeySize + Int32Size*3
	MaxBlockSize    = 1 * 1024 * 1024
	RewardCoins     = 1000
	UtxoSize        = HashSize*2 + 4*Int32Size + PkeySize
)

const (
	OneVoteType     = 0x01
	PercentVoteType = 0x02
)

var ZeroArrayHash = [HashSize]byte{}

var ZeroArraySig = [SigSize]byte{}

var ZeroArrayPkey = [PkeySize]byte{}

//database fields
const (
	DbName     = "blockchain"
	DbUser     = "blockchain"
	DbPassword = "ffff"
	DbHost     = "localhost"
)
