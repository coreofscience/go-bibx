package utils

import (
	"context"
	"path/filepath"
)

// ContextKey represents context keys used for storing and retrieving configuration values.
type ContextKey string

const (
	rootDirKey      ContextKey = "rootDir"
	analysisPathKey ContextKey = "analysisPath"
)

// WithRootDir attaches the root directory and calculated analysis path to the context.
func WithRootDir(ctx context.Context, rootDir string) context.Context {
	analysisPath := filepath.Join(rootDir, ".bibx", "analysis.json.gz")
	ctx = context.WithValue(ctx, rootDirKey, rootDir)
	ctx = context.WithValue(ctx, analysisPathKey, analysisPath)
	return ctx
}

// GetRootDir retrieves the root directory from the context if present.
func GetRootDir(ctx context.Context) (string, bool) {
	rootDir, ok := ctx.Value(rootDirKey).(string)
	return rootDir, ok
}

// GetAnalysisPath retrieves the analysis file path from the context if present.
func GetAnalysisPath(ctx context.Context) (string, bool) {
	analysisPath, ok := ctx.Value(analysisPathKey).(string)
	return analysisPath, ok
}
