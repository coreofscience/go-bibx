package collections

type Set[T comparable] struct {
	elements map[T]struct{}
}

// NewSet creates a new Set instance.
func NewSet[T comparable](elements ...T) *Set[T] {
	internalElements := make(map[T]struct{})
	for _, elem := range elements {
		internalElements[elem] = struct{}{}
	}
	return &Set[T]{
		elements: internalElements,
	}
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

// Items returns an unsorted slice of the elements in the set.
func (s *Set[T]) Items() []T {
	if s == nil {
		return nil
	}
	items := make([]T, 0, len(s.elements))
	for elem := range s.elements {
		items = append(items, elem)
	}
	return items
}
