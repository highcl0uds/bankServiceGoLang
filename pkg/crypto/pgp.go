package crypto

import (
	"bytes"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

func EncryptPGPArmored(plaintext string, armoredPubKey string) (string, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredPubKey))
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	armorWriter, err := armor.Encode(buf, "PGP MESSAGE", nil)
	if err != nil {
		return "", err
	}
	w, err := openpgp.Encrypt(armorWriter, entityList, nil, nil, nil)
	if err != nil {
		return "", err
	}
	if _, err = w.Write([]byte(plaintext)); err != nil {
		return "", err
	}
	if err = w.Close(); err != nil {
		return "", err
	}
	if err = armorWriter.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func DecryptPGPArmored(armoredCipher string, armoredPrivKey string) (string, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredPrivKey))
	if err != nil {
		return "", err
	}

	block, err := armor.Decode(strings.NewReader(armoredCipher))
	if err != nil {
		return "", err
	}
	md, err := openpgp.ReadMessage(block.Body, entityList, nil, nil)
	if err != nil {
		return "", err
	}
	plaintext, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
