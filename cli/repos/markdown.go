package repos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/models"
	"go.yaml.in/yaml/v4"
)

type MarkdownRepo interface {
	Store(ctx context.Context, a *models.Analysis) error
}

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
	sb.WriteString(utils.WrapText(abstractText, 80))
	sb.WriteString("\n\n")

	// Write keywords
	sb.WriteString("# Keywords\n\n")
	var keywordsList []string
	if art.Keywords != nil {
		keywordsList = art.Keywords.Items()
	}
	if len(keywordsList) > 0 {
		keywordsStr := strings.Join(keywordsList, ", ")
		sb.WriteString(utils.WrapText(keywordsStr, 80))
	} else {
		sb.WriteString("No keywords available.")
	}
	sb.WriteString("\n\n")

	// Write references
	sb.WriteString("# References\n\n")
	if len(art.References) > 0 {
		for i, ref := range art.References {
			formatted := formatReference(ref)
			wrapped := utils.WrapText(formatted, 80)

			if ref.Rich {
				id := ""
				if ref.Key() != nil {
					id = *ref.Key()
				}
				tempNode := &models.Node{
					ID:      id,
					Article: ref,
				}
				filename := tempNode.Filename()
				linkName, _ := strings.CutSuffix(filename, ".md")

				sb.WriteString("[[")
				sb.WriteString(linkName)
				sb.WriteString("]]\n")
			}

			sb.WriteString(wrapped)
			if i < len(art.References)-1 {
				sb.WriteString("\n\n")
			}
		}
	} else {
		sb.WriteString("No references available.")
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

func formatReference(ref *models.Article) string {
	if ref == nil {
		return ""
	}
	if !ref.Rich {
		return ref.Label
	}

	var sb strings.Builder

	if len(ref.Authors) > 0 {
		sb.WriteString(strings.Join(ref.Authors, ", "))
	}

	sb.WriteString(`, "`)

	if ref.Title != nil {
		sb.WriteString(*ref.Title)
	}

	sb.WriteString(`", `)

	if ref.Journal != nil && *ref.Journal != "" {
		sb.WriteString("*")
		sb.WriteString(*ref.Journal)
		sb.WriteString("*")
	}

	if ref.Volume != nil && *ref.Volume != "" {
		sb.WriteString(", vol. ")
		sb.WriteString(*ref.Volume)
	}

	if ref.Issue != nil && *ref.Issue != "" {
		sb.WriteString(", no. ")
		sb.WriteString(*ref.Issue)
	}

	if ref.Page != nil && *ref.Page != "" {
		sb.WriteString(", pp. ")
		sb.WriteString(*ref.Page)
	}

	if ref.Year != nil {
		sb.WriteString(", ")
		fmt.Fprintf(&sb, "%d", *ref.Year)
	}

	if ref.DOI != nil && *ref.DOI != "" {
		sb.WriteString(". doi: ")
		sb.WriteString(*ref.DOI)
	}

	sb.WriteString(".")

	return sb.String()
}
