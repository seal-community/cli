//go:build windows

package npm

import (
	"os"
	"path/filepath"
)

const PlatformTestDir = "win"
const defaultTestProjectDir = "C:\\Users\\mococo\\proj"

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", PlatformTestDir, name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}
