package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
)

func enterVotingId(keys *evote.CryptoKeysData, n *Network) (*[evote.HASH_SIZE]byte, error) {
	validateHash := func(input string) error {
		pkey, err := hex.DecodeString(input)
		if err != nil {
			return errors.New("invalid hex")
		}
		if len(pkey) != evote.HASH_SIZE {
			return errors.New("invalid txid size")
		}
		return nil
	}

	promptReceiver := promptui.Prompt{
		Label:    "Enter voting id (txid of the transaction creating vote)",
		Validate: validateHash,
	}

	typeValueStr, err := promptReceiver.Run()

	if err != nil {
		fmt.Printf("Fail: %v\n", err)
		return nil, err
	}

	typeValueSlice, _ := hex.DecodeString(typeValueStr)
	var typeValue [evote.HASH_SIZE]byte
	copy(typeValue[:], typeValueSlice)
	return &typeValue, nil
}

func voteMenu(keys *evote.CryptoKeysData, n *Network, typeValue [evote.HASH_SIZE]byte) {
	prompt := promptui.Select{
		Label: "Select vote type",
		Items: []string{"Info", "See results", "Send vote"},
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Fail %v\n", err)
		return
	}

	if result == "Info" {
		voteInfo(keys, n, typeValue)
	} else if result == "See results" {
		voteResults(keys, n, typeValue)
	} else if result == "Send vote" {
		sendVote(keys, n, typeValue)
	}
}

func vote(keys *evote.CryptoKeysData, n *Network) {
	pkey := keys.PubkeyByte
	//priv := keys.PrivateKey
	utxos, err := n.GetUtxosByPkey(pkey)
	if err != nil {
		if retryQuestion(n) {
			vote(keys, n)
		}
	}
	txs, err := n.GetTxsByPkey(pkey)
	if err != nil {
		if retryQuestion(n) {
			vote(keys, n)
		}
	}
	//key - typeValue, value - outputs sum
	votings := make(map[[evote.HASH_SIZE]byte]uint32)

	for _, tx := range txs {
		if tx.TypeValue != evote.ZERO_ARRAY_HASH {
			votings[tx.TypeValue] = 0
		}
		if tx.TypeVote != 0 {
			var typeValue [evote.HASH_SIZE]byte
			copy(typeValue[:], evote.Hash(tx.ToBytes()))
			votings[typeValue] = 0
		}
	}

	for _, utxo := range utxos {
		if utxo.TypeValue != evote.ZERO_ARRAY_HASH {
			votings[utxo.TypeValue] += utxo.Value
		}
	}

	options := make([]string, 2)
	options[0] = "Create voting"
	options[1] = "Enter voting id"
	optionToId := make(map[string][evote.HASH_SIZE]byte)
	for voteId, balance := range votings {
		option := fmt.Sprintf("id: %v  votes: %v", bToHex(voteId[:]), balance)
		options = append(options, option)
		optionToId[option] = voteId
	}
	prompt := promptui.Select{
		Label: "Select vote type",
		Items: options,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Fail %v\n", err)
		return
	}

	if result == options[0] {
		createVoting(keys, n)
	} else if result == options[1] {
		typeValue, err := enterVotingId(keys, n)
		if err == nil {
			voteMenu(keys, n, *typeValue)
		}
	} else {
		voteMenu(keys, n, optionToId[result])
	}
}
