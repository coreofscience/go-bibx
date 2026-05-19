package formats_test

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreofscience/go-bibx/internal/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestStoreAndLoadGzipJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gzipjson_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	fileName := filepath.Join(tempDir, "test.json.gz")
	data := TestData{
		Name: "Test User",
		Age:  30,
	}

	// Test StoreGzipJSON
	err = formats.StoreGzipJSON(fileName, data)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(fileName)
	assert.NoError(t, err)

	// Test LoadGzipJSON
	var loadedData TestData
	err = formats.LoadGzipJSON(fileName, &loadedData)
	assert.NoError(t, err)
	assert.Equal(t, data, loadedData)
}

func TestLoadGzipJSON_FileNotFound(t *testing.T) {
	var data TestData
	err := formats.LoadGzipJSON("non_existent_file.json.gz", &data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestLoadGzipJSON_InvalidGzip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gzipjson_test_invalid")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	fileName := filepath.Join(tempDir, "invalid.json.gz")
	err = os.WriteFile(fileName, []byte("not a gzip file"), 0644)
	require.NoError(t, err)

	var data TestData
	err = formats.LoadGzipJSON(fileName, &data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gzip reader")
}

func TestStoreGzipJSON_CreateDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gzipjson_test_dir")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Path with a non-existent sub-directory
	fileName := filepath.Join(tempDir, "sub", "test.json.gz")
	data := TestData{Name: "Sub Dir Test"}

	err = formats.StoreGzipJSON(fileName, data)
	assert.NoError(t, err)

	var loadedData TestData
	err = formats.LoadGzipJSON(fileName, &loadedData)
	assert.NoError(t, err)
	assert.Equal(t, data, loadedData)
}

func TestLoadGzipJSON_InvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gzipjson_test_invalid_json")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	fileName := filepath.Join(tempDir, "invalid_json.json.gz")

	file, err := os.Create(fileName)
	require.NoError(t, err)

	gz := gzip.NewWriter(file)
	_, err = gz.Write([]byte("{invalid json}"))
	require.NoError(t, err)
	err = gz.Close()
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	var data TestData
	err = formats.LoadGzipJSON(fileName, &data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode gzip json data")
}
