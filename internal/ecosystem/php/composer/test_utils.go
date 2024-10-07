//go:build mock
// +build mock

// this will only be bundled when running unit-tests

package composer

import (
	"os"
	"path/filepath"
)

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}
