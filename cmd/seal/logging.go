package main

import (
	"cli/internal/common"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

func setupLogging(logfile *os.File, verbosity int) {

	level := slog.LevelWarn
	addSource := false
	var output io.Writer = logfile

	if verbosity > 0 {
		output = io.MultiWriter(os.Stdout, logfile) // will print to console as well as file
		level = slog.LevelInfo
		addSource = true
		if verbosity > 1 {
			level = slog.LevelDebug
		}

		if verbosity > 2 {
			level = common.LevelTrace // http body response etc, will show as DBG due to handler formatting
		}
	}

	handler := tint.NewHandler(output, &tint.Options{
		TimeFormat: time.DateTime,
		AddSource:  addSource,
		Level:      level,
	})

	logger := slog.New(handler)

	// override default global logger
	slog.SetDefault(logger)
}

func hideLogging() {
	// set logging to be discarded until properly set up
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// override default global logger
	slog.SetDefault(logger)
}
