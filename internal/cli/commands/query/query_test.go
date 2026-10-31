package query_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/cli/commands/query"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/nickng/bibtex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func captureStdout(f func() error) (string, error) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("failed to create os pipe: %w", err)
	}
	os.Stdout = w

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	runErr := f()
	_ = w.Close()
	os.Stdout = origStdout
	output := <-outChan
	_ = r.Close()

	return output, runErr
}

func TestQueryCommand_BibtexFormat(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_query_cmd.db")
	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	colSvc := services.NewCollectionService(db)
	anaSvc := services.NewAnalysisService(db, openalex.NewRestyClient())
	ctx := context.Background()

	colID := uuid.NewV7()
	hash := services.ComputeQueryHash("query cmd test", 10)
	require.NoError(t, colSvc.Create(ctx, &services.Collection{
		ID:          colID,
		QueryHash:   hash,
		Query:       "query cmd test",
		ResultLimit: 10,
	}))

	year := 2017
	title := "Attention Is All You Need"
	venue := "NeurIPS"
	work := &models.Work{
		Label:   "openalex:W100",
		Title:   &title,
		Venue:   &venue,
		Authors: []string{"Vaswani, Ashish"},
		Year:    &year,
		IDs:     collections.NewSet("openalex:W100"),
		Rich:    true,
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{
				ID:       "openalex:W100",
				Category: models.CategoryRoot,
				Rootness: 0.95,
				Work:     work,
			},
		},
	}
	require.NoError(t, anaSvc.StoreAnalysis(ctx, colID, analysis))

	rootCmd := &cli.Command{
		Name: "bibx",
		Commands: []*cli.Command{
			query.New(),
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			return utils.WithDBPath(ctx, dbPath), nil
		},
	}

	output, err := captureStdout(func() error {
		return rootCmd.Run(ctx, []string{"bibx", "query", "--category", "root", "--format", "bibtex"})
	})
	require.NoError(t, err)
	assert.Contains(t, output, "@article{vaswani2017,")
	assert.Contains(t, output, "Attention Is All You Need")

	parsed, err := bibtex.Parse(strings.NewReader(output))
	require.NoError(t, err)
	require.Len(t, parsed.Entries, 1)
}
