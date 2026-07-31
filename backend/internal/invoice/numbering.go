package invoice

import (
	"fmt"
	"os"
	"time"

	// Das Nummernjahr hängt an einer echten Zeitzone, deshalb die Zonendaten
	// einbetten statt auf tzdata im Container zu hoffen.
	_ "time/tzdata"
)

const (
	defaultNumberPrefix   = "INV"
	defaultNumberTimezone = "Europe/Berlin"
)

type Numbering struct {
	Prefix   string
	Location *time.Location
}

func DefaultNumbering() Numbering {
	location, err := time.LoadLocation(defaultNumberTimezone)
	if err != nil {
		panic(err)
	}
	return Numbering{Prefix: defaultNumberPrefix, Location: location}
}

func NumberingFromEnv() (Numbering, error) {
	numbering := DefaultNumbering()

	if prefix := os.Getenv("INVOICE_NUMBER_PREFIX"); prefix != "" {
		numbering.Prefix = prefix
	}

	if name := os.Getenv("INVOICE_TIMEZONE"); name != "" {
		location, err := time.LoadLocation(name)
		if err != nil {
			return Numbering{}, fmt.Errorf("invalid INVOICE_TIMEZONE %q: %w", name, err)
		}
		numbering.Location = location
	}

	return numbering, nil
}

func (n Numbering) Format(year, counter int) string {
	return fmt.Sprintf("%s-%d-%04d", n.Prefix, year, counter)
}
