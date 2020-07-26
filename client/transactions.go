package main

import (
	"GO_LOSOVANIE/evote"
	"fmt"
)

func transactions(keys *evote.CryptoKeysData, n *Network) {
	pkey := keys.PubkeyByte
	txs, err := n.GetTxsByPkey(pkey)
	if err != nil {
		if retryQuestion(n) {
			transactions(keys, n)
		}
	}
	for i, tx := range txs {
		fmt.Printf("typeValue: %v\n", bToHex(tx.TypeValue[:]))
		fmt.Println("inputs:")
		for i, input := range tx.Inputs {
			fmt.Printf(
				"%v:\n  prevId: %v\n  outIndex: %v\n",
				i, bToHex(input.PrevId[:]), input.OutIndex,
			)
		}
		fmt.Println("outputs:")
		for i, output := range tx.Outputs {
			fmt.Printf(
				"%v:\n  pkeyTo: %v\n  value: %v\n",
				i, bToHex(output.PkeyTo[:]), output.Value,
			)
		}
		fmt.Printf(
			"typeVote: %v\nduration: %v\nhashLink: %v\n",
			tx.TypeVote, tx.Duration, bToHex(tx.HashLink[:]),
		)
		if i != len(txs)-1 {
			fmt.Println()
		}
	}
}
