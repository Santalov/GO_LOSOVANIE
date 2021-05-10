package main

import (
	"GO_LOSOVANIE/evote"
	"fmt"
)

func voteResults(keys *evote.CryptoKeysData, n *evote.Network, typeValue [evote.HASH_SIZE]byte) {
	results, err := n.VoteResults(typeValue)
	if err != nil {
		if retryQuestion(n) {
			voteResults(keys, n, typeValue)
		}
	}
	for candidate, val := range results {
		fmt.Printf("pkey: %v votes: %v\n", pkeyHex(candidate), val)
	}
	if len(results) == 0 {
		fmt.Println("no votes no results")
	}
}
