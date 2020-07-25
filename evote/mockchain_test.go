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
		InputSize:  0,
		Inputs:     nil,
		OutputSize: 1,
		Outputs: []TransactionOutput{
			{
				Value:  val,
				PkeyTo: pkeyTo,
			},
		},
		Duration:  blockNumber,
		HashLink:  ZERO_ARRAY_HASH,
		Signature: randSig(),
		TypeVote:  0,
		TypeValue: ZERO_ARRAY_HASH,
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
		res[i].Transaction = tx
		copy(res[i].Hash[:], Hash(tx.ToBytes()))
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
		InputSize: 1,
		Inputs: []TransactionInput{
			{
				PrevId:   txHash(COINBASE_TXS[0]),
				OutIndex: 0,
			},
		},
		OutputSize: 2,
		Outputs: []TransactionOutput{
			{
				Value:  400,
				PkeyTo: keyPairs[1].pub,
			},
			{
				Value:  600,
				PkeyTo: keyPairs[0].pub,
			},
		},
		Duration:  0,
		TypeVote:  0,
		TypeValue: ZERO_ARRAY_HASH,
		Signature: randSig(),
		HashLink:  ZERO_ARRAY_HASH,
	},
}

var TXS_BLOCK2 = []*Transaction{
	COINBASE_TXS[2],
	{
		InputSize: 2,
		Inputs: []TransactionInput{
			{
				PrevId:   txHash(TXS_BLOCK1[1]),
				OutIndex: 1,
			},
			{
				PrevId:   txHash(TXS_BLOCK1[0]),
				OutIndex: 0,
			},
		},
		OutputSize: 2,
		Outputs: []TransactionOutput{
			{
				Value:  2400,
				PkeyTo: keyPairs[2].pub,
			},
			{
				Value:  200,
				PkeyTo: keyPairs[2].pub,
			},
		},
		Duration:  0,
		TypeVote:  0,
		TypeValue: ZERO_ARRAY_HASH,
		Signature: randSig(),
		HashLink:  ZERO_ARRAY_HASH,
	},
}

var TXS_BLOCK3 = []*Transaction{
	COINBASE_TXS[3],
	{ // транза создания голосования
		InputSize: 1,
		Inputs: []TransactionInput{
			{
				PrevId:   txHash(TXS_BLOCK2[0]),
				OutIndex: 0,
			},
		},
		OutputSize: 3,
		Outputs: []TransactionOutput{
			{
				Value:  2000,
				PkeyTo: keyPairs[3].pub,
			},
			{
				Value:  3000,
				PkeyTo: keyPairs[4].pub,
			},
			{
				Value:  4000,
				PkeyTo: keyPairs[5].pub,
			},
		},
		Duration:  100,
		TypeVote:  1, // у транзы создания голосования ненулевой id
		TypeValue: ZERO_ARRAY_HASH,
		HashLink:  ZERO_ARRAY_HASH,
	},
}

var TXS_BLOCK4 = []*Transaction{
	COINBASE_TXS[4],
	{ // траза голосвания
		InputSize: 1,
		Inputs: []TransactionInput{
			{
				PrevId:   txHash(TXS_BLOCK3[1]),
				OutIndex: 0,
			},
		},
		OutputSize: 1,
		Outputs: []TransactionOutput{
			{
				Value:  2000,
				PkeyTo: keyPairs[6].pub,
			},
		},
		Duration:  0,
		TypeVote:  0,
		TypeValue: txHash(TXS_BLOCK3[1]),
		HashLink:  ZERO_ARRAY_HASH,
	},
}

var BLOCK0 = Block{
	PrevBlockHash: ZERO_ARRAY_HASH,
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK0)),
	Trans:         appendHashes(TXS_BLOCK0),
}

var BLOCK1 = Block{
	PrevBlockHash: blockHash(&BLOCK0),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(10 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK1)),
	Trans:         appendHashes(TXS_BLOCK1),
}

var BLOCK2 = Block{
	PrevBlockHash: blockHash(&BLOCK1),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(20 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK2)),
	Trans:         appendHashes(TXS_BLOCK2),
}

var BLOCK3 = Block{
	PrevBlockHash: blockHash(&BLOCK2),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(30 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK3)),
	Trans:         appendHashes(TXS_BLOCK3),
}

var BLOCK4 = Block{
	PrevBlockHash: blockHash(&BLOCK3),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(30 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK4)),
	Trans:         appendHashes(TXS_BLOCK4),
}

var B_AND_H0 = BlocAndkHash{
	B:    &BLOCK0,
	Hash: blockHash(&BLOCK0),
}

var B_AND_H1 = BlocAndkHash{
	B:    &BLOCK1,
	Hash: blockHash(&BLOCK1),
}

var B_AND_H2 = BlocAndkHash{
	B:    &BLOCK2,
	Hash: blockHash(&BLOCK2),
}

var B_AND_H3 = BlocAndkHash{
	B:    &BLOCK3,
	Hash: blockHash(&BLOCK3),
}

var B_AND_H4 = BlocAndkHash{
	B:    &BLOCK4,
	Hash: blockHash(&BLOCK4),
}
