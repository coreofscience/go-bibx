package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/models"
)

var (
	// ErrCollectionNotFound is returned when a requested collection is not present in the store.
	ErrCollectionNotFound = errors.New("collection not found")

	// ErrCollectionExists is returned when attempting to create a collection with a duplicate name or query hash.
	ErrCollectionExists = errors.New("collection already exists")

	// multiSpaceRegex matches consecutive whitespace runs.
	multiSpaceRegex = regexp.MustCompile(`\s+`)
)

// Collection represents a bibliographic inquiry collection stored in the database.
type Collection struct {
	ID           uuid.UUID `json:"id"`
	QueryHash    [32]byte  `json:"query_hash"`
	Query        string    `json:"query"`
	ResultLimit  int       `json:"result_limit"`
	Parameters   []byte    `json:"parameters,omitempty"`
	Name         *string   `json:"name,omitempty"`
	ArticleCount int       `json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Store defines the storage engine contract for collections, articles, and graph analyses.
type Store interface {
	// Collection management
	FindCollectionByHash(ctx context.Context, hash [32]byte) (*Collection, error)
	FindCollectionByID(ctx context.Context, id uuid.UUID) (*Collection, error)
	FindCollectionByName(ctx context.Context, name string) (*Collection, error)
	ListCollections(ctx context.Context) ([]*Collection, error)
	CreateCollection(ctx context.Context, col *Collection) error
	DeleteCollection(ctx context.Context, id uuid.UUID) error

	// Graph persistence and queries
	StoreAnalysis(ctx context.Context, collectionID uuid.UUID, a *models.Analysis) error
	LoadAnalysis(ctx context.Context, collectionID uuid.UUID) (*models.Analysis, error)
	QueryCategory(ctx context.Context, collectionID uuid.UUID, category string, limit int) ([]*models.Result, error)
	Search(ctx context.Context, collectionID uuid.UUID, query string, limit int) ([]*models.Node, error)

	Close() error
}

// NormalizeQuery canonicalizes a search query: trims leading/trailing whitespace,
// transforms to lowercase, and collapses multiple consecutive spaces into a single space.
func NormalizeQuery(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	return multiSpaceRegex.ReplaceAllString(q, " ")
}

type queryFingerprint struct {
	Limit int    `json:"limit"`
	Query string `json:"query"`
}

// ComputeQueryHash calculates a deterministic SHA-256 idempotency key for a query and limit.
func ComputeQueryHash(query string, limit int) [32]byte {
	norm := NormalizeQuery(query)
	raw, _ := json.Marshal(queryFingerprint{
		Limit: limit,
		Query: norm,
	})
	return sha256.Sum256(raw)
}

// UUIDToBytes returns the raw 16-byte slice of a uuid.UUID for BLOB storage.
func UUIDToBytes(id uuid.UUID) []byte {
	return id[:]
}

// ScanUUID parses a SQLite value (either []byte of length 16 or standard string) into a uuid.UUID.
func ScanUUID(src any) (uuid.UUID, error) {
	switch v := src.(type) {
	case []byte:
		if len(v) != 16 {
			return uuid.Nil(), fmt.Errorf("invalid uuid blob length: %d (expected 16)", len(v))
		}
		var id uuid.UUID
		copy(id[:], v)
		return id, nil
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil(), fmt.Errorf("failed to parse uuid string: %w", err)
		}
		return id, nil
	default:
		return uuid.Nil(), fmt.Errorf("unsupported uuid type: %T", src)
	}
}

// ResolveCollection resolves a collection from the store by target string.
// If target is empty, it returns the most recent collection.
// If target parses as a valid UUID, it queries by ID.
// If not found or if target is not a valid UUID, it queries by name.
func ResolveCollection(ctx context.Context, store Store, target string) (*Collection, error) {
	if target == "" {
		cols, err := store.ListCollections(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list collections: %w", err)
		}
		if len(cols) == 0 {
			return nil, ErrCollectionNotFound
		}
		return cols[0], nil
	}

	if id, err := uuid.Parse(target); err == nil {
		col, err := store.FindCollectionByID(ctx, id)
		if err == nil {
			return col, nil
		}
		if !errors.Is(err, ErrCollectionNotFound) {
			return nil, fmt.Errorf("failed to find collection by id: %w", err)
		}
	}

	col, err := store.FindCollectionByName(ctx, target)
	if err != nil {
		if errors.Is(err, ErrCollectionNotFound) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to find collection by name: %w", err)
	}
	return col, nil
}
