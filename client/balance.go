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
	coinsBalance, votesBalance := calcBalance(utxos, evote.ZERO_ARRAY_HASH)
	fmt.Printf(
		"publicKey:     %s\nprivateKey:    %s\ncoins balance: %v\nvotes balance: %v\ntrans num:     %v\n",
		pkeyHex(pkey),
		bToHex(priv.Raw()),
		coinsBalance,
		votesBalance,
		len(txs),
	)
}
