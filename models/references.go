package models

import "encoding/json"

// References prevents circular references by using a slice of Article pointers.
type References []*Article

// MarshalJSON implements the json.Marshaler interface.
func (r References) MarshalJSON() ([]byte, error) {
	toMarshal := make([]*struct {
		*Article
		References any `json:"references,omitempty"`
	}, len(r))
	for i, a := range r {
		toMarshal[i] = &struct {
			*Article
			References any `json:"references,omitempty"`
		}{
			Article:    a,
			References: nil,
		}
	}
	return json.Marshal(toMarshal)
}
