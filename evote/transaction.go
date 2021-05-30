package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
)

func CreateTx(
	inputs []*golosovaniepb.Utxo,
	outputs map[[PkeySize]byte]uint32,
	valueType []byte,
	keys *CryptoKeysData,
	voteType uint32,
	duration uint32,
	ignoreTypeValue bool,
) (*golosovaniepb.Transaction, error) {
	if len(inputs) == 0 || len(outputs) == 0 {
		return nil, fmt.Errorf("at least one output or input required")
	}

	var t golosovaniepb.TxBody

	var maxValInputs uint32 = 0
	var maxValOutputs uint32 = 0
	for pkey, val := range outputs {
		t.Outputs = append(t.Outputs,
			&golosovaniepb.Output{
				ReceiverSpendPkey: pkey[:],
				ReceiverScanPkey:  nil,
				Value:             val,
			})
		maxValOutputs += val
	}

	for _, in := range inputs {
		if (ignoreTypeValue || bytes.Equal(in.ValueType, valueType)) && maxValInputs < maxValOutputs {
			t.Inputs = append(t.Inputs,
				&golosovaniepb.Input{
					PrevTxHash:  in.TxHash,
					OutputIndex: in.Index,
				})
			maxValInputs += in.Value
		}
	}

	if maxValInputs < maxValOutputs {
		return nil, fmt.Errorf("insufficient balance")
	}

	if maxValInputs > maxValOutputs {
		t.Outputs = append(t.Outputs,
			&golosovaniepb.Output{
				ReceiverSpendPkey: inputs[0].ReceiverSpendPkey,
				Value:             maxValInputs - maxValOutputs,
			})
	}
	t.ValueType = valueType
	t.VoteType = voteType
	t.Duration = duration
	// other values are nil
	txBytes, err := proto.Marshal(&t)
	if err != nil {
		return nil, err
	}
	sig := keys.Sign(txBytes)
	return &golosovaniepb.Transaction{
		TxBody: txBytes,
		Sig:    sig,
		Hash:   Hash(txBytes),
	}, nil
}

func CreateMiningReward(keys *CryptoKeysData, rewardForBlock []byte) (*golosovaniepb.Transaction, error) {
	// reward for block is created after that block
	tOut := golosovaniepb.Output{
		ReceiverSpendPkey: keys.PkeyByte[:],
		ReceiverScanPkey:  nil,
		Value:             RewardCoins,
	}
	t := golosovaniepb.TxBody{
		Inputs: nil,
		Outputs: []*golosovaniepb.Output{
			&tOut,
		},
		HashLink:            rewardForBlock[:],
		ValueType:           nil,
		VoteType:            0,
		Duration:            0,
		SenderEphemeralPkey: nil,
		VotersSumPkey:       nil,
	}
	txBytes, err := proto.Marshal(&t)
	if err != nil {
		return nil, err
	}
	sig := keys.Sign(txBytes)
	return &golosovaniepb.Transaction{
		TxBody: txBytes,
		Hash:   Hash(txBytes),
		Sig:    sig,
	}, nil
}
