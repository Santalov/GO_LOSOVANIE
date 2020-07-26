package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"strconv"
	"strings"
)

func createVote(keys *evote.CryptoKeysData, n *Network) {
	var typeVote, amountPerParticipant uint32
	prompt := promptui.Select{
		Label: "Select vote type",
		Items: []string{"Majority", "Percentage"},
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Fail %v\n", err)
		return
	}

	if result == "Majority" {
		typeVote = 1
	} else if result == "Percentage" {
		typeVote = 2
	}

	validateAmount := func(input string) error {
		_, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return errors.New("invalid number")
		}
		return nil
	}

	promptAmount := promptui.Prompt{
		Label:    "Votes per participant (usually 1)",
		Validate: validateAmount,
	}

	amountStr, err := promptAmount.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	amount64, _ := strconv.ParseInt(amountStr, 10, 64)
	amountPerParticipant = uint32(amount64)

	validateParticipants := func(input string) error {
		pkeyStrings := strings.Split(input, " ")
		for _, pkeyStr := range pkeyStrings {
			pkey, err := hex.DecodeString(pkeyStr)
			if err != nil {
				return errors.New("invalid hex")
			}
			if len(pkey) != evote.PKEY_SIZE {
				return errors.New("invalid pkey size")
			}
		}
		if len(pkeyStrings) == 0 {
			return errors.New("empty")
		}
		return nil
	}

	promptParticipants := promptui.Prompt{
		Label:    "List participants",
		Validate: validateParticipants,
	}

	partisipantsStr, err := promptParticipants.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	pkeyStrings := strings.Split(partisipantsStr, " ")
	outputs := make(map[[evote.PKEY_SIZE]byte]uint32)

	for _, pkeyStr := range pkeyStrings {
		var pkey [evote.PKEY_SIZE]byte
		pkeySlice, _ := hex.DecodeString(pkeyStr)
		copy(pkey[:], pkeySlice)
		outputs[pkey] = amountPerParticipant
	}
	utxos, err := n.GetUtxosByPkey(keys.PubkeyByte)
	if err != nil {
		if retryQuestion(n) {
			send(keys, n)
		}
	}
	var tx evote.Transaction
	retCode := tx.CreateTrans(utxos, outputs, evote.ZERO_ARRAY_HASH, keys)
	tx.TypeVote = typeVote
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
