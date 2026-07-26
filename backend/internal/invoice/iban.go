package invoice

import (
	"math/big"
	"strconv"
	"strings"
)

func validateIban(iban string) bool {
	if len(iban) < 15 || len(iban) > 34 {
		return false
	}

	rearranged := iban[4:] + iban[:4]

	var digits strings.Builder
	for _, r := range rearranged {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			digits.WriteString(strconv.Itoa(int(r-'A') + 10))
		default:
			return false
		}
	}

	n, ok := new(big.Int).SetString(digits.String(), 10)
	if !ok {
		return false
	}

	remainder := new(big.Int).Mod(n, big.NewInt(97))
	return remainder.Cmp(big.NewInt(1)) == 0
}
