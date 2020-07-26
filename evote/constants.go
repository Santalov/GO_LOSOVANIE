package evote

//err codes
const (
	OK                = 0
	ERR_TRANS_SIZE    = -1
	ERR_TRANS_VERIFY  = -2
	ERR_BLOCK_SIZE    = -3
	ERR_BLOCK_VERIFY  = -4
	ERR_BLOCK_CREATOR = -5
	ERR_UTXO_SIZE 	  = -6
	ERR_CREATE_TRANS  = -7
)

//size consts
const (
	INT_32_SIZE           = 4
	SIG_SIZE              = 64
	PKEY_SIZE             = 33
	HASH_SIZE             = 32
	TRANS_OUTPUT_SIZE     = INT_32_SIZE + PKEY_SIZE
	TRANS_INPUT_SIZE      = HASH_SIZE + INT_32_SIZE
	MIN_TRANS_SIZE        = INT_32_SIZE*4 + TRANS_OUTPUT_SIZE + SIG_SIZE + HASH_SIZE*2
	MIN_BLOCK_SIZE        = HASH_SIZE*2 + INT_32_SIZE*3
	MAX_BLOCK_SIZE        = 1 * 1024 * 1024
	MAX_TRANS_SIZE        = 100 //тут стоит заглушка, не более 100 транз в блоке
	REWARD                = 1000
	MAX_PREV_BLOCK_HASHES = 10
	UTXO_SIZE 			  = HASH_SIZE*2 + 4*INT_32_SIZE + PKEY_SIZE
)


const (
	ONE_VOTE_TYPE = 0x01
	PERCENT_VOTE_TYPE = 0x02
)

var ZERO_ARRAY_HASH = [HASH_SIZE]byte{}

var ZERO_ARRAY_SIG = [SIG_SIZE]byte{}

// network vars
const (
	PORT = 1337
)

//database fields
const (
	DBNAME     = "blockchain"
	DBUSER     = "blockchain"
	DBPASSWORD = "ffff"
	DBHOST     = "localhost"
)
