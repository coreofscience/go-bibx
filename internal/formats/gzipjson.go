package formats

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func LoadGzipJSON(fileName string, v any) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		err := gz.Close()
		if err != nil {
			slog.Error("failed to close gzip reader", "error", err)
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
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()
	gz := gzip.NewWriter(file)
	defer func() {
		err := gz.Close()
		if err != nil {
			slog.Error("failed to close gzip writer", "error", err)
		}
	}()
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		return fmt.Errorf("failed to encode to gzip json: %w", err)
	}
	return nil
}
