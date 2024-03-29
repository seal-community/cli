package api

import (
	"cli/internal/common"
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"
)

type Server struct {
	Client        http.Client
	AuthToken     string
	BulkChunkSize int
}

type PackageQueryType int

const (
	OnlyVulnerable PackageQueryType = iota
	OnlyFixed      PackageQueryType = iota
	// futue support for query all
)

type ChunkDownloadedCallback func(chunk []PackageVersion, idx int, total int)

const MaxDependencyChunkSize = 800

func (s Server) sendBulkRequest(request *BulkCheckRequest, queryType PackageQueryType) (*Page[PackageVersion], error) {
	var param StringPair

	if queryType == OnlyFixed {
		param = StringPair{Name: "fixed", Value: "1"}
	} else {
		param = StringPair{Name: "fixed", Value: "0"}
	}

	var headers []StringPair

	if s.AuthToken != "" {
		// send token if we have it configured
		headers = []StringPair{BuildBasicAuthHeader(s.AuthToken)}
		common.Trace("sending auth header in bulk request")
	}

	data, statusCode, err := sendApiRequest[BulkCheckRequest, Page[PackageVersion]](
		s.Client,
		"POST",
		"/unauthenticated/artifact-management/v1/library_versions/bulk",
		request,
		headers,
		[]StringPair{param},
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

func (s Server) CheckVulnerablePackages(deps []common.Dependency, metadata Metadata, chunkDone ChunkDownloadedCallback) (*[]PackageVersion, error) {
	return s.FetchPackagesInfo(deps, metadata, OnlyVulnerable, chunkDone)
}

func (s Server) GetFixedPackages(deps []common.Dependency, metadata Metadata, chunkDone ChunkDownloadedCallback) (*[]PackageVersion, error) {
	return s.FetchPackagesInfo(deps, metadata, OnlyFixed, chunkDone)
}

func (s Server) FetchPackagesInfo(deps []common.Dependency, metadata Metadata, queryType PackageQueryType, chunkDone ChunkDownloadedCallback) (*[]PackageVersion, error) {
	defer common.ExecutionTimer().Log()
	g, errCtx := errgroup.WithContext(context.Background()) // allows to run goroutines and cancel them if one fails, or wait for all
	m := &sync.Mutex{}

	chunkSize := s.BulkChunkSize
	if chunkSize == 0 {
		chunkSize = MaxDependencyChunkSize
	}

	chunkCount := int(math.Ceil(float64(len(deps)) / float64(chunkSize)))
	allVersions := make([]PackageVersion, 0, len(deps))

	slog.Info("sending dependencies in chunks", "chunks", chunkCount, "total-deps", len(deps), "chunk-size", chunkSize)
	for i := 0; i < chunkCount; i++ {
		end := (i + 1) * chunkSize
		if end > len(deps) {
			end = len(deps)
		}
		start := i * chunkSize
		chunk := deps[start:end]
		slog.Debug("splitting chunk", "idx", i, "start", start, "end", end, "count", len(chunk))

		g.Go(func(idx int, depsChunk []common.Dependency, ctx context.Context) func() error {
			return func() (err error) {
				defer func() {
					if panicObj := recover(); panicObj != nil {
						slog.Error("panic caught", "err", panicObj)
						err = fmt.Errorf("panic caught: %v", panicObj)
					}
				}()

				// this routine could run in parallel
				data, err := s.sendBulkRequest(&BulkCheckRequest{
					Metadata: metadata,
					Entries:  depsChunk,
				}, queryType)

				// check group was not cancelled due to error
				select {
				case <-ctx.Done():
					slog.Warn("stopping chunk request due to cancel")
					return nil
				default:
					break
				}

				if err != nil {
					slog.Error("failed sending bulk", "idx", idx, "err", err)
					return err
				}

				m.Lock() // to append deps to a list, and allow callback to run 'thread-safe'
				if chunkDone != nil {
					slog.Debug("calling callback", "chunk", idx)
					chunkDone(data.Items, idx, chunkCount)
				}
				allVersions = append(allVersions, data.Items...)
				m.Unlock()
				return nil
			}
		}(i, chunk, errCtx))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sort.SliceStable(allVersions, func(i, j int) bool {
		v1 := allVersions[i]
		v2 := allVersions[j]

		if v1.Library.Name == v2.Library.Name {
			// using lexicographic order for now
			return v1.Version > v2.Version // version in descending
		}

		return v1.Library.Name < v2.Library.Name
	})

	return &allVersions, nil
}
