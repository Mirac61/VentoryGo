package invoice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundedVAT(t *testing.T) {
	tests := []struct {
		name        string
		net         Money
		ratePercent int64
		want        Money
	}{
		{name: "0,50€ @ 19% rounds up (doc example)", net: 50, ratePercent: 19, want: 10},
		{name: "3x33,33€ netto @ 19% (doc example)", net: 9999, ratePercent: 19, want: 1900},
		{name: "remainder 49 rounds down", net: 1, ratePercent: 49, want: 0},
		{name: "remainder 50 rounds up", net: 1, ratePercent: 50, want: 1},
		{name: "remainder 51 rounds up", net: 1, ratePercent: 51, want: 1},
		{name: "zero net", net: 0, ratePercent: 19, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, RoundedVAT(test.net, test.ratePercent))
		})
	}
}
