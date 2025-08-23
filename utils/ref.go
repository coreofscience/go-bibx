package utils

func NewRef[T any](v T) *T {
	return &v
}
