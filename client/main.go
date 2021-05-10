package main

import (
	"GO_LOSOVANIE/evote"
	"errors"
	"flag"
	"fmt"
	"github.com/manifoldco/promptui"
)

var pathToValidatorsConfig = flag.String("v", "", "path to validators config")
var pathToKeyPair = flag.String("k", "", "path to key pair")

const (
	BALANCE      = "balance"
	TRANSACTIONS = "transactions"
	SEND         = "send"
	FAUCET       = "faucet"
	VOTE         = "vote"
)

func main() {
	flag.Parse()
	if *pathToKeyPair == "" || *pathToValidatorsConfig == "" {
		fmt.Println("Usage: go run main.go -v=<path to validators config> -k=<path to key pair>")
		return
	}
	prv, err := evote.LoadPrivateKey(*pathToKeyPair)
	if err != nil {
		panic(err)
	}
	validators, err := evote.LoadValidators(*pathToValidatorsConfig)
	if err != nil {
		panic(err)
	}
	var n evote.Network
	hosts := make([]string, len(validators))
	for i, v := range validators {
		hosts[i] = v.IpAndPort
	}
	var keys evote.CryptoKeysData
	keys.SetupKeys(prv)
	n.Init(hosts)
	n.PingAll()
	fmt.Println("available commands: " +
		BALANCE + ", " + TRANSACTIONS + ", " + SEND + ", " + FAUCET + ", " + VOTE)

	validate := func(input string) error {
		if input == BALANCE || input == TRANSACTIONS ||
			input == SEND || input == FAUCET || input == VOTE {
			return nil
		} else {
			return errors.New("invalid command")
		}
	}

	for {
		prompt := promptui.Prompt{
			Label:    "Command",
			Validate: validate,
		}

		result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Command failed %v\n", err)
			return
		}

		switch result {
		case BALANCE:
			balance(&keys, &n)
		case TRANSACTIONS:
			transactions(&keys, &n)
		case SEND:
			send(&keys, &n)
		case FAUCET:
			faucet(&keys, &n)
		case VOTE:
			vote(&keys, &n)
		}
	}

}
