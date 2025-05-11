package gradle

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const cacheOverrideEnv = "GRADLE_USER_HOME"

func getPatchedGradleWrapperContent(gradlewString string, cacheOverrideLine string) string {
	patchedGradleWrapper := ""
	added := false
	for _, line := range strings.Split(string(gradlewString), "\n") {
		patchedGradleWrapper += line + "\n"
		trimmed := strings.TrimSpace(line)
		if !added && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			patchedGradleWrapper += cacheOverrideLine + "\n"
			added = true
		}
	}

	return strings.TrimSuffix(patchedGradleWrapper, "\n")
}

func verifyGradleWrapper(gradlewString string, cacheOverrideLine string) error {
	if strings.Contains(gradlewString, cacheOverrideLine) {
		slog.Debug("gradlew already patched")
		return nil
	}

	cacheOverridePrefix := fmt.Sprintf("export %s=", cacheOverrideEnv)
	if strings.Contains(gradlewString, cacheOverridePrefix) {
		slog.Error("gradlew already patched with different cache dir")
		return fmt.Errorf("gradlew already patched with different cache dir")
	}

	return nil
}

func (m *GradlePackageManager) patchGradleWrapper() error {
	gradlewPath := filepath.Join(m.runner.targetDir, m.runner.gradleExe)

	gradlewContent, err := os.ReadFile(gradlewPath)
	if err != nil {
		slog.Error("failed reading gradlew file", "file", gradlewPath, "err", err)
		return err
	}

	cacheOverrideLine := fmt.Sprintf("export %s=\"%s\"", cacheOverrideEnv, m.privateHomeDir)
	gradlewString := string(gradlewContent)
	if err := verifyGradleWrapper(gradlewString, cacheOverrideLine); err != nil {
		slog.Error("invalid gradlew content for patching", "err", err)
		return err
	}

	patchedGradleWrapperString := getPatchedGradleWrapperContent(gradlewString, cacheOverrideLine)
	if gradlewString == patchedGradleWrapperString {
		slog.Error("could not find gradlew code start")
		return fmt.Errorf("could not find gradlew code start")
	}

	return common.DumpBytes(gradlewPath, []byte(patchedGradleWrapperString))
}
