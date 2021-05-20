package evote

import (
	"crypto/rand"
	"time"
)

type keyPair struct {
	pub  [PkeySize]byte
	priv []byte
}

func randPkey() [PkeySize]byte {
	res := make([]byte, PkeySize, PkeySize)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	resres := [PkeySize]byte{}
	copy(resres[:], res)
	return resres
}

func randHash() [HashSize]byte {
	res := make([]byte, HashSize, HashSize)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	resres := [HashSize]byte{}
	copy(resres[:], res)
	return resres
}

func randSig() [SigSize]byte {
	res := make([]byte, SigSize, SigSize)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	resres := [SigSize]byte{}
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

func makeCoinbaseTx(val uint32, pkeyTo [PkeySize]byte, rewardForBlock [HashSize]byte) *Transaction {
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
		Duration:  0,
		HashLink:  rewardForBlock,
		Signature: randSig(),
		TypeVote:  0,
		TypeValue: ZeroArrayHash,
	}
}

func txHash(tx *Transaction) [HashSize]byte {
	res := [HashSize]byte{}
	copy(res[:], Hash(tx.ToBytes()))
	return res
}

func blockHash(block *Block) [HashSize]byte {
	var res [HashSize]byte
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

// Z-block, because of coinbase tx update
// just a regular block, i use letter z to avoid changing enumeration
var TXS_BLOCKZ = []*Transaction{}

var BLOCKZ = Block{
	PrevBlockHash: ZeroArrayHash,
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(-10 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCKZ)),
	Trans:         appendHashes(TXS_BLOCKZ),
	proposerPkey:  keyPairs[0].pub,
}

var TXS_BLOCK0 = []*Transaction{
	makeCoinbaseTx(1000, keyPairs[0].pub, blockHash(&BLOCKZ)),
}

var BLOCK0 = Block{
	PrevBlockHash: blockHash(&BLOCKZ),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK0)),
	Trans:         appendHashes(TXS_BLOCK0),
	proposerPkey:  keyPairs[0].pub,
}

var TXS_BLOCK1 = []*Transaction{
	makeCoinbaseTx(2000, keyPairs[0].pub, blockHash(&BLOCK0)),
	{
		InputSize: 1,
		Inputs: []TransactionInput{
			{
				PrevId:   txHash(TXS_BLOCK0[0]),
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
		TypeValue: ZeroArrayHash,
		Signature: randSig(),
		HashLink:  ZeroArrayHash,
	},
}

var BLOCK1 = Block{
	PrevBlockHash: blockHash(&BLOCK0),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(10 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK1)),
	Trans:         appendHashes(TXS_BLOCK1),
	proposerPkey:  keyPairs[1].pub,
}

var TXS_BLOCK2 = []*Transaction{
	makeCoinbaseTx(3000, keyPairs[1].pub, blockHash(&BLOCK1)),
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
		TypeValue: ZeroArrayHash,
		Signature: randSig(),
		HashLink:  ZeroArrayHash,
	},
}

var BLOCK2 = Block{
	PrevBlockHash: blockHash(&BLOCK1),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(20 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK2)),
	Trans:         appendHashes(TXS_BLOCK2),
	proposerPkey:  keyPairs[1].pub,
}

var TXS_BLOCK3 = []*Transaction{
	makeCoinbaseTx(3000, keyPairs[1].pub, blockHash(&BLOCK2)),
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
		TypeValue: ZeroArrayHash,
		HashLink:  ZeroArrayHash,
	},
}

var BLOCK3 = Block{
	PrevBlockHash: blockHash(&BLOCK2),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(30 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK3)),
	Trans:         appendHashes(TXS_BLOCK3),
	proposerPkey:  keyPairs[6].pub,
}

var TXS_BLOCK4 = []*Transaction{
	makeCoinbaseTx(3000, keyPairs[6].pub, blockHash(&BLOCK3)),
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
		HashLink:  ZeroArrayHash,
	},
}

var BLOCK4 = Block{
	PrevBlockHash: blockHash(&BLOCK3),
	MerkleTree:    randHash(),
	Timestamp:     uint64(time.Now().Add(30 * time.Second).UnixNano()),
	TransSize:     uint32(len(TXS_BLOCK4)),
	Trans:         appendHashes(TXS_BLOCK4),
	proposerPkey:  keyPairs[6].pub,
}

var B_AND_HZ = BlocAndkHash{
	B:    &BLOCKZ,
	Hash: blockHash(&BLOCKZ),
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
