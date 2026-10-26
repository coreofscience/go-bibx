package services_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "quantum computing",
			expected: "quantum computing",
		},
		{
			name:     "leading and trailing spaces",
			input:    "   quantum computing   ",
			expected: "quantum computing",
		},
		{
			name:     "uppercase and mixed case",
			input:    "QuAnTuM CoMpUtInG",
			expected: "quantum computing",
		},
		{
			name:     "multiple consecutive spaces",
			input:    "quantum    machine     learning",
			expected: "quantum machine learning",
		},
		{
			name:     "tabs and newlines",
			input:    " \t quantum \n\t  computing \r\n ",
			expected: "quantum computing",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := services.NormalizeQuery(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestComputeQueryHash(t *testing.T) {
	t.Run("identical hashes for whitespace and casing variations", func(t *testing.T) {
		h1 := services.ComputeQueryHash("quantum computing", 100)
		h2 := services.ComputeQueryHash("   Quantum   Computing   ", 100)
		h3 := services.ComputeQueryHash("QUANTUM\tCOMPUTING\n", 100)

		assert.Equal(t, h1, h2)
		assert.Equal(t, h1, h3)
	})

	t.Run("different hashes for different limits", func(t *testing.T) {
		h1 := services.ComputeQueryHash("quantum computing", 100)
		h2 := services.ComputeQueryHash("quantum computing", 200)

		assert.NotEqual(t, h1, h2)
	})

	t.Run("different hashes for different queries", func(t *testing.T) {
		h1 := services.ComputeQueryHash("quantum computing", 100)
		h2 := services.ComputeQueryHash("classical computing", 100)

		assert.NotEqual(t, h1, h2)
	})
}

func TestUUIDHelpers(t *testing.T) {
	t.Run("UUIDToBytes and ScanUUID with []byte", func(t *testing.T) {
		original := uuid.NewV7()
		bytes := services.UUIDToBytes(original)
		require.Len(t, bytes, 16)

		scanned, err := services.ScanUUID(bytes)
		require.NoError(t, err)
		assert.Equal(t, original, scanned)
	})

	t.Run("ScanUUID with valid string", func(t *testing.T) {
		original := uuid.NewV7()
		str := original.String()

		scanned, err := services.ScanUUID(str)
		require.NoError(t, err)
		assert.Equal(t, original, scanned)
	})

	t.Run("ScanUUID with invalid byte length", func(t *testing.T) {
		_, err := services.ScanUUID([]byte{1, 2, 3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid uuid blob length")
	})

	t.Run("ScanUUID with invalid string", func(t *testing.T) {
		_, err := services.ScanUUID("not-a-uuid")
		require.Error(t, err)
	})

	t.Run("ScanUUID with unsupported type", func(t *testing.T) {
		_, err := services.ScanUUID(12345)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported uuid type")
	})
}

func TestCollectionService_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_collection_crud.db")

	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	colSvc := services.NewCollectionService(db)
	ctx := context.Background()

	name1 := "quantum-computing"
	hash1 := services.ComputeQueryHash("quantum computing", 100)
	col1 := &services.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   hash1,
		Query:       "quantum computing",
		ResultLimit: 100,
		Name:        &name1,
	}

	// 1. Create collection
	err = colSvc.Create(ctx, col1)
	require.NoError(t, err)

	// 2. Create collection with duplicate hash fails
	err = colSvc.Create(ctx, &services.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   hash1,
		Query:       "quantum computing",
		ResultLimit: 100,
	})
	require.ErrorIs(t, err, services.ErrCollectionExists)

	// 2b. Create collection with duplicate name fails
	err = colSvc.Create(ctx, &services.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   services.ComputeQueryHash("different query", 100),
		Query:       "different query",
		ResultLimit: 100,
		Name:        &name1,
	})
	require.ErrorIs(t, err, services.ErrCollectionExists)

	// 3. Create nil collection fails
	err = colSvc.Create(ctx, nil)
	require.Error(t, err)

	// 4. Find by hash
	foundHash, err := colSvc.FindByHash(ctx, hash1)
	require.NoError(t, err)
	assert.Equal(t, col1.ID, foundHash.ID)
	assert.Equal(t, col1.Query, foundHash.Query)
	assert.Equal(t, name1, *foundHash.Name)
	assert.Equal(t, time.UTC, foundHash.CreatedAt.Location())

	// 5. Find by non-existent hash
	_, err = colSvc.FindByHash(ctx, services.ComputeQueryHash("non-existent", 10))
	require.ErrorIs(t, err, services.ErrCollectionNotFound)

	// 6. Find by ID
	foundID, err := colSvc.FindByID(ctx, col1.ID)
	require.NoError(t, err)
	assert.Equal(t, col1.ID, foundID.ID)

	// 7. Find by non-existent ID
	_, err = colSvc.FindByID(ctx, uuid.NewV7())
	require.ErrorIs(t, err, services.ErrCollectionNotFound)

	// 8. Find by name
	foundName, err := colSvc.FindByName(ctx, name1)
	require.NoError(t, err)
	assert.Equal(t, col1.ID, foundName.ID)

	// 9. Find by non-existent name
	_, err = colSvc.FindByName(ctx, "does-not-exist")
	require.ErrorIs(t, err, services.ErrCollectionNotFound)

	// 10. List collections
	hash2 := services.ComputeQueryHash("deep learning", 50)
	col2 := &services.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   hash2,
		Query:       "deep learning",
		ResultLimit: 50,
	}
	require.NoError(t, colSvc.Create(ctx, col2))

	list, err := colSvc.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, col2.ID, list[0].ID) // Ordered by created_at DESC, id DESC

	// 10b. UpdateName to an existing collection's name fails with ErrCollectionExists
	err = colSvc.UpdateName(ctx, col2.ID, &name1)
	require.ErrorIs(t, err, services.ErrCollectionExists)

	// 11. Delete collection
	err = colSvc.Delete(ctx, col1.ID)
	require.NoError(t, err)

	_, err = colSvc.FindByID(ctx, col1.ID)
	require.ErrorIs(t, err, services.ErrCollectionNotFound)

	// 12. Delete non-existent collection
	err = colSvc.Delete(ctx, col1.ID)
	require.ErrorIs(t, err, services.ErrCollectionNotFound)
}

func TestCollectionService_Resolve(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "resolve_test.db")

	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	colSvc := services.NewCollectionService(db)

	// Empty store resolution
	_, err = colSvc.Resolve(ctx, "")
	require.ErrorIs(t, err, services.ErrCollectionNotFound)

	name1 := "col-one"
	col1 := &services.Collection{
		ID:          uuid.NewV7(),
		Name:        &name1,
		QueryHash:   services.ComputeQueryHash("q1", 10),
		Query:       "q1",
		ResultLimit: 10,
	}
	require.NoError(t, colSvc.Create(ctx, col1))

	name2 := "col-two"
	col2 := &services.Collection{
		ID:          uuid.NewV7(),
		Name:        &name2,
		QueryHash:   services.ComputeQueryHash("q2", 20),
		Query:       "q2",
		ResultLimit: 20,
	}
	require.NoError(t, colSvc.Create(ctx, col2))

	// Resolve latest (target "")
	latest, err := colSvc.Resolve(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, col2.ID, latest.ID)

	// Resolve by UUID
	byID, err := colSvc.Resolve(ctx, col1.ID.String())
	require.NoError(t, err)
	assert.Equal(t, col1.ID, byID.ID)

	// Resolve by Name
	byName, err := colSvc.Resolve(ctx, "col-one")
	require.NoError(t, err)
	assert.Equal(t, col1.ID, byName.ID)

	// Resolve nonexistent
	_, err = colSvc.Resolve(ctx, "nonexistent")
	require.ErrorIs(t, err, services.ErrCollectionNotFound)
}
