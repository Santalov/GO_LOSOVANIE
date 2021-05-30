package main

import (
	"GO_LOSOVANIE/evote"
	"GO_LOSOVANIE/evote/golosovaniepb"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
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
	utxos, err := n.GetUtxosByPkey(pkey[:])
	if retryQuestion(err, n) {
		send(keys, n)
	}
	tx, err := evote.CreateTx(utxos, outputs, nil, keys, 0, 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}
	sendTx(tx, n)
}

func sendTx(tx *golosovaniepb.Transaction, n *evote.Network) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		panic(err)
	}
	err = n.SubmitTx(txBytes)
	if retryQuestion(err, n) {
		sendTx(tx, n)
	}
}
