package invoice

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	quantityScale          int64 = 1000
	quantityRoundingOffset       = quantityScale / 2
)

type Quantity int64

func (q Quantity) TotalAtPrice(unitPrice Money) Money {
	scaledAmount := int64(q) * int64(unitPrice)
	roundedAmount := (scaledAmount + quantityRoundingOffset) / quantityScale
	return Money(roundedAmount)
}

func (q Quantity) MarshalJSON() ([]byte, error) {
	scaled := int64(q)
	sign := ""
	if scaled < 0 {
		sign = "-"
		scaled = -scaled
	}

	whole := scaled / quantityScale
	fraction := scaled % quantityScale
	if fraction == 0 {
		return []byte(sign + strconv.FormatInt(whole, 10)), nil
	}

	decimal := strconv.FormatInt(fraction+quantityScale, 10)[1:]
	decimal = strings.TrimRight(decimal, "0")
	return []byte(sign + strconv.FormatInt(whole, 10) + "." + decimal), nil
}

func (q *Quantity) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" || strings.ContainsAny(raw, "eE") {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}

	sign := int64(1)
	if strings.HasPrefix(raw, "-") {
		sign = -1
		raw = raw[1:]
	} else if strings.HasPrefix(raw, "+") {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}

	parts := strings.Split(raw, ".")
	if len(parts) > 2 || (len(parts) == 1 && parts[0] == "") || (len(parts) == 2 && parts[0] == "" && parts[1] == "") {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}

	fractionText := ""
	if len(parts) == 2 {
		fractionText = parts[1]
	}
	if len(fractionText) > 3 || (fractionText != "" && strings.Trim(fractionText, "0123456789") != "") {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}
	for len(fractionText) < 3 {
		fractionText += "0"
	}
	fraction, err := strconv.ParseInt(fractionText, 10, 64)
	if err != nil {
		return fmt.Errorf("quantity must be a decimal number with at most three decimal places")
	}

	if whole > (int64(^uint64(0)>>1)-fraction)/quantityScale {
		return fmt.Errorf("quantity is too large")
	}
	*q = Quantity(sign * (whole*quantityScale + fraction))
	return nil
}

var _ json.Marshaler = Quantity(0)
var _ json.Unmarshaler = (*Quantity)(nil)
