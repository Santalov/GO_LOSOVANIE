package main

import (
	"GO_LOSOVANIE/evote"
	"GO_LOSOVANIE/evote/golosovaniepb"
	"fmt"
	"github.com/golang/protobuf/proto"
)

func transactions(keys *evote.CryptoKeysData, n *evote.Network) {
	pkey := keys.PkeyByte
	txs, err := n.GetTxsByPkey(pkey[:])
	if retryQuestion(err, n) {
		transactions(keys, n)
	}
	for i, tx := range txs {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		if err != nil {
			fmt.Println("error during parse tx ", bToHex(tx.Hash), err)
			continue
		}
		fmt.Printf("typeValue: %v\n", bToHex(body.ValueType))
		fmt.Println("inputs:")
		for i, input := range body.Inputs {
			fmt.Printf(
				"%v:\n  prevId: %v\n  outIndex: %v\n",
				i, bToHex(input.PrevTxHash), input.OutputIndex,
			)
		}
		fmt.Println("outputs:")
		for i, output := range body.Outputs {
			fmt.Printf(
				"%v:\n  pkeyTo: %v\n  value: %v\n",
				i, bToHex(output.ReceiverSpendPkey), output.Value,
			)
		}
		fmt.Printf(
			"typeVote: %v\nduration: %v\nhashLink: %v\n",
			body.VoteType, body.Duration, bToHex(body.HashLink),
		)
		if i != len(txs)-1 {
			fmt.Println()
		}
	}
}
