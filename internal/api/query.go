package api

import (
	"cli/internal/common"
	"errors"
)

type Backend interface {
	GetPackageChunkSize() int      // used for requesting chunks of vulnerable packages from the BE
	GetRemoteConfigChunkSize() int // used for requesting chunks of configured fixes from the BE

	QueryPackages(request *BulkCheckRequest, queryType PackageQueryType) (*Page[PackageVersion], error)

	QueryPackagesAuth(request *BulkCheckRequest, queryType PackageQueryType, generateActivity bool) (*Page[PackageVersion], error)

	QueryRemoteConfig(query []RemoteOverrideQuery) (*Page[PackageVersion], error)

	CheckAuthenticationValid() error

	InitializeProject(displayName string) (*ProjectDescriptor, error)
}

var ArtifactServerUnsupportedMethod = errors.New("unsupported http method")
var NilResponseObjectType = errors.New("bad request response type")

type ArtifactServer interface {
	Get(uri string, params []StringPair, extraHdrs []StringPair) (data []byte, code int, err error)

	// will convert response from json and populate input obj
	// if non-pointer type is passed json marshal will return error
	GetJsonObject(url string, headers []StringPair, params []StringPair, obj any) (int, error)
}

type PackageQueryType int

const (
	OnlyVulnerable PackageQueryType = iota
	OnlyFixed      PackageQueryType = iota
	// future support for query all
)

type RemoteOverrideQuery struct {
	LibraryId            string  `json:"libray_id"` // the API has a typo
	OriginVersionId      string  `json:"origin_version_id"`
	RecommendedVersionId *string `json:"recommended_version_id"` // could be null
}

type ChunkDownloadedCallback func(chunk []PackageVersion, idx int)

var NonExistentProjectError = common.NewPrintableError("specified project does not exist")
var MissingTokenForApiRequest = errors.New("missing authentication token for querying remote config")

type ProjectInitRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type ProjectDescriptor struct {
	ProjectInitRequest
	New bool `json:"is_new"`
}
