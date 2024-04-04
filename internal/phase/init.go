package phase

import (
	"cli/internal/common"
	"cli/internal/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const SealInternalFolderName = ".seal"

func getProjectDir(p string) string {
	if p == "" {
		return common.CliCWD
	}

	return common.GetAbsDirPath(p)
}

func getTargetFile(p string) string {
	if p == "" {
		return ""
	}

	abs, _ := filepath.Abs(p) // ignoring err, propagated from internal call to os.Cwd

	f, err := os.Stat(abs)
	if err != nil || f.IsDir() {
		slog.Debug("bad target path", "err", err, "path", abs) // ignoring error case here, same logic
		return ""
	}

	// strip input from file component
	return abs
}

func validateProjectName(name string) string {
	// validate name according to BE limitations
	if name == "" {
		return "empty string is not allowed"
	}

	if len(name) > 255 {
		return "name must not exceed 255 characters"
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9_\-\.]*$`)
	if !re.MatchString(name) {
		return "can only contain a letter, digit, underscore, hyphen or period"
	}

	return ""
}

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
