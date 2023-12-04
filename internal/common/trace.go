package common

import (
	"context"

	"log/slog"
)

const LevelTrace = slog.Level(-8) // ref: https://pkg.go.dev/golang.org/x/exp/slog

// wrapper for convenience
func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...) // similar implementation to th slog logging functions
}
