package common

import (
	"log/slog"
	"runtime"
	"time"
)

type Timer struct {
	start    time.Time
	file     string
	no       int
	funcName string
}

func (t Timer) Log() {
	slog.Debug("function runtime",
		"total_ms",
		time.Since(t.start).Milliseconds(),
		"file",
		t.file,
		"line_no",
		t.no,
		"func",
		t.funcName)
}

func ExecutionTimer() Timer {
	pc, file, no, ok := runtime.Caller(1)
	if !ok {
		panic("not ok runtime caller")
	}
	
	function := runtime.FuncForPC(pc)

	return Timer{start: time.Now(), file: file, no: no, funcName: function.Name()}
}
