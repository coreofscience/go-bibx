package utils

// Keep returns the first non-nil option from the given list of options.
func Keep[T any](options ...*T) *T {
	for _, option := range options {
		if option != nil {
			return option
		}
	}
	return nil
}
