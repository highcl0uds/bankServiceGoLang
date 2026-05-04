package crypto

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/big"
	"strconv"
)

func GenerateCardNumber(prefix string) (string, error) {
	number := prefix
	for len(number) < 15 {
		n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		number += strconv.FormatInt(n.Int64(), 10)
	}
	checkDigit := luhnCheckDigit(number)
	return fmt.Sprintf("%s%d", number, checkDigit), nil
}

func luhnCheckDigit(partial string) int {
	sum := 0
	nDigits := len(partial) + 1
	parity := nDigits % 2
	for i := 0; i < len(partial); i++ {
		digit, _ := strconv.Atoi(string(partial[i]))
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	remainder := sum % 10
	if remainder == 0 {
		return 0
	}
	return 10 - remainder
}
