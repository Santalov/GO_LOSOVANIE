package main

import (
	"GO_LOSOVANIE/evote"
	"fmt"
)

func balance(keys *evote.CryptoKeysData, n *Network) {
	pkey := keys.PubkeyByte
	priv := keys.PrivateKey
	utxos, err := n.GetUtxosByPkey(pkey)
	if err != nil {
		if retryQuestion(n) {
			balance(keys, n)
		}
	}
	txs, err := n.GetTxsByPkey(pkey)
	if err != nil {
		if retryQuestion(n) {
			balance(keys, n)
		}
	}
	fmt.Printf(
		"publicKey:  %s\nprivateKey: %s\nbalance:    %v\ntrans num:  %v\n",
		pkeyHex(pkey),
		bToHex(priv.Raw()),
		calcBalance(utxos),
		len(txs),
	)
}
