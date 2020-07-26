package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"fmt"
	"github.com/manifoldco/promptui"
)

func retryQuestion(n *Network) bool {
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

func calcBalance(utxos []*evote.UTXO) uint32 {
	var balance uint32
	for _, utxo := range utxos {
		balance += utxo.Value
	}
	return balance
}

func pkeyHex(pkey [evote.PKEY_SIZE]byte) string {
	return hex.EncodeToString(pkey[:])
}

func bToHex(data []byte) string {
	return hex.EncodeToString(data)
}
