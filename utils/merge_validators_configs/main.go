package main

import (
	"GO_LOSOVANIE/evote"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// outputs single config with validators configuration, accepted by GO_LOSOVANIE app
// tendermint configuration MUST be created by tendermint testnet command and pre-processed by configurate.sh

var tendermintDir = flag.String("t", "", "directory, generated by tendermint testnet command")

type GenesisValidatorJson struct {
	Address string `json:"address"`
	PubKey  struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"pub_key"`
}

type GenesisJson struct {
	Validators []*GenesisValidatorJson `json:"validators"`
}

// not using tendermint types, cos it requires whole tendermint to build, and i just need to read some strings from json

func main() {
	flag.Parse()
	if *tendermintDir == "" {
		fmt.Println(
			"Usage: go run main.go -t=<directory generated by tendermint testnet command>",
		)
		os.Exit(1)
	}
	if !strings.HasSuffix(*tendermintDir, "/") {
		*tendermintDir += "/"
	}
	var validators []*evote.ValidatorJson
	// config must be generated by tendermint testnet command
	// if that is, genesis file contains all validators addresses and public keys
	genesisData, err := ioutil.ReadFile(*tendermintDir + "node0/config/genesis.json")
	if err != nil {
		panic(err)
	}
	var genesis GenesisJson
	err = json.Unmarshal(genesisData, &genesis)
	if err != nil {
		panic(err)
	}
	for i, v := range genesis.Validators {
		vJson := evote.ValidatorJson{
			TendermintAddress: v.Address,
		}
		path := *tendermintDir + fmt.Sprintf("node%d", i)
		golosovaniePrv, err := evote.LoadPrivateKey(path + "/golosovanie_private_key.json")
		if err != nil {
			panic(err)
		}
		var keys evote.CryptoKeysData
		keys.SetupKeys(golosovaniePrv)
		vJson.GolosovaniePkey = hex.EncodeToString(keys.PkeyByte[:])
		ipAndPortBytes, err := ioutil.ReadFile(path + "/ip_and_port")
		if err != nil {
			panic(err)
		}
		ipAndPort := strings.Trim(string(ipAndPortBytes), "\n\r ")
		vJson.IpAndPort = ipAndPort
		validators = append(validators, &vJson)
	}
	var validatorsSet evote.ValidatorsSetJson
	validatorsSet.Validators = validators
	serialized, err := json.Marshal(validatorsSet)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(serialized))
}
