package invoice

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	quantityScale          int64 = 1000
	quantityRoundingOffset       = quantityScale / 2
)

// Quantity stores a decimal quantity as a fixed-point integer scaled by 1000.
// For example, 1.5 is stored as 1500.
type Quantity int64

// TotalAtPrice returns the line total in cents, rounded half-up.
func (q Quantity) TotalAtPrice(unitPrice Money) (Money, error) {
	scaledQuantity := int64(q)
	unitPriceCents := int64(unitPrice)

	if scaledQuantity > 0 && unitPriceCents > 0 &&
		scaledQuantity > (math.MaxInt64-quantityRoundingOffset)/unitPriceCents {
		return 0, ErrInvalidInput
	}

	return Money((scaledQuantity*unitPriceCents + quantityRoundingOffset) / quantityScale), nil
}

func (q Quantity) MarshalJSON() ([]byte, error) {
	value := int64(q)
	negative := value < 0
	if negative {
		value = -value
	}

	whole := value / quantityScale
	fraction := value % quantityScale

	out := strconv.FormatInt(whole, 10)
	if fraction != 0 {
		out += "." + strings.TrimRight(strconv.FormatInt(fraction+quantityScale, 10)[1:], "0")
	}
	if negative {
		out = "-" + out
	}
	return []byte(out), nil
}

func (q *Quantity) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if input == "" || input == "null" {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}

	sign := int64(1)
	if strings.HasPrefix(input, "-") {
		sign = -1
		input = input[1:]
	}

	wholePart, fractionPart := input, ""
	if dot := strings.IndexByte(input, '.'); dot >= 0 {
		wholePart, fractionPart = input[:dot], input[dot+1:]
	}

	if len(fractionPart) > 3 || strings.Trim(wholePart+fractionPart, "0123456789") != "" {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}
	for len(fractionPart) < 3 {
		fractionPart += "0"
	}

	whole, errWhole := strconv.ParseInt(wholePart, 10, 64)
	fraction, errFrac := strconv.ParseInt(fractionPart, 10, 64)
	if errWhole != nil || errFrac != nil || whole > (math.MaxInt64-fraction)/quantityScale {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}

	*q = Quantity(sign * (whole*quantityScale + fraction))
	return nil
}

var _ json.Marshaler = Quantity(0)
var _ json.Unmarshaler = (*Quantity)(nil)
