package main

import (
	"GO_LOSOVANIE/evote"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"strconv"
)

func faucet(keys *evote.CryptoKeysData, n *evote.Network) {
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
	sendFaucet(amount, keys.PkeyByte, n)
}

func sendFaucet(amount uint32, pkey [evote.PkeySize]byte, n *evote.Network) {
	err := n.Faucet(amount, pkey[:])
	if retryQuestion(err, n) {
		sendFaucet(amount, pkey, n)
	}
}
