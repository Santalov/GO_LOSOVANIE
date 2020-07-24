package evote

import (
	"crypto/rand"
	"time"
)

type keyPair struct {
	pub  [PKEY_SIZE]byte
	priv []byte
}

func randPkey() [PKEY_SIZE]byte {
	res := make([]byte, PKEY_SIZE, PKEY_SIZE)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	resres := [PKEY_SIZE]byte{}
	copy(resres[:], res)
	return resres
}

func randHash() [HASH_SIZE]byte {
	res := make([]byte, HASH_SIZE, HASH_SIZE)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	resres := [HASH_SIZE]byte{}
	copy(resres[:], res)
	return resres
}

func randSig() [SIG_SIZE]byte {
	res := make([]byte, SIG_SIZE, SIG_SIZE)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	resres := [SIG_SIZE]byte{}
	copy(resres[:], res)
	return resres
}

var keyPairs = []keyPair{
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
	{
		pub:  randPkey(),
		priv: make([]byte, 32, 32),
	},
}

func makeCoinbaseTx(val uint32, pkeyTo [PKEY_SIZE]byte, blockNumber uint32) *Transaction {
	return &Transaction{
		inputSize:  0,
		inputs:     nil,
		outputSize: 1,
		outputs: []TransactionOutput{
			{
				value:  val,
				pkeyTo: pkeyTo,
			},
		},
		duration:  0,
		hashLink:  ZERO_ARRAY_HASH,
		signature: randSig(),
		typeVote:  blockNumber,
		typeValue: ZERO_ARRAY_HASH,
	}
}

func txHash(tx *Transaction) [HASH_SIZE]byte {
	res := [HASH_SIZE]byte{}
	copy(res[:], Hash(tx.ToBytes()))
	return res
}

func blockHash(block *Block) [HASH_SIZE]byte {
	var res [HASH_SIZE]byte
	copy(res[:], Hash(block.ToBytes()))
	return res
}

func appendHashes(txs []*Transaction) []TransAndHash {
	res := make([]TransAndHash, len(txs))
	for i, tx := range txs {
		res[i].transaction = tx
		copy(res[i].hash[:], Hash(tx.ToBytes()))
	}
	return res
}

var COINBASE_TXS = []*Transaction{
	makeCoinbaseTx(1000, keyPairs[0].pub, 0),
	makeCoinbaseTx(2000, keyPairs[0].pub, 1),
	makeCoinbaseTx(3000, keyPairs[1].pub, 2),
	makeCoinbaseTx(3000, keyPairs[1].pub, 3),
	makeCoinbaseTx(3000, keyPairs[1].pub, 4),
}

var TXS_BLOCK0 = []*Transaction{
	COINBASE_TXS[0],
}

var TXS_BLOCK1 = []*Transaction{
	COINBASE_TXS[1],
	{
		inputSize: 1,
		inputs: []TransactionInput{
			{
				prevId:   txHash(COINBASE_TXS[0]),
				outIndex: 0,
			},
		},
		outputSize: 2,
		outputs: []TransactionOutput{
			{
				value:  400,
				pkeyTo: keyPairs[1].pub,
			},
			{
				value:  600,
				pkeyTo: keyPairs[0].pub,
			},
		},
		duration:  0,
		typeVote:  0,
		typeValue: ZERO_ARRAY_HASH,
		signature: randSig(),
		hashLink:  ZERO_ARRAY_HASH,
	},
}

var TXS_BLOCK2 = []*Transaction{
	COINBASE_TXS[2],
	{
		inputSize: 2,
		inputs: []TransactionInput{
			{
				prevId:   txHash(TXS_BLOCK1[1]),
				outIndex: 1,
			},
			{
				prevId:   txHash(TXS_BLOCK1[0]),
				outIndex: 0,
			},
		},
		outputSize: 2,
		outputs: []TransactionOutput{
			{
				value:  2400,
				pkeyTo: keyPairs[2].pub,
			},
			{
				value:  200,
				pkeyTo: keyPairs[2].pub,
			},
		},
		duration:  0,
		typeVote:  0,
		typeValue: ZERO_ARRAY_HASH,
		signature: randSig(),
		hashLink:  ZERO_ARRAY_HASH,
	},
}

var TXS_BLOCK3 = []*Transaction{
	COINBASE_TXS[3],
	{ // транза создания голосования
		inputSize: 1,
		inputs: []TransactionInput{
			{
				prevId:   txHash(TXS_BLOCK2[0]),
				outIndex: 0,
			},
		},
		outputSize: 3,
		outputs: []TransactionOutput{
			{
				value:  2000,
				pkeyTo: keyPairs[3].pub,
			},
			{
				value:  3000,
				pkeyTo: keyPairs[4].pub,
			},
			{
				value:  4000,
				pkeyTo: keyPairs[5].pub,
			},
		},
		duration:  100,
		typeVote:  1,
		typeValue: ZERO_ARRAY_HASH,
		hashLink:  ZERO_ARRAY_HASH,
	},
}

var TXS_BLOCK4 = []*Transaction{
	COINBASE_TXS[4],
	{ // траза голосвания
		inputSize: 1,
		inputs: []TransactionInput{
			{
				prevId:   txHash(TXS_BLOCK3[1]),
				outIndex: 0,
			},
		},
		outputSize: 1,
		outputs: []TransactionOutput{
			{
				value:  2000,
				pkeyTo: keyPairs[6].pub,
			},
		},
		duration:  0,
		typeVote:  0,
		typeValue: txHash(TXS_BLOCK3[1]),
		hashLink:  ZERO_ARRAY_HASH,
	},
}

var BLOCK0 = Block{
	prevBlockHash: ZERO_ARRAY_HASH,
	merkleTree:    randHash(),
	timestamp:     uint64(time.Now().UnixNano()),
	transSize:     uint32(len(TXS_BLOCK0)),
	trans:         appendHashes(TXS_BLOCK0),
}

var BLOCK1 = Block{
	prevBlockHash: blockHash(&BLOCK0),
	merkleTree:    randHash(),
	timestamp:     uint64(time.Now().Add(10 * time.Second).UnixNano()),
	transSize:     uint32(len(TXS_BLOCK1)),
	trans:         appendHashes(TXS_BLOCK1),
}

var BLOCK2 = Block{
	prevBlockHash: blockHash(&BLOCK1),
	merkleTree:    randHash(),
	timestamp:     uint64(time.Now().Add(20 * time.Second).UnixNano()),
	transSize:     uint32(len(TXS_BLOCK2)),
	trans:         appendHashes(TXS_BLOCK2),
}

var BLOCK3 = Block{
	prevBlockHash: blockHash(&BLOCK2),
	merkleTree:    randHash(),
	timestamp:     uint64(time.Now().Add(30 * time.Second).UnixNano()),
	transSize:     uint32(len(TXS_BLOCK3)),
	trans:         appendHashes(TXS_BLOCK3),
}

var BLOCK4 = Block{
	prevBlockHash: blockHash(&BLOCK3),
	merkleTree:    randHash(),
	timestamp:     uint64(time.Now().Add(30 * time.Second).UnixNano()),
	transSize:     uint32(len(TXS_BLOCK4)),
	trans:         appendHashes(TXS_BLOCK4),
}

var B_AND_H0 = BlocAndkHash{
	b:    &BLOCK0,
	hash: blockHash(&BLOCK0),
}

var B_AND_H1 = BlocAndkHash{
	b:    &BLOCK1,
	hash: blockHash(&BLOCK1),
}

var B_AND_H2 = BlocAndkHash{
	b:    &BLOCK2,
	hash: blockHash(&BLOCK2),
}

var B_AND_H3 = BlocAndkHash{
	b:    &BLOCK3,
	hash: blockHash(&BLOCK3),
}

var B_AND_H4 = BlocAndkHash{
	b:    &BLOCK4,
	hash: blockHash(&BLOCK4),
}
