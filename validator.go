package main

import (
	"GO_LOSOVANIE/evote"
	"flag"
	"fmt"
	abciserver "github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var dbPortFlag = flag.String(
	"p",
	strconv.Itoa(5432),
	"port to connect to database",
)

var pathToValidatorsKeys = flag.String(
	"v",
	"",
	"file with map from tendermint validator pkey to golosovanie pkey",
)

var pathToPrivateKey = flag.String(
	"k",
	"",
	"file with validator's private keys for making chain transactions",
)

var socketAddr = flag.String(
	"s",
	"",
	"unix domain socket address",
)

func main() {
	flag.Parse()
	if *pathToValidatorsKeys == "" || *pathToPrivateKey == "" || *socketAddr == "" {
		fmt.Println(
			"Usage: go run main.go -v=<path to validators keys map> -k=<path to private key> -p=<database port> -s=<unix socket addr for ABCI>",
		)
		os.Exit(1)
	}
	validators, err := evote.LoadValidators(*pathToValidatorsKeys)
	if err != nil {
		panic(err)
	}
	prv, err := evote.LoadPrivateKey(*pathToPrivateKey)
	if err != nil {
		panic(err)
	}
	dbPort, err := strconv.Atoi(*dbPortFlag)
	if err != nil {
		panic(err)
	}
	bc := evote.NewBlockchainApp(prv, validators, dbPort, "0.0.1", 1)
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	fmt.Println("starting server on addr", *socketAddr)
	server := abciserver.NewSocketServer(*socketAddr, bc)
	server.SetLogger(logger)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error starting socket server: %v", err)
		os.Exit(1)
	}
	defer server.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
