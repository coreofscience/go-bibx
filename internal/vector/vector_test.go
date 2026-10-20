package vector_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/vector"
	"github.com/stretchr/testify/assert"
)

func TestCosineDistance(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		a := []float32{1.0, 2.0, 3.0}
		b := []float32{1.0, 2.0, 3.0}
		dist := vector.CosineDistance(a, b)
		assert.InDelta(t, float32(0.0), dist, 1e-6)
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{0.0, 1.0}
		dist := vector.CosineDistance(a, b)
		assert.InDelta(t, float32(1.0), dist, 1e-6)
	})

	t.Run("opposite vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{-1.0, 0.0}
		dist := vector.CosineDistance(a, b)
		assert.InDelta(t, float32(2.0), dist, 1e-6)
	})

	t.Run("zero vector returns max distance 2.0 without NaN", func(t *testing.T) {
		a := []float32{0.0, 0.0, 0.0}
		b := []float32{1.0, 2.0, 3.0}
		dist := vector.CosineDistance(a, b)
		assert.Equal(t, float32(2.0), dist)

		distRev := vector.CosineDistance(b, a)
		assert.Equal(t, float32(2.0), distRev)
	})

	t.Run("both zero vectors returns 2.0 without NaN", func(t *testing.T) {
		a := []float32{0.0, 0.0}
		b := []float32{0.0, 0.0}
		dist := vector.CosineDistance(a, b)
		assert.Equal(t, float32(2.0), dist)
	})

	t.Run("empty vectors return 2.0 without NaN", func(t *testing.T) {
		a := []float32{}
		b := []float32{}
		dist := vector.CosineDistance(a, b)
		assert.Equal(t, float32(2.0), dist)
	})

	t.Run("clamping prevents negative distance from floating point drift", func(t *testing.T) {
		// Vectors with tiny floating point imprecision
		a := []float32{0.57735026, 0.57735026, 0.57735026}
		b := []float32{0.57735026, 0.57735026, 0.57735026}
		dist := vector.CosineDistance(a, b)
		assert.GreaterOrEqual(t, dist, float32(0.0))
		assert.LessOrEqual(t, dist, float32(2.0))
	})

	t.Run("mismatched vector lengths panics", func(t *testing.T) {
		a := []float32{1.0}
		b := []float32{1.0, 2.0}
		assert.Panics(t, func() {
			vector.CosineDistance(a, b)
		})
	})
}
