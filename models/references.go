package models

import (
	"encoding/json"
	"fmt"
)

// References prevents circular references by using a slice of Work pointers.
type References []*Work

// MarshalJSON implements the json.Marshaler interface.
func (r References) MarshalJSON() ([]byte, error) {
	toMarshal := make([]*struct {
		*Work
		References any `json:"references,omitempty"`
	}, len(r))
	for i, a := range r {
		toMarshal[i] = &struct {
			*Work
			References any `json:"references,omitempty"`
		}{
			Work:       a,
			References: nil,
		}
	}
	bytes, err := json.Marshal(toMarshal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal References: %w", err)
	}
	return bytes, nil
}
