package main

import (
	"GO_LOSOVANIE/evote"
	"flag"
	"fmt"
	"time"
)

var pathToGlobalConf = flag.String("g", "", "path to global config")
var pathToLocalConf = flag.String("l", "", "path to config of this validator")

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
	bc := new(evote.Blockchain)
	bc.Setup(lConf.Prv, lConf.Addr, gConf.Validators,
		gConf.BlockAppendTime, gConf.BlockVotingTime, gConf.JustWaitingTime, 10*time.Second,
		evote.ZERO_ARRAY_HASH)
	bc.Start()
}
