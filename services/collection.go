package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/repository"
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
	ID          uuid.UUID `json:"id"`
	QueryHash   [32]byte  `json:"query_hash"`
	Query       string    `json:"query"`
	ResultLimit int       `json:"result_limit"`
	Parameters  []byte    `json:"parameters,omitempty"`
	Name        *string   `json:"name,omitempty"`
	WorkCount   int       `json:"work_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CollectionService defines collection lifecycle and resolution operations.
type CollectionService interface {
	FindByHash(ctx context.Context, hash [32]byte) (*Collection, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Collection, error)
	FindByName(ctx context.Context, name string) (*Collection, error)
	Resolve(ctx context.Context, target string) (*Collection, error)
	List(ctx context.Context) ([]*Collection, error)
	Create(ctx context.Context, col *Collection) error
	UpdateName(ctx context.Context, id uuid.UUID, name *string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SQLiteCollectionService implements [CollectionService] backed by [repository.Queries].
type SQLiteCollectionService struct {
	queries *repository.Queries
}

// NewCollectionService creates a [*SQLiteCollectionService] backed by the provided [repository.DBTX].
func NewCollectionService(db repository.DBTX) *SQLiteCollectionService {
	return &SQLiteCollectionService{
		queries: repository.New(db),
	}
}

// NewCollectionServiceWithQueries creates a [*SQLiteCollectionService] wrapping existing [*repository.Queries].
func NewCollectionServiceWithQueries(queries *repository.Queries) *SQLiteCollectionService {
	return &SQLiteCollectionService{
		queries: queries,
	}
}

// FindByHash finds a collection matching the given query hash.
func (s *SQLiteCollectionService) FindByHash(ctx context.Context, hash [32]byte) (*Collection, error) {
	c, err := s.queries.GetCollectionByHash(ctx, hash[:])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection by hash: %w", err)
	}
	return toCollection(c), nil
}

// FindByID finds a collection by its UUID.
func (s *SQLiteCollectionService) FindByID(ctx context.Context, id uuid.UUID) (*Collection, error) {
	c, err := s.queries.GetCollectionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection by id: %w", err)
	}
	return toCollection(c), nil
}

// FindByName finds a collection by its human-friendly name alias.
func (s *SQLiteCollectionService) FindByName(ctx context.Context, name string) (*Collection, error) {
	c, err := s.queries.GetCollectionByName(ctx, sql.NullString{String: name, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection by name: %w", err)
	}
	return toCollection(c), nil
}

// Resolve resolves a collection by target string.
// If target is empty, it returns the most recent collection.
// If target parses as a valid UUID, it queries by ID.
// If not found or if target is not a valid UUID, it queries by name alias.
func (s *SQLiteCollectionService) Resolve(ctx context.Context, target string) (*Collection, error) {
	if target == "" {
		cols, err := s.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list collections: %w", err)
		}
		if len(cols) == 0 {
			return nil, ErrCollectionNotFound
		}
		return cols[0], nil
	}

	if id, err := uuid.Parse(target); err == nil {
		col, err := s.FindByID(ctx, id)
		if err == nil {
			return col, nil
		}
		if !errors.Is(err, ErrCollectionNotFound) {
			return nil, fmt.Errorf("failed to find collection by id: %w", err)
		}
	}

	col, err := s.FindByName(ctx, target)
	if err != nil {
		if errors.Is(err, ErrCollectionNotFound) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to find collection by name: %w", err)
	}
	return col, nil
}

// List returns all collections ordered by creation timestamp descending.
func (s *SQLiteCollectionService) List(ctx context.Context) ([]*Collection, error) {
	items, err := s.queries.ListCollections(ctx, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	cols := make([]*Collection, len(items))
	for i, item := range items {
		cols[i] = toCollection(item)
	}
	return cols, nil
}

// Create inserts a new collection record into the database.
func (s *SQLiteCollectionService) Create(ctx context.Context, col *Collection) error {
	if col == nil {
		return errors.New("cannot create nil collection")
	}
	if col.ID == uuid.Nil() {
		col.ID = uuid.NewV7()
	}

	var nameNull sql.NullString
	if col.Name != nil {
		nameNull = sql.NullString{String: *col.Name, Valid: true}
	}

	err := s.queries.CreateCollection(ctx, repository.CreateCollectionParams{
		ID:          col.ID,
		QueryHash:   col.QueryHash[:],
		Query:       col.Query,
		ResultLimit: int64(col.ResultLimit),
		Parameters:  col.Parameters,
		Name:        nameNull,
		WorkCount:   sql.NullInt64{Int64: int64(col.WorkCount), Valid: true},
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrCollectionExists
		}
		return fmt.Errorf("failed to create collection: %w", err)
	}
	return nil
}

// UpdateName updates the human-friendly name alias of an existing collection.
func (s *SQLiteCollectionService) UpdateName(ctx context.Context, id uuid.UUID, name *string) error {
	if _, err := s.queries.GetCollectionByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollectionNotFound
		}
		return fmt.Errorf("failed to check collection before updating name: %w", err)
	}

	if err := s.queries.UpdateCollectionName(ctx, repository.UpdateCollectionNameParams{
		Name: toNullString(name),
		ID:   id,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrCollectionExists
		}
		return fmt.Errorf("failed to update collection name: %w", err)
	}
	return nil
}

// Delete removes a collection and cascades to its nodes and links.
func (s *SQLiteCollectionService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.queries.GetCollectionByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollectionNotFound
		}
		return fmt.Errorf("failed to check collection before deletion: %w", err)
	}

	if err := s.queries.DeleteCollectionById(ctx, id); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// NormalizeQuery canonicalizes a search query: trims leading/trailing whitespace,
// transforms to lowercase, and collapses multiple consecutive spaces into a single space.
func NormalizeQuery(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	return multiSpaceRegex.ReplaceAllString(q, " ")
}

// queryFingerprintVersion changes whenever the works fetched for a query change, so
// that old queries fetch again instead of returning collections built the old way.
const queryFingerprintVersion = 2

type queryFingerprint struct {
	Version int    `json:"version"`
	Limit   int    `json:"limit"`
	Query   string `json:"query"`
}

// ComputeQueryHash calculates a deterministic SHA-256 idempotency key for a query and limit.
func ComputeQueryHash(query string, limit int) [32]byte {
	norm := NormalizeQuery(query)
	raw, _ := json.Marshal(queryFingerprint{
		Version: queryFingerprintVersion,
		Limit:   limit,
		Query:   norm,
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

func toCollection(c repository.Collection) *Collection {
	var hash [32]byte
	if b, ok := c.QueryHash.([]byte); ok {
		copy(hash[:], b)
	}

	var namePtr *string
	if c.Name.Valid {
		namePtr = &c.Name.String
	}

	var createdAt, updatedAt time.Time
	if c.CreatedAt.Valid {
		createdAt = c.CreatedAt.Time.UTC()
	}
	if c.UpdatedAt.Valid {
		updatedAt = c.UpdatedAt.Time.UTC()
	}

	return &Collection{
		ID:          c.ID,
		QueryHash:   hash,
		Query:       c.Query,
		ResultLimit: int(c.ResultLimit),
		Parameters:  c.Parameters,
		Name:        namePtr,
		WorkCount:   int(c.WorkCount.Int64),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}
