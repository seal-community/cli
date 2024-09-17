package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"sync"
	"testing"
)

type fakeBackend struct {
	PackageChunkSize      int
	RemoteConfigChunkSize int

	m                      *sync.Mutex
	QueryPackagesCount     int
	QueryPackagesAuthCount int
	QueryRemoteCount       int
}

func (be *fakeBackend) GetPackageChunkSize() int {
	return be.PackageChunkSize
}

func (be *fakeBackend) GetRemoteConfigChunkSize() int {
	return be.RemoteConfigChunkSize
}

func (be *fakeBackend) QueryPackages(request *api.BulkCheckRequest, queryType api.PackageQueryType) (*api.Page[api.PackageVersion], error) {
	be.m.Lock()
	be.QueryPackagesCount += 1
	be.m.Unlock()
	return &api.Page[api.PackageVersion]{}, nil
}

func (be *fakeBackend) QueryPackagesAuth(request *api.BulkCheckRequest, queryType api.PackageQueryType, generateActivity bool) (*api.Page[api.PackageVersion], error) {
	be.m.Lock()
	be.QueryPackagesAuthCount += 1
	be.m.Unlock()
	return &api.Page[api.PackageVersion]{}, nil
}

func (be *fakeBackend) QueryRemoteConfig(query []api.RemoteOverrideQuery) (*api.Page[api.PackageVersion], error) {
	be.m.Lock()
	be.QueryRemoteCount += 1
	be.m.Unlock()
	return &api.Page[api.PackageVersion]{}, nil
}

func (be *fakeBackend) CheckAuthenticationValid() error {
	return nil
}

func (be *fakeBackend) InitializeProject(displayName string) (*api.ProjectDescriptor, error) {
	panic("not implemented")
}

func TestBulkQuerySingleChunk(t *testing.T) {
	be := &fakeBackend{m: &sync.Mutex{}}
	_, err := fetchPackagesInfo(be, []common.Dependency{
		{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
	},
		api.Metadata{},
		api.OnlyVulnerable,
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if be.QueryPackagesCount != 1 {
		t.Fatalf("wrong number of chunks sent: %d", be.QueryPackagesCount)
	}
}

func TestBulkQueryChunks(t *testing.T) {
	be := &fakeBackend{m: &sync.Mutex{}, PackageChunkSize: 2}
	_, err := fetchPackagesInfo(be,
		[]common.Dependency{
			{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
			{Name: "b", Version: "1.0.0", PackageManager: "mmm"},
			{Name: "c", Version: "0.0.1", PackageManager: "mmm"},
		},
		api.Metadata{},
		api.OnlyVulnerable,
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if be.QueryPackagesCount != 2 {
		t.Fatalf("wrong number of chunks sent: %d", be.QueryPackagesCount)
	}
}

func TestBulkQuerySingleChunkAuth(t *testing.T) {
	be := &fakeBackend{m: &sync.Mutex{}}
	_, err := fetchPackagesInfoAuth(be, []common.Dependency{
		{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
	},
		api.Metadata{},
		api.OnlyVulnerable,
		nil, false,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if be.QueryPackagesAuthCount != 1 {
		t.Fatalf("wrong number of chunks sent: %d", be.QueryPackagesAuthCount)
	}
}

func TestBulkQueryChunksAuth(t *testing.T) {
	be := &fakeBackend{m: &sync.Mutex{}, PackageChunkSize: 2}
	_, err := fetchPackagesInfoAuth(be,
		[]common.Dependency{
			{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
			{Name: "b", Version: "1.0.0", PackageManager: "mmm"},
			{Name: "c", Version: "0.0.1", PackageManager: "mmm"},
		},
		api.Metadata{},
		api.OnlyVulnerable,
		nil, false,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if be.QueryPackagesAuthCount != 2 {
		t.Fatalf("wrong number of chunks sent: %d", be.QueryPackagesAuthCount)
	}
}

func TestOverriddenPackagesSingleChunk(t *testing.T) {
	be := &fakeBackend{m: &sync.Mutex{}}
	_, err := fetchOverriddenPackagesInfo(be, []api.RemoteOverrideQuery{
		{LibraryId: "a", OriginVersionId: "1.2.3", RecommendedVersionId: nil},
	},
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if be.QueryRemoteCount != 1 {
		t.Fatalf("wrong number of chunks sent: %d", be.QueryRemoteCount)
	}
}

func TestOverriddenPackagesChunks(t *testing.T) {
	be := &fakeBackend{m: &sync.Mutex{}, RemoteConfigChunkSize: 2}
	_, err := fetchOverriddenPackagesInfo(be,
		[]api.RemoteOverrideQuery{
			{LibraryId: "a", OriginVersionId: "1.2.3", RecommendedVersionId: nil},
			{LibraryId: "b", OriginVersionId: "1.2.4", RecommendedVersionId: nil},
			{LibraryId: "c", OriginVersionId: "1.2.5", RecommendedVersionId: nil},
		},
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if be.QueryRemoteCount != 2 {
		t.Fatalf("wrong number of chunks sent: %d", be.QueryRemoteCount)
	}
}
