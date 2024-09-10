package project

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"cli/internal/repository"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
)

type ProjectInfo struct {

	// mostly relevant for authenticated flow
	Tag           string // project's unique ID, calculated in the CLI
	NameCandidate string // generated in CLI before we know if the project is new
	FoundLocally  bool   // if project tag was found in configuration
	New           bool   // initialized from remote, if the project was newly created
	RemoteName    string // user friendly name for creating a new project; initialized after server response
}

// used for pretty names
const MaxProjectNameLen = 255

func NormalizeProjectName(name string) string {
	re1 := regexp.MustCompile(`[ /\\]`)
	name = re1.ReplaceAllString(name, "-")

	re2 := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
	name = re2.ReplaceAllString(name, "")

	// Trim the name to the maximum allowed length
	if len(name) > MaxProjectNameLen {
		name = name[:MaxProjectNameLen]
	}

	return name
}

// normalize paths to unix style slashes to allow finding the target files in config even if run from different OS than the one genereated it
func NormalizeTarget(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// format the display name, similarly to what was the legacy project id
func GenerateProjectDisplayName(manager shared.PackageManager, projectDir string) string {
	baseName := manager.GetProjectName()
	if baseName == "" {
		slog.Warn("manager project name not viable, using folder name")
		baseName = filepath.Base(projectDir)
	}

	name := NormalizeProjectName(baseName)
	return name
}

func ValidateProjectId(name string) string {
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

func hashProjectDescriptor(desc string) string {
	shaBytes := sha1.Sum([]byte(desc))
	return strings.ToUpper(hex.EncodeToString(shaBytes[:]))
}

func formatProjectIdForRepo(relativeTargetPath string, remoteUrl string) string {
	remote, err := repository.GetProjectFromRemote(remoteUrl)
	if err != nil {
		slog.Error("failed getting project path from remote", "err", err, "remote-url", remoteUrl)
		return ""
	}

	normalizedPath := filepath.ToSlash(relativeTargetPath) // normalize disk paths to forward slashes
	payload := fmt.Sprintf("%s/%s", remote, normalizedPath)

	return payload
}

// project name could be empty string, however the others should not
func formatProjectIdFallback(projectDirName string, relativeTargetPath string, projectName string) string {
	normalizedPath := filepath.ToSlash(relativeTargetPath) // normalize disk paths to forward slashes
	payload := fmt.Sprintf("%s/%s", projectDirName, normalizedPath)
	if projectName != "" {
		payload = fmt.Sprintf("%s/%s", payload, projectName)
	}

	return payload
}

func findProjectIdByTarget(projMap map[string]config.ProjectInfo, providedTarget string) string {
	// we normalize slashes in case the file was generated on a different OS
	// and yet perform case-sensitive exact comparison just in case it matters
	normPovidedTarget := NormalizeTarget(providedTarget)
	for projId, pi := range projMap {
		for _, confTarget := range pi.Targets {
			normConfigTarget := NormalizeTarget(confTarget)
			if normConfigTarget == normPovidedTarget {
				slog.Debug("found project id in config project map", "target", providedTarget, "id", projId, "norm-target", normPovidedTarget)
				return projId
			}
		}
	}

	slog.Debug("target does not appear in config project map", "target", providedTarget, "norm", normPovidedTarget)
	return ""
}

func ChooseProjectId(manager shared.PackageManager, projectDir string, relativeTarget string, userProvidedProjId string, projMap map[string]config.ProjectInfo, remoteRepoUrl string) (string, bool, error) {
	mappedProjectId := findProjectIdByTarget(projMap, relativeTarget) // we might not have any mapping set in config
	// id was provided alongside target file, however it does not match the project map in config
	if mappedProjectId != "" && userProvidedProjId != "" &&
		userProvidedProjId != mappedProjectId {
		slog.Error("project id mismatch in map", "provided", userProvidedProjId, "mapped", mappedProjectId)

		return "", false, common.NewPrintableError("Project ID %s is inconsistent with the ID found in config: %s", userProvidedProjId, mappedProjectId)
	}

	// matches the map according to target file
	if userProvidedProjId != "" &&
		userProvidedProjId == mappedProjectId {
		return userProvidedProjId, true, nil
	}

	// means we found the target file in the map wihtout a user-provided project-id
	if mappedProjectId != "" && userProvidedProjId == "" {
		return mappedProjectId, true, nil
	}

	// means we cannot find this project in the map
	// from this point onwards this project id is considered not-found, user should be warned
	isFound := false
	// not found, possibly new but provided
	if userProvidedProjId != "" && mappedProjectId == "" {
		return userProvidedProjId, isFound, nil

	}

	if remoteRepoUrl != "" { // generate id by repo remote
		payload := formatProjectIdForRepo(relativeTarget, remoteRepoUrl)
		if payload != "" {
			projId := hashProjectDescriptor(payload)
			slog.Info("generated project id from repo", "payload", payload, "id", projId)
			return projId, isFound, nil
		}

		// continuing to use fallback as best-effort to not break exising usage of CLI until repo discovery is stable
		slog.Warn("failed generating project", "remote-url", remoteRepoUrl, "target", relativeTarget)
	}

	// not a repo, generate id using fallback
	payload := formatProjectIdFallback(filepath.Base(projectDir), relativeTarget, manager.GetProjectName())
	projId := hashProjectDescriptor(payload)
	slog.Warn("generated project id from disk fallback", "payload", payload, "id", projId)

	return projId, isFound, nil
}
