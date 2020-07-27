package main

import (
	"GO_LOSOVANIE/evote"
	"errors"
	"flag"
	"fmt"
	"github.com/manifoldco/promptui"
)

var pathToGlobalConf = flag.String("g", "", "path to global config")
var pathToKeyPair = flag.String("k", "", "path to key pair")

const (
	BALANCE       = "balance"
	TRANSACTIONS  = "transactions"
	SEND          = "send"
	FAUCET        = "faucet"
	CREATE_VOTING = "createVoting"
)

func main() {
	flag.Parse()
	if *pathToKeyPair == "" || *pathToGlobalConf == "" {
		fmt.Println("Usage: go run main.go -g=<path to global config> -l=<path to local config>")
		return
	}
	gConf, keyPair, err := evote.LoadConfig(*pathToGlobalConf, *pathToKeyPair)
	if err != nil {
		panic(err)
	}

	var n Network
	hosts := make([]string, len(gConf.Validators))
	for i, v := range gConf.Validators {
		hosts[i] = v.Addr
	}
	var keys evote.CryptoKeysData
	keys.SetupKeys(keyPair.Prv)
	n.Init(hosts)
	fmt.Println("available commands: " +
		BALANCE + ", " + TRANSACTIONS + ", " + SEND + ", " + FAUCET + ", " + CREATE_VOTING)

	validate := func(input string) error {
		if input == BALANCE || input == TRANSACTIONS ||
			input == SEND || input == FAUCET || input == CREATE_VOTING {
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
		case CREATE_VOTING:
			createVoting(&keys, &n)
		}
	}

}
