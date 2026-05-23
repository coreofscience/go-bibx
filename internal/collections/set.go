package collections

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
)

type Set[T cmp.Ordered] struct {
	elements map[T]struct{}
}

// NewSet creates a new Set instance.
func NewSet[T cmp.Ordered](elements ...T) *Set[T] {
	internalElements := make(map[T]struct{})
	for _, elem := range elements {
		internalElements[elem] = struct{}{}
	}
	return &Set[T]{
		elements: internalElements,
	}
}

// Clone returns a copy of the set.
func (s *Set[T]) Clone() *Set[T] {
	if s == nil {
		return nil
	}
	internalElements := make(map[T]struct{})
	for elem := range s.elements {
		internalElements[elem] = struct{}{}
	}
	return &Set[T]{
		elements: internalElements,
	}
}

// Contains returns true if the set contains the given element.
func (s *Set[T]) Contains(elem T) bool {
	if s == nil {
		return false
	}
	_, ok := s.elements[elem]
	return ok
}

// Add adds an element to the set.
func (s *Set[T]) Add(elem T) {
	if s == nil {
		return
	}
	if s.elements == nil {
		s.elements = make(map[T]struct{})
	}
	s.elements[elem] = struct{}{}
}

// Len returns the number of elements in the set.
func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.elements)
}

// Union returns a new Set that is the union of the current set and another set.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	if s == nil {
		return other
	}
	if other == nil {
		return s
	}

	unionSet := NewSet[T]()
	for elem := range s.elements {
		unionSet.elements[elem] = struct{}{}
	}
	for elem := range other.elements {
		unionSet.elements[elem] = struct{}{}
	}
	return unionSet
}

func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	if s == nil || other == nil {
		return nil
	}
	intersectSet := NewSet[T]()
	for elem := range s.elements {
		if other.Contains(elem) {
			intersectSet.Add(elem)
		}
	}
	return intersectSet
}

// Items returns a sorted slice of the elements in the set.
func (s *Set[T]) Items() []T {
	if s == nil {
		return nil
	}
	items := make([]T, 0, len(s.elements))
	for elem := range s.elements {
		items = append(items, elem)
	}
	slices.Sort(items)
	return items
}

// MarshalJSON implements the json.Marshaler interface.
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	bytes, err := json.Marshal(s.Items())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal set: %w", err)
	}
	return bytes, nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (s *Set[T]) MarshalYAML() (any, error) {
	return s.Items(), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("failed to unmarshal set: %w", err)
	}
	s.elements = make(map[T]struct{})
	for _, item := range items {
		s.elements[item] = struct{}{}
	}
	return nil
}
