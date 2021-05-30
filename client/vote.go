package main

import (
	"GO_LOSOVANIE/evote"
	"GO_LOSOVANIE/evote/golosovaniepb"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/manifoldco/promptui"
)

func enterVotingId() (*[evote.HashSize]byte, error) {
	validateHash := func(input string) error {
		pkey, err := hex.DecodeString(input)
		if err != nil {
			return errors.New("invalid hex")
		}
		if len(pkey) != evote.HashSize {
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
	var typeValue [evote.HashSize]byte
	copy(typeValue[:], typeValueSlice)
	return &typeValue, nil
}

func voteMenu(keys *evote.CryptoKeysData, n *evote.Network, typeValue [evote.HashSize]byte) {
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

func vote(keys *evote.CryptoKeysData, n *evote.Network) {
	pkey := keys.PkeyByte
	//priv := keys.PrivateKey
	utxos, err := n.GetUtxosByPkey(pkey[:])
	if retryQuestion(err, n) {
		vote(keys, n)
	}
	txs, err := n.GetTxsByPkey(pkey[:])
	if retryQuestion(err, n) {
		vote(keys, n)
	}
	//key - typeValue, value - outputs sum
	votings := make(map[[evote.HashSize]byte]uint32)

	for _, tx := range txs {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		if err != nil {
			fmt.Println("error during parsing tx", tx.Hash, tx.TxBody)
			return
		}
		if len(body.ValueType) != 0 {
			votings[evote.SliceToHash(body.ValueType)] = 0
		}
		if body.VoteType != 0 {
			votings[evote.SliceToHash(tx.Hash)] = 0
		}
	}

	for _, utxo := range utxos {
		if len(utxo.ValueType) != 0 {
			votings[evote.SliceToHash(utxo.ValueType)] += utxo.Value
		}
	}

	options := make([]string, 2)
	options[0] = "Create voting"
	options[1] = "Enter voting id"
	optionToId := make(map[string][evote.HashSize]byte)
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
		typeValue, err := enterVotingId()
		if err == nil {
			voteMenu(keys, n, *typeValue)
		}
	} else {
		voteMenu(keys, n, optionToId[result])
	}
}
