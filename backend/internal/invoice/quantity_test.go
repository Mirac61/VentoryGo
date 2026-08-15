package invoice

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quantity(units int64) Quantity {
	return Quantity(units * quantityScale)
}

func TestQuantityTotalAtPrice(t *testing.T) {
	tests := []struct {
		name     string
		quantity Quantity
		price    Money
		want     Money
	}{
		{name: "one and a half units", quantity: Quantity(1500), price: 1000, want: 1500},
		{name: "three quarters of a unit", quantity: Quantity(750), price: 1000, want: 750},
		{name: "rounds half cent up", quantity: Quantity(1005), price: 999, want: 1004},
		{name: "rounds below half cent down", quantity: Quantity(1004), price: 999, want: 1003},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.quantity.TotalAtPrice(test.price))
		})
	}
}

func TestQuantityJSONRoundTrip(t *testing.T) {
	for _, test := range []struct {
		json string
		want Quantity
	}{
		{json: "1.5", want: Quantity(1500)},
		{json: "0.75", want: Quantity(750)},
		{json: "1.005", want: Quantity(1005)},
		{json: "2", want: Quantity(2000)},
	} {
		var got Quantity
		require.NoError(t, json.Unmarshal([]byte(test.json), &got))
		assert.Equal(t, test.want, got)

		encoded, err := json.Marshal(got)
		require.NoError(t, err)
		assert.Equal(t, test.json, string(encoded))
	}
}

func TestQuantityJSONRejectsMoreThanThreeDecimalPlaces(t *testing.T) {
	var got Quantity
	err := json.Unmarshal([]byte("1.0001"), &got)

	assert.Error(t, err)
}
