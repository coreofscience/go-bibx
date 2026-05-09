package utils

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

func SetDefaultLogger(verbose bool) {
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.Kitchen,
	}))
	slog.SetDefault(logger)
}
