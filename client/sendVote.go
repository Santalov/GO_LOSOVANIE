package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"strconv"
)

func sendVote(keys *evote.CryptoKeysData, n *Network, typeValue [evote.HASH_SIZE]byte) {
	validateAmount := func(input string) error {
		_, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return errors.New("invalid number")
		}
		return nil
	}

	promptAmount := promptui.Prompt{
		Label:    "Number of votes",
		Validate: validateAmount,
	}

	amountStr, err := promptAmount.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	amount64, _ := strconv.ParseInt(amountStr, 10, 64)
	amount := uint32(amount64)

	validateReceiver := func(input string) error {
		pkey, err := hex.DecodeString(input)
		if err != nil {
			return errors.New("invalid hex")
		}
		if len(pkey) != evote.PKEY_SIZE {
			return errors.New("invalid pkey size")
		}
		return nil
	}

	promptReceiver := promptui.Prompt{
		Label:    "For candidate",
		Validate: validateReceiver,
	}

	receiverStr, err := promptReceiver.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	receiverSlice, _ := hex.DecodeString(receiverStr)
	var receiver [evote.PKEY_SIZE]byte
	copy(receiver[:], receiverSlice)

	outputs := make(map[[evote.PKEY_SIZE]byte]uint32)
	outputs[receiver] = amount

	pkey := keys.PubkeyByte
	utxos, err := n.GetUtxosByPkey(pkey)
	if err != nil {
		if retryQuestion(n) {
			send(keys, n)
		}
	}
	var tx evote.Transaction
	retCode := tx.CreateTrans(utxos, outputs, typeValue, keys, 0, 0, false)
	if retCode == evote.ERR_CREATE_TRANS {
		fmt.Println("insufficient balance")
		return
	}
	if retCode == evote.OK {
		sendTx(&tx, n)
	} else {
		panic("unknown err")
	}
}
