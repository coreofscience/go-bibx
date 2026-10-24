package utils

import (
	"context"
	"path/filepath"
)

// ContextKey represents context keys used for storing and retrieving configuration values.
type ContextKey string

const (
	rootDirKey ContextKey = "rootDir"
	dbPathKey  ContextKey = "dbPath"
)

// WithRootDir attaches the root directory and default SQLite DB path to the context.
func WithRootDir(ctx context.Context, rootDir string) context.Context {
	dbPath := filepath.Join(rootDir, ".bibx", "bibx.db")
	ctx = context.WithValue(ctx, rootDirKey, rootDir)
	ctx = context.WithValue(ctx, dbPathKey, dbPath)
	return ctx
}

// WithDBPath overrides the database path in the context.
func WithDBPath(ctx context.Context, dbPath string) context.Context {
	return context.WithValue(ctx, dbPathKey, dbPath)
}

// GetRootDir retrieves the root directory from the context if present.
func GetRootDir(ctx context.Context) (string, bool) {
	rootDir, ok := ctx.Value(rootDirKey).(string)
	return rootDir, ok
}

// GetDBPath retrieves the SQLite database file path from the context if present.
func GetDBPath(ctx context.Context) (string, bool) {
	dbPath, ok := ctx.Value(dbPathKey).(string)
	return dbPath, ok
}
