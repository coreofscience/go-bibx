package utils

import "slices"

// KeepLongestSlice returns the longest of two slices
func KeepLongestSlice[T any](a, b []T) []T {
	if len(a) >= len(b) {
		return a
	}
	return b
}

// Chunks splits a slice into chunks of the specified size
func Chunks[T any](slice []T, chunkSize int) [][]T {
	numChunks := (len(slice) + chunkSize - 1) / chunkSize
	chunked := make([][]T, 0, numChunks)
	for i := 0; i < len(slice); i += chunkSize {
		end := min(i+chunkSize, len(slice))
		chunked = append(chunked, slice[i:end])
	}
	return chunked
}

func Limit[T any](slice []T, n int, compare func(a, b T) int) []T {
	slices.SortFunc(slice, compare)
	if n >= len(slice) {
		return slice
	}
	return slice[:n]
}
