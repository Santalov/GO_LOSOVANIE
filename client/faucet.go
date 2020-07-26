package main

import (
	"GO_LOSOVANIE/evote"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"strconv"
)

func faucet(keys *evote.CryptoKeysData, n *Network) {
	validateAmount := func(input string) error {
		_, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return errors.New("invalid number")
		}
		return nil
	}

	promptAmount := promptui.Prompt{
		Label:    "Amount",
		Validate: validateAmount,
	}

	amountStr, err := promptAmount.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	amount64, _ := strconv.ParseInt(amountStr, 10, 64)
	amount := uint32(amount64)
	sendFaucet(amount, keys.PubkeyByte, n)
}

func sendFaucet(amount uint32, pkey [evote.PKEY_SIZE]byte, n *Network) {
	err := n.Faucet(amount, pkey)
	if err != nil {
		if retryQuestion(n) {
			sendFaucet(amount, pkey, n)
		}
	}
}
