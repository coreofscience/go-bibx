package utils

import "iter"

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
