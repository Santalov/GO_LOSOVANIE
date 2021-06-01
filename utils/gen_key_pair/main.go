package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
)

const PkeySize = 33

type CryptoKeysData struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	PkeyByte   [PkeySize]byte
}

func (keys *CryptoKeysData) SetupKeys(prv []byte) {
	privateKey, err := crypto.ToECDSA(prv)
	if err != nil {
		panic(fmt.Errorf("error during coverting bytes to ecdsa key: %v", err))
	}
	keys.PrivateKey = privateKey
	keys.PublicKey = &ecdsa.PublicKey{
		X:     privateKey.X,
		Y:     privateKey.Y,
		Curve: privateKey.Curve,
	}
	compressedPkey := crypto.CompressPubkey(keys.PublicKey)
	if len(compressedPkey) != PkeySize {
		panic(
			fmt.Errorf(
				"error during compressing public key, expected exactly %d bytes, but got %v",
				PkeySize,
				len(compressedPkey),
			),
		)
	}
	copy(keys.PkeyByte[:], compressedPkey)
}

type PrivateKeyJson struct {
	Pkey string `json:"pkey"`
	Prv  string `json:"prv"`
}

func main() {
	prv, _ := crypto.GenerateKey()
	var keys CryptoKeysData
	keys.SetupKeys(crypto.FromECDSA(prv))
	keyPair := PrivateKeyJson{
		Pkey: hex.EncodeToString(keys.PkeyByte[:]),
		Prv:  hex.EncodeToString(crypto.FromECDSA(keys.PrivateKey)),
	}
	data, err := json.Marshal(keyPair)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
