package formats

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func LoadGzipJSON(fileName string, v any) (err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			slog.Error("failed to close file", "error", cerr)
			err = errors.Join(err, fmt.Errorf("failed to close file: %w", cerr))
		}
	}()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if cerr := gz.Close(); cerr != nil {
			slog.Error("failed to close gzip reader", "error", cerr)
			err = errors.Join(err, fmt.Errorf("failed to close gzip reader: %w", cerr))
		}
	}()
	if err := json.NewDecoder(gz).Decode(v); err != nil {
		return fmt.Errorf("failed to decode gzip json data: %w", err)
	}
	return nil
}

func StoreGzipJSON(fileName string, v any) error {
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	perm := filePerm(fileName)

	tempFile, err := os.CreateTemp(dir, fmt.Sprintf(".%s-*.tmp", filepath.Base(fileName)))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempName := tempFile.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()

	gz := gzip.NewWriter(tempFile)
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		_ = gz.Close()
		_ = tempFile.Close()
		return fmt.Errorf("failed to encode to gzip json: %w", err)
	}

	if err := gz.Close(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tempName, perm); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	if err := os.Rename(tempName, fileName); err != nil {
		return fmt.Errorf("failed to rename temp file to target file: %w", err)
	}

	return nil
}
