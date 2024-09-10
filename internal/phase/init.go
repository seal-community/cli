package phase

import (
	"cli/internal/common"
	"cli/internal/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const SealInternalFolderName = ".seal"

func InitConfiguration(path string) (*config.Config, error) {

	var confFile *os.File
	var confReader io.Reader
	confReader, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("failed opening conf file", "err", err, "path", path)
			return nil, common.NewPrintableError("could not open config file in %s", path)
		}
		slog.Warn("initializing without config file")
		confReader = strings.NewReader("")
	} else {
		defer confFile.Close()
	}

	configuration, err := config.Load(confReader, nil)
	if err != nil {
		slog.Error("failed opening conf file", "err", err, "path", path)
		return nil, common.FallbackPrintableMsg(err, "failed parsing config file")
	}

	return configuration, nil
}

func createInternalSealFolder(projectDir string) (string, error) {
	p := filepath.Join(projectDir, SealInternalFolderName)
	err := os.RemoveAll(p)
	if err != nil {
		return "", err
	}

	slog.Debug("creating tmp folder", "path", p)

	err = os.MkdirAll(p, os.ModePerm) // will allow it if exists
	if err != nil {
		return "", err
	}

	return p, nil
}
