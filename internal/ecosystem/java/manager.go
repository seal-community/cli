package java

import (
	"cli/internal/config"
	"cli/internal/ecosystem/java/maven"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	if targetFile != "" {
		slog.Debug("checking package manager for target file")
		if maven.IsMavenIndicatorFile(targetFile) {
			return maven.NewMavenManager(config, targetFile, targetDir), nil
		}

		return nil, fmt.Errorf("not a java file indicator")
	}

	slog.Debug("checking package manager for target dir")
	javaFile, err := maven.GetJavaIndicatorFile(targetDir)
	if err != nil || javaFile == "" {
		return nil, fmt.Errorf("failed detecting java directory %w", err)
	}

	m := maven.NewMavenManager(config, javaFile, targetDir)
	slog.Info("version: ", "Version", m.GetVersion(targetDir))
	return m, nil
}
