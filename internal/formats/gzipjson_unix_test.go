//go:build unix

package formats_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/coreofscience/go-bibx/internal/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreGzipJSON_HonorsUmask(t *testing.T) {
	// Set restrictive umask 0077
	prevUmask := syscall.Umask(0077)
	t.Cleanup(func() {
		syscall.Umask(prevUmask)
	})

	tempDir := t.TempDir()
	fileName := filepath.Join(tempDir, "umask_test.json.gz")

	data := TestData{Name: "Restrictive", Age: 30}
	err := formats.StoreGzipJSON(fileName, data)
	require.NoError(t, err)

	info, err := os.Stat(fileName)
	require.NoError(t, err)
	// With umask 0077, 0666 &^ 0077 should result in 0600 permissions.
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
