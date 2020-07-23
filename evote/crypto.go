package evote

import (
	"crypto/rand"
	"fmt"
	"go.cypherpunks.ru/gogost/v4/gost3410"
	"go.cypherpunks.ru/gogost/v4/gost34112012256"
	"math/big"
)

var curve = gost3410.CurveIdGostR34102001CryptoProAParamSet()
var expCoff = big.NewInt(0).Div(big.NewInt(0).Add(curve.P, big.NewInt(1)), big.NewInt(4))

type CryptoKeysData struct {
	privateKey *gost3410.PrivateKey
	publickKey *gost3410.PublicKey
	pubKeyByte [PKEY_SIZE]byte
}

func Hash(data []byte) []byte {
	var res []byte
	var h = gost34112012256.New()
	h.Write(data)
	h.Sum(res)
	return res
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

func (keys *CryptoKeysData) Sign(data []byte) []byte {
	var res, err = keys.privateKey.SignDigest(data, rand.Reader)
	if err != nil {
		fmt.Println("sign error", err)
	}
	return res
}

func (keys *CryptoKeysData) AppendSign(data []byte) []byte {
	res := keys.Sign(append(data, ZERO_ARRAY_SIG[:]...))
	return append(data, res...)
}

func VerifyData(data, signature []byte, pkey [PKEY_SIZE]byte) bool {
	var pkeyX = pkey[1:]
	for i, j := 0, len(pkeyX)-1; i < j; i, j = i+1, j-1 {
		pkeyX[i], pkeyX[j] = pkeyX[j], pkeyX[i]
	}
	var x = big.NewInt(0).SetBytes(pkeyX)
	fx := big.NewInt(0)
	tmp := big.NewInt(0)
	root := big.NewInt(0)
	y := big.NewInt(0)
	fx.Exp(x, big.NewInt(0x03), curve.P)
	fx.Add(fx, curve.B)
	tmp.Mul(curve.A, x)
	fx.Add(fx, tmp)
	fx.Mod(fx, curve.P)
	root.Exp(fx, expCoff, curve.P)
	if pkey[0] == 0x03 {
		if tmp.Mod(root, big.NewInt(2)).Uint64() == 1 {
			y = root
		} else {
			y = root.Neg(root).Mod(root, curve.P)
		}
	} else {
		if tmp.Mod(root, big.NewInt(2)).Uint64() == 0 {
			y = root
		} else {
			y = root.Neg(root).Mod(root, curve.P)
		}
	}
	var key gost3410.PublicKey
	key.C = curve
	key.Mode = gost3410.Mode2001
	key.X = x
	key.Y = y
	res, err := key.VerifyDigest(append(data, ZERO_ARRAY_SIG[:]...), signature)
	if err != nil {
		fmt.Println("verify digest error", err)
	}
	return res
}
