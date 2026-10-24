package storage_test

import (
	"testing"
	"uuid"

	"github.com/coreofscience/go-bibx/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "quantum computing",
			expected: "quantum computing",
		},
		{
			name:     "leading and trailing spaces",
			input:    "   quantum computing   ",
			expected: "quantum computing",
		},
		{
			name:     "uppercase and mixed case",
			input:    "QuAnTuM CoMpUtInG",
			expected: "quantum computing",
		},
		{
			name:     "multiple consecutive spaces",
			input:    "quantum    machine     learning",
			expected: "quantum machine learning",
		},
		{
			name:     "tabs and newlines",
			input:    " \t quantum \n\t  computing \r\n ",
			expected: "quantum computing",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := storage.NormalizeQuery(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestComputeQueryHash(t *testing.T) {
	t.Run("identical hashes for whitespace and casing variations", func(t *testing.T) {
		h1 := storage.ComputeQueryHash("quantum computing", 100)
		h2 := storage.ComputeQueryHash("   Quantum   Computing   ", 100)
		h3 := storage.ComputeQueryHash("QUANTUM\tCOMPUTING\n", 100)

		assert.Equal(t, h1, h2)
		assert.Equal(t, h1, h3)
	})

	t.Run("different hashes for different limits", func(t *testing.T) {
		h1 := storage.ComputeQueryHash("quantum computing", 100)
		h2 := storage.ComputeQueryHash("quantum computing", 200)

		assert.NotEqual(t, h1, h2)
	})

	t.Run("different hashes for different queries", func(t *testing.T) {
		h1 := storage.ComputeQueryHash("quantum computing", 100)
		h2 := storage.ComputeQueryHash("classical computing", 100)

		assert.NotEqual(t, h1, h2)
	})
}

func TestUUIDHelpers(t *testing.T) {
	t.Run("UUIDToBytes and ScanUUID with []byte", func(t *testing.T) {
		original := uuid.NewV7()
		bytes := storage.UUIDToBytes(original)
		require.Len(t, bytes, 16)

		scanned, err := storage.ScanUUID(bytes)
		require.NoError(t, err)
		assert.Equal(t, original, scanned)
	})

	t.Run("ScanUUID with valid string", func(t *testing.T) {
		original := uuid.NewV7()
		str := original.String()

		scanned, err := storage.ScanUUID(str)
		require.NoError(t, err)
		assert.Equal(t, original, scanned)
	})

	t.Run("ScanUUID with invalid byte length", func(t *testing.T) {
		_, err := storage.ScanUUID([]byte{1, 2, 3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid uuid blob length")
	})

	t.Run("ScanUUID with invalid string", func(t *testing.T) {
		_, err := storage.ScanUUID("not-a-uuid")
		require.Error(t, err)
	})

	t.Run("ScanUUID with unsupported type", func(t *testing.T) {
		_, err := storage.ScanUUID(12345)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported uuid type")
	})
}
