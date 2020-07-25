package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"strconv"
)

func send(keys *evote.CryptoKeysData, n *Network) {
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
		if len(pkey) != evote.PKEY_SIZE {
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
	var receiver [evote.PKEY_SIZE]byte
	copy(receiver[:], receiverSlice)

	outputs := make(map[[evote.PKEY_SIZE]byte]uint32)
	outputs[receiver] = amount

	pkey := keys.PubkeyByte
	utxos, err := n.GetUtxosByPkey(pkey)
	if err != nil {
		if retryQuestion() {
			send(keys, n)
		}
	}
	var tx evote.Transaction
	retCode := tx.CreateTrans(utxos, outputs, evote.ZERO_ARRAY_HASH, keys)
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

func sendTx(tx *evote.Transaction, n *Network) {
	err := n.SubmitTx(tx.ToBytes())
	if err != nil {
		if retryQuestion() {
			sendTx(tx, n)
		}
	}
}
