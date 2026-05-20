package formats

import (
	"compress/gzip"
	"encoding/json"
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
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if cerr := gz.Close(); cerr != nil {
			slog.Error("failed to close gzip reader", "error", cerr)
			err = fmt.Errorf("failed to close gzip reader: %w", cerr)
		}
	}()
	if err := json.NewDecoder(gz).Decode(v); err != nil {
		return fmt.Errorf("failed to decode gzip json data: %w", err)
	}
	return nil
}

func StoreGzipJSON(fileName string, v any) (err error) {
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			slog.Error("failed to close file", "error", cerr)
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()
	gz := gzip.NewWriter(file)
	defer func() {
		if cerr := gz.Close(); cerr != nil {
			slog.Error("failed to close gzip writer", "error", cerr)
			err = fmt.Errorf("failed to close gzip writer: %w", cerr)
		}
	}()
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		return fmt.Errorf("failed to encode to gzip json: %w", err)
	}
	return nil
}
