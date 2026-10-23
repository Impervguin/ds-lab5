package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

const DateLayout = "2006-01-02"

type Date struct {
	time.Time
}

func ParseDate(raw string) (Date, error) {
	parsed, err := time.Parse(DateLayout, raw)
	if err != nil {
		return Date{}, fmt.Errorf("%q is not a date in the %s format", raw, DateLayout)
	}
	return Date{Time: parsed}, nil
}

func (d Date) String() string {
	return d.Format(DateLayout)
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsed, err := ParseDate(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
