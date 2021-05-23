package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"strconv"
)

func send(keys *evote.CryptoKeysData, n *evote.Network) {
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

	validateReceiver := func(input string) error {
		pkey, err := hex.DecodeString(input)
		if err != nil {
			return errors.New("invalid hex")
		}
		if len(pkey) != evote.PkeySize {
			return errors.New("invalid pkey size")
		}
		return nil
	}

	promptReceiver := promptui.Prompt{
		Label:    "To",
		Validate: validateReceiver,
	}

	receiverStr, err := promptReceiver.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	receiverSlice, _ := hex.DecodeString(receiverStr)
	var receiver [evote.PkeySize]byte
	copy(receiver[:], receiverSlice)

	outputs := make(map[[evote.PkeySize]byte]uint32)
	outputs[receiver] = amount

	pkey := keys.PkeyByte
	utxos, err := n.GetUtxosByPkey(pkey)
	if retryQuestion(err, n) {
		send(keys, n)
	}
	var tx evote.Transaction
	retCode := tx.CreateTrans(utxos, outputs, evote.ZeroArrayHash, keys, 0, 0, false)
	if retCode == evote.ErrCreateTrans {
		fmt.Println("insufficient balance")
		return
	}
	if retCode == evote.OK {
		sendTx(&tx, n)
	} else {
		panic("unknown err")
	}
}

func sendTx(tx *evote.Transaction, n *evote.Network) {
	err := n.SubmitTx(tx.ToBytes())
	if retryQuestion(err, n) {
		sendTx(tx, n)
	}
}
