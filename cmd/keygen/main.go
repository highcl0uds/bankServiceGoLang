package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func main() {
	entity, err := openpgp.NewEntity("BankService", "", "bank@example.com", &packet.Config{
		RSABits: 2048,
		Time:    func() time.Time { return time.Now() },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate key: %v\n", err)
		os.Exit(1)
	}

	pubKey, err := exportKey(false, entity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "export public key: %v\n", err)
		os.Exit(1)
	}
	privKey, err := exportKey(true, entity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "export private key: %v\n", err)
		os.Exit(1)
	}

	hmacRaw := make([]byte, 32)
	rand.Read(hmacRaw)
	jwtRaw := make([]byte, 32)
	rand.Read(jwtRaw)

	fmt.Printf("JWT_SECRET=%s\n", hex.EncodeToString(jwtRaw))
	fmt.Printf("HMAC_SECRET=%s\n", hex.EncodeToString(hmacRaw))
	fmt.Printf("PGP_PUBLIC_KEY=%s\n", pubKey)
	fmt.Printf("PGP_PRIVATE_KEY=%s\n", privKey)
}

func exportKey(private bool, entity *openpgp.Entity) (string, error) {
	var buf bytes.Buffer
	blockType := "PGP PUBLIC KEY BLOCK"
	if private {
		blockType = "PGP PRIVATE KEY BLOCK"
	}
	w, err := armor.Encode(&buf, blockType, nil)
	if err != nil {
		return "", err
	}
	if private {
		err = entity.SerializePrivate(w, nil)
	} else {
		err = entity.Serialize(w)
	}
	if err != nil {
		return "", err
	}
	w.Close()
	return buf.String(), nil
}
