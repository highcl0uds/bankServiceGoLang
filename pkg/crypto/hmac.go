package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

func ComputeHMAC(data string, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func VerifyHMAC(data string, mac []byte, secret []byte) bool {
	expected := ComputeHMAC(data, secret)
	return hmac.Equal(expected, mac)
}
