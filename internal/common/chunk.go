package common

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"golang.org/x/sync/errgroup"
)

// split input slice into parts of `chunkSize`, if 0 is given it will only use 1 chunk
// call worker func for each one in a different goroutine concurrently
// will call done callback for each chunk from inside a lock to make it safe
func ConcurrentChunks[
	In any,
	Out any,
](
	input []In,
	chunkSize int,
	chunkWorkerFunc func(chunk []In, chunkIdx int) (Out, error), // runs concurrently, not safe
	chunkDoneCallback func(result Out, chunkIdx int) error, // from within a mutex, safe to perform operations
) error {

	defer ExecutionTimer().Log()
	g, errCtx := errgroup.WithContext(context.Background()) // allows to run goroutines and cancel them if one fails, or wait for all
	m := &sync.Mutex{}

	if chunkSize == 0 {
		chunkSize = len(input)
	}

	chunkCount := int(math.Ceil(float64(len(input)) / float64(chunkSize)))
	slog.Info("splitting to chunks", "chunks", chunkCount, "total", len(input), "chunk-size", chunkSize)

	for i := 0; i < chunkCount; i++ {
		end := (i + 1) * chunkSize
		if end > len(input) {
			end = len(input)
		}
		start := i * chunkSize
		chunk := input[start:end]
		slog.Debug("splitting chunk", "idx", i, "start", start, "end", end, "count", len(chunk))

		g.Go(func(idx int, inputChunk []In, ctx context.Context) func() error {
			return func() (err error) {
				defer func() {
					if panicObj := recover(); panicObj != nil {
						slog.Error("panic caught", "err", panicObj)
						err = fmt.Errorf("panic caught: %v", panicObj)
					}
				}()

				data, err := chunkWorkerFunc(inputChunk, idx)
				if err != nil {
					slog.Error("failed sending bulk", "idx", idx, "err", err)
					return err
				}

				// check group was not cancelled due to error
				select {
				case <-ctx.Done():
					slog.Warn("stopping chunk request due to cancel")
					return nil
				default:
					break
				}

				m.Lock() // allow callback to run 'thread-safe'
				if chunkDoneCallback != nil {
					slog.Debug("calling callback", "chunk", idx)
					err = chunkDoneCallback(data, idx)
				}
				m.Unlock()
				return err
			}
		}(i, chunk, errCtx))
	}

	err := g.Wait()
	return err
}
