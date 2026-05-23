package utils

import (
	"context"
	"path/filepath"
)

type ContextKey string

const (
	RootDirKey      ContextKey = "rootDir"
	AnalysisPathKey ContextKey = "analysisPath"
	SearchPathKey   ContextKey = "searchPath"
	RawPathKey      ContextKey = "rawPath"
)

func WithRootDir(ctx context.Context, rootDir string) context.Context {
	analysisPath := filepath.Join(rootDir, ".bibx", "collection.json.gz")
	searchPath := filepath.Join(rootDir, ".bibx", "search.json.gz")
	rawPath := filepath.Join(rootDir, "raw", "collection")
	ctx = context.WithValue(ctx, RootDirKey, rootDir)
	ctx = context.WithValue(ctx, AnalysisPathKey, analysisPath)
	ctx = context.WithValue(ctx, SearchPathKey, searchPath)
	ctx = context.WithValue(ctx, RawPathKey, rawPath)
	return ctx
}

func GetRootDir(ctx context.Context) (string, bool) {
	rootDir, ok := ctx.Value(RootDirKey).(string)
	return rootDir, ok
}

func GetAnalysisPath(ctx context.Context) (string, bool) {
	analysisPath, ok := ctx.Value(AnalysisPathKey).(string)
	return analysisPath, ok
}

func GetSearchPath(ctx context.Context) (string, bool) {
	searchPath, ok := ctx.Value(SearchPathKey).(string)
	return searchPath, ok
}

func GetRawPath(ctx context.Context) (string, bool) {
	rawPath, ok := ctx.Value(RawPathKey).(string)
	return rawPath, ok
}
