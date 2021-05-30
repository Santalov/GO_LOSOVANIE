package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"crypto/rand"
	"github.com/golang/protobuf/proto"
	"time"
)

type keyPair struct {
	pub  []byte
	priv []byte
}

func randPkey() []byte {
	res := make([]byte, PkeySize, PkeySize)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	return res
}

func randHash() []byte {
	res := make([]byte, HashSize, HashSize)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	return res
}

func randSig() []byte {
	res := make([]byte, SigSize, SigSize)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	return res
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

func tx(txBody *golosovaniepb.TxBody) *golosovaniepb.Transaction {
	bodyBytes, err := proto.Marshal(txBody)
	if err != nil {
		panic(err)
	}
	return &golosovaniepb.Transaction{
		TxBody: bodyBytes,
		Hash:   Hash(bodyBytes),
		Sig:    randSig(),
	}
}

func makeCoinbaseTx(val uint32, pkeyTo []byte, rewardForBlock []byte) *golosovaniepb.Transaction {
	txBody := golosovaniepb.TxBody{
		Inputs: nil,
		Outputs: []*golosovaniepb.Output{
			{
				Value:             val,
				ReceiverSpendPkey: pkeyTo[:],
			},
		},
		HashLink:            rewardForBlock,
		ValueType:           nil,
		VoteType:            0,
		Duration:            0,
		SenderEphemeralPkey: nil,
		VotersSumPkey:       nil,
	}
	return tx(&txBody)
}

func block(
	transactions []*golosovaniepb.Transaction,
	prevHash []byte,
	timestamp time.Time,
	proposerPkey []byte,
) *golosovaniepb.Block {
	var proposer [PkeySize]byte
	copy(proposer[:], proposerPkey)
	b, err := CreateBlock(
		transactions,
		prevHash,
		timestamp,
		proposer,
	)
	if err != nil {
		panic(err)
	}
	return b
}

// Z-block, because of coinbase tx update
// just a regular block, i use letter z to avoid changing enumeration
var TxsBlockZ []*golosovaniepb.Transaction

var BlockZ = block(
	TxsBlockZ,
	nil,
	time.Now().Add(-10*time.Second),
	keyPairs[0].pub,
)

var TxsBlock0 = []*golosovaniepb.Transaction{
	makeCoinbaseTx(1000, keyPairs[0].pub, BlockZ.Hash),
}

var Block0 = block(
	TxsBlock0,
	BlockZ.Hash,
	time.Now(),
	keyPairs[0].pub,
)

var TxsBlock1 = []*golosovaniepb.Transaction{
	makeCoinbaseTx(2000, keyPairs[0].pub, Block0.Hash),
	tx(&golosovaniepb.TxBody{
		Inputs: []*golosovaniepb.Input{
			{
				PrevTxHash:  TxsBlock0[0].Hash,
				OutputIndex: 0,
			},
		},
		Outputs: []*golosovaniepb.Output{
			{
				Value:             400,
				ReceiverSpendPkey: keyPairs[1].pub,
			},
			{
				Value:             600,
				ReceiverSpendPkey: keyPairs[0].pub,
			},
		},
		HashLink:            nil,
		ValueType:           nil,
		VoteType:            0,
		Duration:            0,
		SenderEphemeralPkey: nil,
		VotersSumPkey:       nil,
	}),
}

var Block1 = block(
	TxsBlock1,
	Block0.Hash,
	time.Now().Add(10*time.Second),
	keyPairs[1].pub,
)

var TxsBlock2 = []*golosovaniepb.Transaction{
	makeCoinbaseTx(3000, keyPairs[1].pub, Block1.Hash),
	tx(&golosovaniepb.TxBody{
		Inputs: []*golosovaniepb.Input{
			{
				PrevTxHash:  TxsBlock1[1].Hash,
				OutputIndex: 1,
			},
			{
				PrevTxHash:  TxsBlock1[0].Hash,
				OutputIndex: 0,
			},
		},
		Outputs: []*golosovaniepb.Output{
			{
				Value:             2400,
				ReceiverSpendPkey: keyPairs[2].pub,
			},
			{
				Value:             200,
				ReceiverSpendPkey: keyPairs[2].pub,
			},
		},
		HashLink:            nil,
		ValueType:           nil,
		VoteType:            0,
		Duration:            0,
		SenderEphemeralPkey: nil,
		VotersSumPkey:       nil,
	}),
}

var Block2 = block(
	TxsBlock2,
	Block1.Hash,
	time.Now().Add(20*time.Second),
	keyPairs[1].pub,
)

var TxsBlock3 = []*golosovaniepb.Transaction{
	makeCoinbaseTx(3000, keyPairs[1].pub, Block2.Hash),
	tx(&golosovaniepb.TxBody{ // транза создания голосования
		Inputs: []*golosovaniepb.Input{
			{
				PrevTxHash:  TxsBlock2[0].Hash,
				OutputIndex: 0,
			},
		},
		Outputs: []*golosovaniepb.Output{
			{
				Value:             2000,
				ReceiverSpendPkey: keyPairs[3].pub,
			},
			{
				Value:             3000,
				ReceiverSpendPkey: keyPairs[4].pub,
			},
			{
				Value:             4000,
				ReceiverSpendPkey: keyPairs[5].pub,
			},
		},
		HashLink:            nil,
		ValueType:           nil,
		VoteType:            1, // у транзаы создания голосования ненулевой voteType
		Duration:            100,
		SenderEphemeralPkey: nil,
		VotersSumPkey:       nil,
	}),
}

var Block3 = block(
	TxsBlock3,
	Block2.Hash,
	time.Now().Add(30*time.Second),
	keyPairs[6].pub,
)

var TxsBlock4 = []*golosovaniepb.Transaction{
	makeCoinbaseTx(3000, keyPairs[6].pub, Block3.Hash),
	tx(&golosovaniepb.TxBody{ // транзакция отправки голоса
		Inputs: []*golosovaniepb.Input{
			{
				PrevTxHash:  TxsBlock3[1].Hash,
				OutputIndex: 0,
			},
		},
		Outputs: []*golosovaniepb.Output{
			{
				Value:             2000,
				ReceiverSpendPkey: keyPairs[6].pub,
			},
		},
		HashLink:            nil,
		ValueType:           TxsBlock3[1].Hash,
		VoteType:            0,
		Duration:            0,
		SenderEphemeralPkey: nil,
		VotersSumPkey:       nil,
	}),
}

var Block4 = block(
	TxsBlock4,
	Block3.Hash,
	time.Now().Add(30*time.Second),
	keyPairs[6].pub,
)
