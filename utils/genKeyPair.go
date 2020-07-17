package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.cypherpunks.ru/gogost/v4/gost3410"
	"math/big"
)

var curve = gost3410.CurveIdGostR34102001CryptoProAParamSet()

type CryptoKeysData struct {
	privateKey *gost3410.PrivateKey
	publickKey *gost3410.PublicKey
	pubKeyByte [33]byte
}

func (keys *CryptoKeysData) SetupKeys(prv []byte) {
	keys.privateKey, _ = gost3410.NewPrivateKey(curve, gost3410.Mode2001, prv)
	keys.publickKey, _ = keys.privateKey.PublicKey()
	var pkeyX = keys.publickKey.Raw()[:32]
	var tmp = make([]byte, 1)
	if big.NewInt(0).Mod(keys.publickKey.Y, big.NewInt(2)).Uint64() == 0 {
		tmp[0] = 0x02
	} else {
		tmp[0] = 0x03
	}
	copy(keys.pubKeyByte[:], append(tmp, pkeyX[:]...))
}

type KeyPair struct {
	Pkey string `json:"pkey"`
	Prv  string `json:"prv"`
}

func main() {
	prv, _ := gost3410.GenPrivateKey(gost3410.CurveIdGostR34102001CryptoProAParamSet(), gost3410.Mode2001, rand.Reader)
	var keys CryptoKeysData
	keys.SetupKeys(prv.Raw())
	keyPair := KeyPair{
		Pkey: hex.EncodeToString(keys.pubKeyByte[:]),
		Prv:  hex.EncodeToString(keys.privateKey.Raw()),
	}
	data, err := json.Marshal(keyPair)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
