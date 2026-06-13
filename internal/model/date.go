package model

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"
)

type Date struct {
	sql.NullTime `json:"-"`
}

func (d Date) IsZero() bool {
	return !d.Valid
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	s := time.Time(d.Time).Format(`"2006-01-02"`)
	return []byte(s), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	d.Time = t
	d.Valid = true
	return nil
}

var _ json.Marshaler = &Date{}
var _ json.Unmarshaler = &Date{}
