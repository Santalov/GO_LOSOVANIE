package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"fmt"
	"github.com/manifoldco/promptui"
)

func retryQuestion(n *evote.Network) bool {
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

func calcBalance(utxos []*evote.UTXO, typeValue [evote.HASH_SIZE]byte) (balanceOfTypeValue, otherBalance uint32) {
	for _, utxo := range utxos {
		if utxo.TypeValue == typeValue {
			balanceOfTypeValue += utxo.Value
		} else {
			otherBalance += utxo.Value
		}
	}
	return
}

func pkeyHex(pkey [evote.PKEY_SIZE]byte) string {
	return hex.EncodeToString(pkey[:])
}

func bToHex(data []byte) string {
	return hex.EncodeToString(data)
}
