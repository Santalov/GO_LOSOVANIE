package evote

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
)

type CryptoKeysData struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	PkeyByte   [PkeySize]byte
}

func Hash(data []byte) []byte {
	return crypto.Keccak256(data)
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

// Sign produced signature is in the [R || S || V] format where V is 0 or 1.
func (keys *CryptoKeysData) Sign(data []byte) []byte {
	sig, err := crypto.Sign(Hash(data), keys.PrivateKey)
	if err != nil {
		panic(fmt.Errorf("error during signing: %v", err))
	}
	return sig
}

// VerifyData expects data to be prepared, i.e to be in exact same form, as when passing into Sign function
// signature is expected to be the same, as produced by Sign function, in the [R || S || V] format,
// but V byte is not actually used
func VerifyData(data, signature []byte, pkey [PkeySize]byte) bool {
	if len(signature) != SigSize {
		panic(fmt.Errorf("signature is expected to be %v bytes long, but got %v", SigSize, len(signature)))
	}
	return crypto.VerifySignature(pkey[:], Hash(data), signature[:SigSize-1])
}
