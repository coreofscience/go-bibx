package utils

import "go.yaml.in/yaml/v4"

// FoldedString is a string type that serializes to YAML using folded style (>).
type FoldedString string

// NewFoldedString creates a new [FoldedString] pointer from a string pointer.
func NewFoldedString(s *string) *FoldedString {
	if s == nil {
		return nil
	}
	folded := FoldedString(*s)
	return &folded
}

// MarshalYAML implements the [yaml.Marshaler] interface using folded style.
func (f FoldedString) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: string(f),
		Style: yaml.FoldedStyle,
	}, nil
}
