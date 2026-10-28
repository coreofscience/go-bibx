package collection_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/nickng/bibtex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportCommand(t *testing.T) {
	dbPath, colSvc, cleanup := setupTestStore(t)
	defer cleanup()

	rootCmd := makeRootCmd(dbPath)
	ctx := context.Background()

	t.Run("fails when no collections exist", func(t *testing.T) {
		_, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export"})
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no collections found")
	})

	// Setup database with an analysis
	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	anaSvc := services.NewAnalysisService(db, openalex.NewRestyClient())

	colID := uuid.NewV7()
	hash := services.ComputeQueryHash("export test", 10)
	alias := "my-export-alias"
	require.NoError(t, colSvc.Create(ctx, &services.Collection{
		ID:          colID,
		QueryHash:   hash,
		Query:       "export test",
		ResultLimit: 10,
		Name:        &alias,
	}))

	year := 2021
	title := "Attention Mechanism in Vision"
	journal := "CVPR"
	art := &models.Article{
		Label:   "openalex:W999",
		Title:   &title,
		Journal: &journal,
		Authors: []string{"Dosovitskiy, Alexey"},
		Year:    &year,
		IDs:     collections.NewSet("openalex:W999"),
		Rich:    true,
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{
				ID:       "openalex:W999",
				Category: models.CategoryRoot,
				Rootness: 0.99,
				Article:  art,
			},
		},
	}
	require.NoError(t, anaSvc.StoreAnalysis(ctx, colID, analysis))

	t.Run("exports latest collection as bibtex to stdout by default", func(t *testing.T) {
		output, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export"})
		})
		require.NoError(t, err)
		assert.Contains(t, output, "@article{dosovitskiy2021,")
		assert.Contains(t, output, "Attention Mechanism in Vision")

		parsed, err := bibtex.Parse(strings.NewReader(output))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 1)
	})

	t.Run("exports specified collection by alias", func(t *testing.T) {
		output, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export", alias})
		})
		require.NoError(t, err)
		assert.Contains(t, output, "@article{dosovitskiy2021,")
	})

	t.Run("exports to file when --output specified", func(t *testing.T) {
		tempDir := t.TempDir()
		outFile := filepath.Join(tempDir, "refs.bib")

		output, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export", "--output", outFile})
		})
		require.NoError(t, err)
		assert.Empty(t, output) // file output does not write to stdout

		content, err := os.ReadFile(outFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "@article{dosovitskiy2021,")
	})

	t.Run("exports as json when format specified", func(t *testing.T) {
		output, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export", "-f", "json"})
		})
		require.NoError(t, err)
		assert.Contains(t, output, `"nodes"`)
		assert.Contains(t, output, `"Attention Mechanism in Vision"`)
	})

	t.Run("fails on non-existent collection target", func(t *testing.T) {
		_, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export", "non-existent-col"})
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection not found: non-existent-col")
	})

	t.Run("fails on invalid format", func(t *testing.T) {
		_, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "export", "-f", "invalid-format"})
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format")
	})
}
