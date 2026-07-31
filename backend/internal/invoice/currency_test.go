package invoice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCurrency_KnownCodes(t *testing.T) {
	tests := []string{"EUR", "USD", "GBP", "CHF", "JPY"}

	for _, code := range tests {
		t.Run(code, func(t *testing.T) {
			assert.True(t, validateCurrency(code))
		})
	}
}

func TestValidateCurrency_RejectsUnknownOrMalformedCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{name: "empty string", code: ""},
		{name: "made-up 3-letter code", code: "XYZ"},
		{name: "lowercase valid code is case-sensitive", code: "eur"},
		{name: "mixed case", code: "Eur"},
		{name: "too short", code: "EU"},
		{name: "too long", code: "EURO"},
		{name: "numeric ISO code instead of alpha", code: "978"}, // 978 is EUR's numeric ISO code, not accepted here
		{name: "currency symbol instead of code", code: "€"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.False(t, validateCurrency(test.code))
		})
	}
}
