package invoice

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scaledQuantity(units int64) Quantity {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.quantity.TotalAtPrice(tt.price)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQuantityTotalAtPriceRejectsOverflow(t *testing.T) {
	_, err := Quantity(math.MaxInt64).TotalAtPrice(2)

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestQuantityJSONRoundTrip(t *testing.T) {
	tests := []struct {
		json string
		want Quantity
	}{
		{json: "1.5", want: Quantity(1500)},
		{json: "0.75", want: Quantity(750)},
		{json: "1.005", want: Quantity(1005)},
		{json: "2", want: Quantity(2000)},
	}

	for _, tt := range tests {
		t.Run(tt.json, func(t *testing.T) {
			var got Quantity
			require.NoError(t, json.Unmarshal([]byte(tt.json), &got))
			assert.Equal(t, tt.want, got)

			encoded, err := json.Marshal(got)
			require.NoError(t, err)
			assert.Equal(t, tt.json, string(encoded))
		})
	}
}

func TestQuantityJSONRejectsMoreThanThreeDecimalPlaces(t *testing.T) {
	var got Quantity
	err := json.Unmarshal([]byte("1.0001"), &got)

	assert.Error(t, err)
}
