package api

import (
	"cli/internal/common"
	"net/http"
	"sync"
	"testing"
)

func TestBulkQuerySingleChunk(t *testing.T) {
	chunksRequested := 0
	m := &sync.Mutex{}
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
		Validator: func(r *http.Request) {
			m.Lock()
			chunksRequested += 1
			m.Unlock()
		}}
	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client}
	_, err := s.CheckVulnerablePackages([]common.Dependency{
		{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
	},
		Metadata{},
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if chunksRequested != 1 {
		t.Fatalf("wrong number of chunks sent: %d", chunksRequested)
	}
}

func TestBulkQueryChunks(t *testing.T) {
	chunksRequested := 0
	m := &sync.Mutex{}
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
		Validator: func(r *http.Request) {
			m.Lock()
			chunksRequested += 1
			m.Unlock()
		}}
	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, BulkChunkSize: 1}
	_, err := s.CheckVulnerablePackages([]common.Dependency{
		{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
		{Name: "b", Version: "1.0.0", PackageManager: "mmm"},
		{Name: "c", Version: "0.0.1", PackageManager: "mmm"},
	},
		Metadata{},
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if chunksRequested != 3 {
		t.Fatalf("wrong number of chunks sent: %d", chunksRequested)
	}
}
