package repos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreofscience/go-bibx/models"
	"go.yaml.in/yaml/v4"
)

type FolderMarkdownRepo struct {
	Dir string
}

func NewFolderMarkdownRepo(dir string) *FolderMarkdownRepo {
	return &FolderMarkdownRepo{
		Dir: dir,
	}
}

func (r *FolderMarkdownRepo) Store(ctx context.Context, a *models.Analysis) error {
	// Clean/clear the directory first
	if err := os.RemoveAll(r.Dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean target directory %s: %w", r.Dir, err)
	}

	// Recreate target directory
	if err := os.MkdirAll(r.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", r.Dir, err)
	}

	for _, node := range a.Nodes {
		if node.Article == nil {
			continue
		}

		filename := node.Filename()
		filePath := filepath.Join(r.Dir, filename)

		content, err := formatNodeMarkdown(node)
		if err != nil {
			return fmt.Errorf("failed to format node %s: %w", node.ID, err)
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write markdown file %s: %w", filePath, err)
		}
	}

	return nil
}

func formatNodeMarkdown(node *models.Node) (string, error) {
	meta := node.Metadata()
	if meta == nil {
		return "", fmt.Errorf("node has no metadata")
	}

	yamlBytes, err := yaml.Dump(meta, yaml.WithIndent(2), yaml.WithLineWidth(80))
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter to yaml: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(yamlBytes)
	sb.WriteString("---\n\n")

	art := node.Article

	// Write abstract
	sb.WriteString("# Abstract\n\n")
	var abstractText string
	if art.Abstract != nil && *art.Abstract != "" {
		abstractText = *art.Abstract
	} else {
		abstractText = "No abstract available."
	}
	sb.WriteString(wrapText(abstractText, 80))
	sb.WriteString("\n\n")

	// Write keywords
	sb.WriteString("# Keywords\n\n")
	var keywordsList []string
	if art.Keywords != nil {
		keywordsList = art.Keywords.Items()
	}
	if len(keywordsList) > 0 {
		keywordsStr := strings.Join(keywordsList, ", ")
		sb.WriteString(wrapText(keywordsStr, 80))
	} else {
		sb.WriteString("No keywords available.")
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

func wrapText(text string, limit int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var sb strings.Builder
	lineLen := 0
	for i, word := range words {
		if lineLen+len(word)+1 > limit && lineLen > 0 {
			sb.WriteString("\n")
			lineLen = 0
		} else if i > 0 {
			sb.WriteString(" ")
			lineLen++
		}
		sb.WriteString(word)
		lineLen += len(word)
	}
	return sb.String()
}
