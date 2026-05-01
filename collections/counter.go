package collections

import (
	"cmp"
	"sort"
)

type CounterItem[T cmp.Ordered] struct {
	Item  T
	Count int
}

type Counter[T cmp.Ordered] struct {
	counts map[T]int
}

// NewCounter creates a new Counter instance.
func NewCounter[T cmp.Ordered](items ...T) *Counter[T] {
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
		if ss[i].Count == ss[j].Count {
			return ss[i].Item < ss[j].Item
		}
		return ss[i].Count > ss[j].Count
	})
	if count > len(ss) {
		count = len(ss)
	}
	return ss[:count]
}
