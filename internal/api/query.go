package api

import (
	"cli/internal/common"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

type RemoteOverrideQuery struct {
	LibraryId            string  `json:"libray_id"`
	OriginVersionId      string  `json:"origin_version_id"`
	RecommendedVersionId *string `json:"recommended_version_id"` // could be null
}

type ChunkDownloadedCallback func(chunk []PackageVersion, idx int)

const MaxDependencyChunkSize = 800
const MaxRemoteOverrideChunkSize = 800

var NonExistentProjectError = common.NewPrintableError("specified project does not exist")
var MissingTokenForApiRequest = errors.New("missing authentication token for querying remote config")

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

	data, statusCode, err := sendSealApiRequest[BulkCheckRequest, Page[PackageVersion]](
		s.Client,
		"POST",
		"/unauthenticated/v1/bulk",
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

func (s Server) sendBulkRequestAuth(request *BulkCheckRequest, queryType PackageQueryType, project string, generateActivity bool) (*Page[PackageVersion], error) {
	params := []StringPair{}
	if queryType == OnlyFixed {
		params = append(params, StringPair{Name: "fixed", Value: "1"})
	} else {
		params = append(params, StringPair{Name: "fixed", Value: "0"})
	}

	if generateActivity {
		common.Trace("will instruct server to generate activity for the requested packages")
		params = append(params, StringPair{Name: "store", Value: "1"}) // defaults to false
	}

	if s.AuthToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers := []StringPair{BuildBasicAuthHeader(s.AuthToken)}

	data, statusCode, err := sendSealApiRequest[BulkCheckRequest, Page[PackageVersion]](
		s.Client,
		"POST",
		fmt.Sprintf("/authenticated/v1/scan/%s", project),
		request,
		headers,
		params,
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

// performs the BE request to get the approved remote config
func (s Server) sendRemoteFixesQuery(query []RemoteOverrideQuery, project string) (*Page[PackageVersion], error) {

	var headers []StringPair

	if s.AuthToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers = []StringPair{BuildBasicAuthHeader(s.AuthToken)}
	common.Trace("sending auth header in bulk request")

	data, statusCode, err := sendSealApiRequest[[]RemoteOverrideQuery, Page[PackageVersion]](
		s.Client,
		"POST",
		fmt.Sprintf("/authenticated/v1/fixes/remote/%s", project),
		&query,
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		if statusCode == 404 {
			// specific case for non-existent project
			return nil, NonExistentProjectError
		}

		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

func (s Server) FetchPackagesInfo(deps []common.Dependency, metadata Metadata, queryType PackageQueryType, chunkDone ChunkDownloadedCallback) (*[]PackageVersion, error) {
	allVersions := make([]PackageVersion, 0, len(deps))
	chunkSize := s.BulkChunkSize
	if chunkSize == 0 {
		chunkSize = MaxDependencyChunkSize
	}

	err := common.ConcurrentChunks(deps, chunkSize,
		func(chunk []common.Dependency, chunkIdx int) (*Page[PackageVersion], error) {
			return s.sendBulkRequest(&BulkCheckRequest{
				Metadata: metadata,
				Entries:  chunk,
			}, queryType)
		},
		func(data *Page[PackageVersion], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allVersions = append(allVersions, data.Items...)
			if chunkDone != nil {
				chunkDone(data.Items, chunkIdx)
			}
			return nil
		})

	return &allVersions, err
}

func (s Server) FetchPackagesInfoAuth(deps []common.Dependency, metadata Metadata, queryType PackageQueryType, chunkDone ChunkDownloadedCallback, project string, generateActivity bool) (*[]PackageVersion, error) {
	allVersions := make([]PackageVersion, 0, len(deps))
	chunkSize := s.BulkChunkSize
	if chunkSize == 0 {
		chunkSize = MaxDependencyChunkSize
	}

	err := common.ConcurrentChunks(deps, chunkSize,
		func(chunk []common.Dependency, chunkIdx int) (*Page[PackageVersion], error) {
			return s.sendBulkRequestAuth(&BulkCheckRequest{
				Metadata: metadata,
				Entries:  chunk,
			}, queryType, project, generateActivity)
		},
		func(data *Page[PackageVersion], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allVersions = append(allVersions, data.Items...)
			if chunkDone != nil {
				chunkDone(data.Items, chunkIdx)
			}
			return nil
		})

	return &allVersions, err
}

func (s Server) FetchOverriddenPackagesInfo(query []RemoteOverrideQuery, project string, chunkDone ChunkDownloadedCallback) (*[]PackageVersion, error) {
	allVersions := make([]PackageVersion, 0, len(query))
	chunkSize := s.BulkChunkSize
	if chunkSize == 0 {
		chunkSize = MaxRemoteOverrideChunkSize
	}

	err := common.ConcurrentChunks(query, chunkSize,
		func(chunk []RemoteOverrideQuery, chunkIdx int) (*Page[PackageVersion], error) {
			return s.sendRemoteFixesQuery(query, project)
		},
		func(data *Page[PackageVersion], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allVersions = append(allVersions, data.Items...)
			if chunkDone != nil {
				chunkDone(data.Items, chunkIdx)
			}
			return nil
		})

	return &allVersions, err
}
