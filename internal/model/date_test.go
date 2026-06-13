package model

import (
	"database/sql"
	"testing"
	"time"

	"github.com/aaa2ppp/be"
)

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		value   Date
		want    string
		wantErr bool
	}{
		{
			"valid",
			Date{sql.NullTime{Time: date(2026, 5, 30), Valid: true}},
			`"2026-05-30"`,
			false,
		},
		{
			"not valid",
			Date{sql.NullTime{Time: date(2026, 5, 30), Valid: false}},
			`null`,
			false,
		},
		{
			"zero",
			Date{},
			`null`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.value
			got, gotErr := d.MarshalJSON()
			be.Err(t, gotErr, tt.wantErr)
			be.True(t, got != nil)
			be.Equal(t, string(got), tt.want)
		})
	}
}

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		b       []byte
		want    Date
		wantErr bool
	}{
		{
			"correct",
			[]byte(`"2026-05-30"`),
			Date{sql.NullTime{Time: date(2026, 5, 30), Valid: true}},
			false,
		},
		{
			"incorrect format",
			[]byte(`"is not date"`),
			Date{},
			true,
		},
		{
			"incorrect type",
			[]byte(`123456`),
			Date{},
			true,
		},
		{
			"null",
			[]byte("null"),
			Date{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			gotErr := d.UnmarshalJSON(tt.b)
			be.Err(t, gotErr, tt.wantErr)
			be.Equal(t, d, tt.want)
		})
	}
}
