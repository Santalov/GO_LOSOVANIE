package main

import (
	"GO_LOSOVANIE/evote"
	"fmt"
)

func voteInfo(keys *evote.CryptoKeysData, n *evote.Network, typeValue [evote.HASH_SIZE]byte) {
	txs, err := n.GetTxsByHashes([][evote.HASH_SIZE]byte{typeValue})
	if err != nil {
		if retryQuestion(n) {
			voteInfo(keys, n, typeValue)
		}
	}
	if len(txs) != 1 {
		fmt.Println("Error: no transactions with hash", typeValue)
	}
	tx := txs[0]
	var typeVote string
	if tx.TypeVote == evote.ONE_VOTE_TYPE {
		typeVote = "Majority"
	} else if tx.TypeVote == evote.PERCENT_VOTE_TYPE {
		typeVote = "Percentage"
	} else {
		fmt.Println("It is not a voting")
		return
	}
	fmt.Println("Voting type:", typeVote)
	fmt.Println("Duration:", tx.Duration, "seconds")
	fmt.Println("Participants:")
	for _, output := range tx.Outputs {
		fmt.Printf("  %v votes: %v\n", pkeyHex(output.PkeyTo), output.Value)
	}
}
