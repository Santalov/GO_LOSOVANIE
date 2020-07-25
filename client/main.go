package main

import (
	"flag"
	"fmt"
)

var pathToGlobalConf = flag.String("g", "", "path to global config")
var pathToKeyPair = flag.String("k", "", "path to key pair")

func main() {
	flag.Parse()
	if *pathToKeyPair == "" || *pathToGlobalConf == "" {
		fmt.Println("Usage: go run main.go -g=<path to global config> -l=<path to local config>")
		return
	}
}
