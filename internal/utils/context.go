package utils

import (
	"context"
	"path/filepath"
)

type ContextKey string

const (
	rootDirKey      ContextKey = "rootDir"
	analysisPathKey ContextKey = "analysisPath"
)

func WithRootDir(ctx context.Context, rootDir string) context.Context {
	analysisPath := filepath.Join(rootDir, ".bibx", "analysis.json.gz")
	ctx = context.WithValue(ctx, rootDirKey, rootDir)
	ctx = context.WithValue(ctx, analysisPathKey, analysisPath)
	return ctx
}

func GetRootDir(ctx context.Context) (string, bool) {
	rootDir, ok := ctx.Value(rootDirKey).(string)
	return rootDir, ok
}

func GetAnalysisPath(ctx context.Context) (string, bool) {
	analysisPath, ok := ctx.Value(analysisPathKey).(string)
	return analysisPath, ok
}
