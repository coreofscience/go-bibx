package vector

import "math"

// CosineDistance calculates the cosine distance between two vectors. The
// cosine distance is defined as 1 - (dot product of a and b) / (magnitude of a
// * magnitude of b). The result is in the range [0, 2], where 0 means the
// vectors are identical and 2 means they are opposite.
func CosineDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("vectors must have the same length")
	}
	var dotProduct float32
	var normA float32
	var normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	return 1 - (dotProduct / (sqrt(normA) * sqrt(normB)))
}

func sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}
