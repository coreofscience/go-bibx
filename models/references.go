package models

import (
	"encoding/json"
	"fmt"
)

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
	bytes, err := json.Marshal(toMarshal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal References: %w", err)
	}
	return bytes, nil
}
