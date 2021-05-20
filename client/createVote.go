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

func createVoting(keys *evote.CryptoKeysData, n *evote.Network) {
	var typeVote, amountPerParticipant, duration uint32
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
		typeVote = evote.OneVoteType
	} else if result == "Percentage" {
		typeVote = evote.PercentVoteType
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

	promptDuration := promptui.Prompt{
		Label:    "Voting duration (seconds)",
		Validate: validateAmount,
	}

	durationStr, err := promptDuration.Run()
	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	duration64, _ := strconv.ParseInt(durationStr, 10, 32)
	duration = uint32(duration64)

	validateParticipants := func(input string) error {
		pkeyStrings := strings.Split(input, " ")
		for _, pkeyStr := range pkeyStrings {
			pkey, err := hex.DecodeString(pkeyStr)
			if err != nil {
				return errors.New("invalid hex")
			}
			if len(pkey) != evote.PkeySize {
				return errors.New("invalid pkey size")
			}
		}
		if len(pkeyStrings) == 0 {
			return errors.New("empty")
		}
		return nil
	}

	promptParticipants := promptui.Prompt{
		Label: "List participants",
		//Validate: validateParticipants,
	}

	partisipantsStr, err := promptParticipants.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	// так как Validate на 5 строчек выше вызывал экран ошибок,
	// валидация перенесена вниз
	err = validateParticipants(partisipantsStr)

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return
	}

	pkeyStrings := strings.Split(partisipantsStr, " ")
	outputs := make(map[[evote.PkeySize]byte]uint32)

	for _, pkeyStr := range pkeyStrings {
		var pkey [evote.PkeySize]byte
		pkeySlice, _ := hex.DecodeString(pkeyStr)
		copy(pkey[:], pkeySlice)
		outputs[pkey] = amountPerParticipant
	}
	var utxos []*evote.UTXO
	for {
		utxos, err = n.GetUtxosByPkey(keys.PubkeyByte)
		if !retryQuestion(err, n) {
			break
		}
	}
	var tx evote.Transaction
	retCode := tx.CreateTrans(utxos, outputs, evote.ZeroArrayHash, keys, typeVote, duration, false)
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
