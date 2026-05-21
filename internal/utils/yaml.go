package utils

import "go.yaml.in/yaml/v4"

type FoldedString string

func NewFoldedString(s *string) *FoldedString {
	if s == nil {
		return nil
	}
	folded := FoldedString(*s)
	return &folded
}

func (f FoldedString) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: string(f),
		Style: yaml.FoldedStyle,
	}, nil
}
