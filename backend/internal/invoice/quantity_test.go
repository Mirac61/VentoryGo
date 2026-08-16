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

func TestQuantityTotalAtPriceEdgeCases(t *testing.T) {
	maxQuantity := Quantity(math.MaxInt64 - quantityRoundingOffset)

	tests := []struct {
		name     string
		quantity Quantity
		price    Money
		want     Money
		wantErr  error
	}{
		{
			name:     "largest quantity that still fits after rounding",
			quantity: maxQuantity,
			price:    1,
			want:     Money(math.MaxInt64 / quantityScale),
		},
		{
			name:     "quantity one unit larger overflows",
			quantity: maxQuantity + 1,
			price:    1,
			wantErr:  ErrInvalidInput,
		},
		{
			// Negative quantities/prices never reach TotalAtPrice through the
			// API (validateInvoiceData rejects them); this only pins the type
			// behavior. Note: negative results truncate toward zero.
			name:     "negative quantity",
			quantity: Quantity(-1500),
			price:    100,
			want:     -149,
		},
		{
			name:     "zero quantity",
			quantity: 0,
			price:    50,
			want:     0,
		},
		{
			name:     "zero price",
			quantity: Quantity(1500),
			price:    0,
			want:     0,
		},
		{
			name:     "negative price",
			quantity: Quantity(1500),
			price:    -100,
			want:     -149,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.quantity.TotalAtPrice(tt.price)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
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

func TestQuantityJSONEdgeCases(t *testing.T) {
	t.Run("null is rejected", func(t *testing.T) {
		var got Quantity
		assert.Error(t, json.Unmarshal([]byte("null"), &got))
	})

	t.Run("empty string is rejected", func(t *testing.T) {
		var got Quantity
		assert.Error(t, json.Unmarshal([]byte(`""`), &got))
	})

	t.Run("max representable value round trips", func(t *testing.T) {
		var got Quantity
		require.NoError(t, json.Unmarshal([]byte("9223372036854775.807"), &got))
		assert.Equal(t, Quantity(math.MaxInt64), got)

		encoded, err := json.Marshal(got)
		require.NoError(t, err)
		assert.Equal(t, "9223372036854775.807", string(encoded))
	})

	t.Run("value beyond int64 is rejected", func(t *testing.T) {
		var got Quantity
		assert.Error(t, json.Unmarshal([]byte("9223372036854775.808"), &got))
	})

	t.Run("negative value round trips", func(t *testing.T) {
		var got Quantity
		require.NoError(t, json.Unmarshal([]byte("-1.5"), &got))
		assert.Equal(t, Quantity(-1500), got)

		encoded, err := json.Marshal(got)
		require.NoError(t, err)
		assert.Equal(t, "-1.5", string(encoded))
	})

	t.Run("zero round trips", func(t *testing.T) {
		var got Quantity
		require.NoError(t, json.Unmarshal([]byte("0"), &got))
		assert.Equal(t, Quantity(0), got)

		encoded, err := json.Marshal(got)
		require.NoError(t, err)
		assert.Equal(t, "0", string(encoded))
	})
}
