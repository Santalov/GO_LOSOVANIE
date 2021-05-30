package main

import (
	"GO_LOSOVANIE/evote"
	"GO_LOSOVANIE/evote/golosovaniepb"
	"fmt"
	"github.com/golang/protobuf/proto"
)

func voteInfo(keys *evote.CryptoKeysData, n *evote.Network, typeValue [evote.HashSize]byte) {
	txs, err := n.GetTxsByHashes([][]byte{typeValue[:]})
	if retryQuestion(err, n) {
		voteInfo(keys, n, typeValue)
	}
	if len(txs) != 1 {
		fmt.Println("Error: no transactions with hash", typeValue)
	}
	tx := txs[0]
	var body golosovaniepb.TxBody
	err = proto.Unmarshal(tx.TxBody, &body)
	if err != nil {
		fmt.Println("error during parsing tx", tx.TxBody, err)
		return
	}
	var typeVote string
	if body.VoteType == evote.OneVoteType {
		typeVote = "Majority"
	} else if body.VoteType == evote.PercentVoteType {
		typeVote = "Percentage"
	} else {
		fmt.Println("It is not a voting")
		return
	}
	fmt.Println("Voting type:", typeVote)
	fmt.Println("Duration:", body.Duration, "seconds")
	fmt.Println("Participants:")
	for _, output := range body.Outputs {
		fmt.Printf("  %v votes: %v\n", bToHex(output.ReceiverSpendPkey), output.Value)
	}
}
