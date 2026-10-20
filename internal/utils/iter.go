package utils

import "iter"

// Zip returns an iterator yielding key-value pairs from two slices until the shorter slice is exhausted.
func Zip[K any, V any](keys []K, values []V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i, key := range keys {
			if i >= len(values) {
				return
			}
			if !yield(key, values[i]) {
				return
			}
		}
	}
}
