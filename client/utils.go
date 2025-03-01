package main

import (
	"GO_LOSOVANIE/evote"
	"GO_LOSOVANIE/evote/golosovaniepb"
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/manifoldco/promptui"
)

func retryQuestion(err error, n *evote.Network) bool {
	if err == nil {
		return false
	}
	fmt.Println("An error occurred during request:", err)
	prompt := promptui.Select{
		Label: "Retry?",
		Items: []string{"Yes", "No"},
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return false
	}

	if result == "Yes" {
		n.SelectNextHost()
		return true
	} else {
		return false
	}
}

func calcBalance(utxos []*golosovaniepb.Utxo, valueType []byte) (balanceOfValueType, otherBalance uint32) {
	for _, utxo := range utxos {
		if bytes.Equal(utxo.ValueType, valueType) {
			balanceOfValueType += utxo.Value
		} else {
			otherBalance += utxo.Value
		}
	}
	return
}

func pkeyHex(pkey [evote.PkeySize]byte) string {
	return hex.EncodeToString(pkey[:])
}

func bToHex(data []byte) string {
	return hex.EncodeToString(data)
}
