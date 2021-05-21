package main

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

// TODO: переименовать в РПЗ search_key в scan_key
type Part struct {
	scanKey  *ecdsa.PrivateKey
	spendKey *ecdsa.PrivateKey
}

func generateOrDie() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return key
}

func NewParticipant() *Part {
	return &Part{
		scanKey:  generateOrDie(),
		spendKey: generateOrDie(),
	}
}

func main() {
	curve := crypto.S256()
	receiver := NewParticipant()

	// sender generates ephemeral key
	senderEphemeralPrv := generateOrDie()
	senderEphemeralPkey := senderEphemeralPrv.PublicKey
	sharedSecret, _ := curve.ScalarMult(receiver.scanKey.X, receiver.scanKey.Y, senderEphemeralPrv.D.Bytes())
	sharedSecretHash := crypto.Keccak256(sharedSecret.Bytes())
	fmt.Println("shared secret generated by sender", sharedSecretHash)
	sharedKey, err := crypto.ToECDSA(sharedSecretHash)
	if err != nil {
		panic(fmt.Errorf("sender cannot create shred key %v", err))
	}
	ephemeralDestinationX, ephemeralDestinationY := curve.Add(
		sharedKey.X,
		sharedKey.Y,
		receiver.spendKey.PublicKey.X,
		receiver.spendKey.PublicKey.Y,
	)
	// this public key is generated entirely by sender
	ephemeralDestinationPkey := ecdsa.PublicKey{
		X:     ephemeralDestinationX,
		Y:     ephemeralDestinationY,
		Curve: curve,
	}

	// receiver generates shared secret
	sharedSecretReceiver, _ := curve.ScalarMult(senderEphemeralPkey.X, senderEphemeralPkey.Y, receiver.scanKey.D.Bytes())
	sharedSecretReceiverHash := crypto.Keccak256(sharedSecretReceiver.Bytes())
	fmt.Println("share secret generated by receiver", sharedSecretReceiverHash)
	sharedKeyReceiver, err := crypto.ToECDSA(sharedSecretReceiverHash)
	if err != nil {
		panic(fmt.Errorf("receiver cannot create shred key %v", err))
	}
	//ephemeralDestinationReceiverX, ephemeralDestinationReceiverY := curve.Add(
	//	sharedKeyReceiver.X,
	//	sharedKeyReceiver.Y,
	//	receiver.spendKey.PublicKey.X,
	//	receiver.spendKey.PublicKey.Y,
	//)
	//ephemeralDestinationPkeyReceiver := ecdsa.PublicKey{
	//	X:     ephemeralDestinationReceiverX,
	//	Y:     ephemeralDestinationReceiverY,
	//	Curve: curve,
	//}
	//if ephemeralDestinationPkeyReceiver.Equal(ephemeralDestinationPkey) {
	//	fmt.Println("success: ephemeral destination public keys are equal")
	//} else {
	//	fmt.Println("error: ephemeral destination public keys are different")
	//	fmt.Println(ephemeralDestinationPkeyReceiver, ephemeralDestinationPkey)
	//}
	ephemeralDestinationD := new(big.Int).Mod(new(big.Int).Add(sharedKeyReceiver.D, receiver.spendKey.D), curve.Params().N)
	fmt.Println("len ephemeralDestinationD", len(ephemeralDestinationD.Bytes()))
	ephemeralDestinationKey, err := crypto.ToECDSA(ephemeralDestinationD.Bytes())
	if err != nil {
		panic(fmt.Errorf("receiver cannot create private key for ephemeral destination %v", err))
	}
	fmt.Println("=== keys generated by sender and derived from private key, generated by sender ===")
	fmt.Println(ephemeralDestinationPkey)
	fmt.Println(ephemeralDestinationKey.PublicKey)
	data := crypto.Keccak256([]byte("lol"))
	sig, err := crypto.Sign(data, ephemeralDestinationKey)
	if err != nil {
		panic(fmt.Errorf("error during signing message"))
	}
	sigCorrect := crypto.VerifySignature(crypto.FromECDSAPub(&ephemeralDestinationPkey), data, sig[:crypto.SignatureLength-1])
	if sigCorrect {
		fmt.Println("SUCCESS: signature matches ephemeral destination public key")
	} else {
		fmt.Println("ERROR: signature does not match ephemeral destination public key")
	}
}
