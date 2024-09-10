package java

import (
	"cli/internal/config"
	"cli/internal/ecosystem/java/maven"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for maven indicator", "file", targetFile, "dir", targetDir)
	if targetFile != "" {
		if maven.IsMavenIndicatorFile(targetFile) {
			slog.Debug("java manager supports target", "target-file", targetFile, "target-dir", targetDir)

			return maven.NewMavenManager(config, targetFile, targetDir), nil
		}

		return nil, fmt.Errorf("not a java file indicator")
	}

	javaFile, err := maven.GetJavaIndicatorFile(targetDir)
	if err != nil || javaFile == "" {
		return nil, fmt.Errorf("failed detecting java directory %w", err)
	}

	slog.Debug("java manager supports target", "target-file", targetFile, "target-dir", targetDir)
	m := maven.NewMavenManager(config, javaFile, targetDir)
	return m, nil
}
