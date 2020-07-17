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
	fmt.Printf("%+v \n %+v\n", gConf, lConf)
	bc := new(evote.Blockchain)
	bc.Setup(lConf.Prv, gConf.Validators, time.Now(), gConf.NextLeaderPeriod, gConf.BlockAppendTime, evote.ZERO_ARRAY_HASH)
}
