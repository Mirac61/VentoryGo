package invoice

import (
	"fmt"
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

func (n Numbering) Format(year, counter int) string {
	return fmt.Sprintf("%s-%d-%04d", n.Prefix, year, counter)
}

func NewNumbering(prefix, timezone string) (Numbering, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Numbering{}, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}
	return Numbering{Prefix: prefix, Location: location}, nil
}
