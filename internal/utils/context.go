package utils

type ContextKey string

const (
	RootDirKey      ContextKey = "rootDir"
	AnalysisPathKey ContextKey = "analysisPath"
	SearchPathKey   ContextKey = "searchPath"
)
