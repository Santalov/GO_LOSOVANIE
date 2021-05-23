package main

import (
	"GO_LOSOVANIE/evote"
	"fmt"
)

func balance(keys *evote.CryptoKeysData, n *evote.Network) {
	pkey := keys.PkeyByte
	utxos, err := n.GetUtxosByPkey(pkey)
	if retryQuestion(err, n) {
		balance(keys, n)
	}
	txs, err := n.GetTxsByPkey(pkey)
	if retryQuestion(err, n) {
		balance(keys, n)
	}
	coinsBalance, votesBalance := calcBalance(utxos, evote.ZeroArrayHash)
	fmt.Printf(
		"publicKey:     %s\ncoins balance: %v\nvotes balance: %v\ntrans num:     %v\n",
		pkeyHex(pkey),
		coinsBalance,
		votesBalance,
		len(txs),
	)
}
