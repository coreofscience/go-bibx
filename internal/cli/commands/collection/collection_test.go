package collection_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/cli/commands/collection"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
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

func setupTestStore(t *testing.T) (string, services.CollectionService, func()) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_collection_cmd.db")
	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	service := services.NewCollectionService(db)
	return dbPath, service, func() {
		require.NoError(t, db.Close())
	}
}

func makeRootCmd(dbPath string) *cli.Command {
	return &cli.Command{
		Name: "bibx",
		Commands: []*cli.Command{
			collection.New(),
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			return utils.WithDBPath(ctx, dbPath), nil
		},
	}
}

func TestCollectionCommands(t *testing.T) {
	dbPath, store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	rootCmd := makeRootCmd(dbPath)

	t.Run("list empty", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "list"})
		})
		require.NoError(t, err)
		assert.Contains(t, out, "No collections found.")
	})

	t.Run("show empty", func(t *testing.T) {
		err := rootCmd.Run(ctx, []string{"bibx", "collection", "show"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no collections found")
	})

	// Seed collections
	name1 := "first-col"
	col1 := &services.Collection{
		ID:           uuid.NewV7(),
		Name:         &name1,
		QueryHash:    services.ComputeQueryHash("first query", 10),
		Query:        "first query",
		ResultLimit:  10,
		ArticleCount: 5,
	}
	require.NoError(t, store.Create(ctx, col1))

	name2 := "second-col"
	col2 := &services.Collection{
		ID:           uuid.NewV7(),
		Name:         &name2,
		QueryHash:    services.ComputeQueryHash("second query", 20),
		Query:        "second query",
		ResultLimit:  20,
		ArticleCount: 12,
	}
	require.NoError(t, store.Create(ctx, col2))

	t.Run("list populated", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "list"})
		})
		require.NoError(t, err)
		assert.Contains(t, out, "first-col")
		assert.Contains(t, out, "second-col")
		assert.Contains(t, out, col1.ID.String())
		assert.Contains(t, out, col2.ID.String())

		savedCol1, err := store.FindByID(ctx, col1.ID)
		require.NoError(t, err)
		assert.Equal(t, time.UTC, savedCol1.CreatedAt.Location())
		assert.Contains(t, out, savedCol1.CreatedAt.Local().Format("2006-01-02 15:04"))
	})

	t.Run("show latest", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "show"})
		})
		require.NoError(t, err)
		assert.Contains(t, out, col2.ID.String())
		assert.Contains(t, out, "second-col")
		assert.Contains(t, out, "second query")

		savedCol2, err := store.FindByID(ctx, col2.ID)
		require.NoError(t, err)
		assert.Equal(t, time.UTC, savedCol2.CreatedAt.Location())
		assert.Contains(t, out, savedCol2.CreatedAt.Local().Format("2006-01-02 15:04:05"))
		assert.Contains(t, out, savedCol2.UpdatedAt.Local().Format("2006-01-02 15:04:05"))
	})

	t.Run("show by name", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "show", "first-col"})
		})
		require.NoError(t, err)
		assert.Contains(t, out, col1.ID.String())
		assert.Contains(t, out, "first-col")
		assert.Contains(t, out, "first query")
	})

	t.Run("show by UUID", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "show", col2.ID.String()})
		})
		require.NoError(t, err)
		assert.Contains(t, out, col2.ID.String())
		assert.Contains(t, out, "second-col")
	})

	t.Run("show nonexistent", func(t *testing.T) {
		err := rootCmd.Run(ctx, []string{"bibx", "collection", "show", "nonexistent-target"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection not found")
	})

	t.Run("delete without arg", func(t *testing.T) {
		err := rootCmd.Run(ctx, []string{"bibx", "collection", "delete"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection ID or name argument is required")
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		err := rootCmd.Run(ctx, []string{"bibx", "collection", "delete", "nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collection not found")
	})

	t.Run("delete by name", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "delete", "first-col"})
		})
		require.NoError(t, err)
		assert.Contains(t, out, "Deleted collection")
		assert.Contains(t, out, "first-col")

		// Verify it was deleted
		_, err = store.FindByID(ctx, col1.ID)
		require.ErrorIs(t, err, services.ErrCollectionNotFound)
	})

	t.Run("delete by UUID", func(t *testing.T) {
		out, err := captureStdout(func() error {
			return rootCmd.Run(ctx, []string{"bibx", "collection", "delete", col2.ID.String()})
		})
		require.NoError(t, err)
		assert.Contains(t, out, "Deleted collection")
		assert.Contains(t, out, col2.ID.String())

		// Verify it was deleted
		_, err = store.FindByID(ctx, col2.ID)
		require.ErrorIs(t, err, services.ErrCollectionNotFound)
	})
}
