package repos

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

//go:embed all:templates
var templatesFS embed.FS

type ScaffoldRepo interface {
	Scaffold() error
}

type TemplateScaffoldRepo struct {
	rootDir string
}

func NewTemplateScaffoldRepo(rootDir string) *TemplateScaffoldRepo {
	return &TemplateScaffoldRepo{
		rootDir: rootDir,
	}
}

func (r *TemplateScaffoldRepo) Scaffold() error {
	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk templates: %w", err)
		}

		// Get the relative path from the "templates" directory
		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Construct the destination path in the root directory
		destPath := filepath.Join(r.rootDir, relPath)

		// If it's a directory, create it in the destination
		if d.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			return nil
		}

		// If it's a file, copy it from the embedded filesystem to the destination
		if err := copyEmbedFile(templatesFS, path, destPath); err != nil {
			return fmt.Errorf("failed to copy file from %s to %s: %w", path, destPath, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scaffold templates: %w", err)
	}
	return nil
}

func copyEmbedFile(embedFS embed.FS, srcPath, destPath string) (err error) {
	// Open the source file from the embedded filesystem
	srcFile, err := embedFS.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open embedded file %s: %w", srcPath, err)
	}
	defer func() {
		cerr := srcFile.Close()
		if cerr != nil {
			slog.Error("failed to close source file", "file", srcPath, "error", cerr)
			err = errors.Join(err, fmt.Errorf("failed to close destination file %s: %w", destPath, cerr))
		}
	}()

	// Ensure the destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer func() {
		cerr := destFile.Close()
		if cerr != nil {
			slog.Error("failed to close destination file", "file", destPath, "error", cerr)
			err = errors.Join(err, fmt.Errorf("failed to close destination file %s: %w", destPath, cerr))
		}
	}()

	// Copy the contents from the source file to the destination file
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", srcPath, destPath, err)
	}

	return nil
}
