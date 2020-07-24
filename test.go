package main

import (
	"GO_LOSOVANIE/evote"
	"flag"
	"fmt"
	"strconv"
	"time"
)

var pathToGlobalConf = flag.String("g", "", "path to global config")
var pathToLocalConf = flag.String("l", "", "path to config of this validator")
var dbPort = flag.String("p", string(5432), "port to connect to database")

func main() {
	flag.Parse()
	if *pathToLocalConf == "" || *pathToGlobalConf == "" {
		fmt.Println("Usage: go run main.go -g=<path to global config> -l=<path to local config>")
		return
	}
	gConf, lConf, err := evote.LoadConfig(*pathToGlobalConf, *pathToLocalConf)
	if err != nil {
		panic(err)
	}
	dbPort, err := strconv.Atoi(*dbPort)
	if err != nil {
		panic(err)
	}
	bc := new(evote.Blockchain)
	bc.Setup(lConf.Prv, lConf.Addr, gConf.Validators,
		gConf.BlockAppendTime, gConf.BlockVotingTime, gConf.JustWaitingTime, 10*time.Second,
		evote.ZERO_ARRAY_HASH, dbPort)
	bc.Start()
}
