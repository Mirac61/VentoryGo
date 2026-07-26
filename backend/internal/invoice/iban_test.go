package invoice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIban_ValidIbans(t *testing.T) {
	tests := []struct {
		name    string
		country string
		iban    string
	}{
		{name: "Germany", country: "DE", iban: "DE89370400440532013000"},
		{name: "Norway (shortest IBAN, 15 chars)", country: "NO", iban: "NO9386011117947"},
		{name: "United Kingdom", country: "GB", iban: "GB29NWBK60161331926819"},
		{name: "France", country: "FR", iban: "FR1420041010050500013M02606"},
		{name: "Netherlands", country: "NL", iban: "NL91ABNA0417164300"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, validateIban(tt.iban), "expected %s IBAN %q to be valid", tt.country, tt.iban)
		})
	}
}

func TestValidateIban_InvalidChecksum(t *testing.T) {
	valid := "DE89370400440532013000"
	broken := valid[:len(valid)-1] + "1" // last digit tampered, checksum must fail

	assert.False(t, validateIban(broken))
}

func TestValidateIban_RejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		iban string
	}{
		{name: "empty string", iban: ""},
		{name: "too short (14 chars, below IBAN minimum of 15)", iban: "DE893704004405"},
		{name: "too long (35 chars, above IBAN maximum of 34)", iban: strings.Repeat("A", 35)},
		{name: "lowercase letters are not normalized", iban: "de89370400440532013000"},
		{name: "contains spaces (human-readable formatting)", iban: "DE89 3704 0044 0532 0130 00"},
		{name: "contains dashes", iban: "DE89-3704-0044-0532-0130-00"},
		{name: "letter O used instead of digit 0", iban: "DE89370400440532013O00"},
		{name: "only digits, no country code letters", iban: "123456789012345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, validateIban(tt.iban))
		})
	}
}

func TestValidateIban_DoesNotPanicOnBoundaryLengths(t *testing.T) {
	tests := []struct {
		name string
		iban string
	}{
		{name: "exactly 15 chars (minimum)", iban: strings.Repeat("A", 15)},
		{name: "exactly 34 chars (maximum)", iban: strings.Repeat("A", 34)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { validateIban(tt.iban) })
		})
	}
}
