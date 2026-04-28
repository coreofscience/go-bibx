package collections

import "sort"

type CounterItem[T comparable] struct {
	Item  T
	Count int
}

type Counter[T comparable] struct {
	counts map[T]int
}

// NewCounter creates a new Counter instance.
func NewCounter[T comparable](items ...T) *Counter[T] {
	counts := make(map[T]int)
	for _, item := range items {
		counts[item]++
	}
	return &Counter[T]{
		counts: counts,
	}
}

// MostCommon returns the most common items and their counts.
func (c *Counter[T]) MostCommon(count int) []CounterItem[T] {
	if c == nil || len(c.counts) == 0 || count <= 0 {
		return nil
	}
	var ss []CounterItem[T]
	for k, v := range c.counts {
		ss = append(ss, CounterItem[T]{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Count > ss[j].Count
	})
	if count > len(ss) {
		count = len(ss)
	}
	return ss[:count]
}
