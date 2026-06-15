package api

import "encoding/json"

type Nullable[T any] struct {
	Value   T    `json:"-"`
	Defined bool `json:"-"`
	Valid   bool `json:"-"`
}

func (n Nullable[T]) IsZero() bool {
	return !n.Valid
}

// UnmarshalJSON implements [json.Unmarshaler].
func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Defined = true
		return nil
	}
	if err := json.Unmarshal(b, &n.Value); err != nil {
		return err
	}
	n.Defined = true
	n.Valid = true
	return nil
}

var _ json.Unmarshaler = &Nullable[int]{}
