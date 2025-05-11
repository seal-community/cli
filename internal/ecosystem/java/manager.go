package java

import (
	"cli/internal/config"
	"cli/internal/ecosystem/java/gradle"
	"cli/internal/ecosystem/java/maven"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for maven indicator", "file", targetFile, "dir", targetDir)
	if targetFile != "" {
		if maven.IsMavenIndicatorFile(targetFile) {
			slog.Debug("maven manager supports target", "target-file", targetFile, "target-dir", targetDir)
			return maven.NewMavenManager(config, targetFile, targetDir), nil
		}

		if gradle.IsGradleIndicatorFile(targetFile) {
			slog.Debug("gradle manager supports target", "target-file", targetFile, "target-dir", targetDir)
			return gradle.NewGradleManager(config, targetFile, targetDir), nil
		}

		return nil, fmt.Errorf("not a java file indicator")
	}

	mavenFile, err := maven.GetMavenIndicatorFile(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed detecting maven directory %w", err)
	}

	if mavenFile != "" {
		slog.Debug("maven manager supports target", "target-file", targetFile, "target-dir", targetDir)
		m := maven.NewMavenManager(config, mavenFile, targetDir)
		return m, nil
	}

	gradleFile, err := gradle.GetGradleIndicatorFile(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed detecting gradle directory %w", err)
	}

	if gradleFile != "" {
		slog.Debug("gradle manager supports target", "target-file", targetFile, "target-dir", targetDir)
		m := gradle.NewGradleManager(config, gradleFile, targetDir)
		return m, nil
	}

	return nil, fmt.Errorf("failed detecting java directory %w", err)
}
