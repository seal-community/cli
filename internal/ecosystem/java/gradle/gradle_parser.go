package gradle

import (
	"cli/internal/common"
	"cli/internal/ecosystem/java/utils"
	"log/slog"
	"os"
	"regexp"
)

type ScopeConfiguration string

// see https://docs.gradle.org/current/userguide/dependency_configurations.html
const (
	CompileClasspath ScopeConfiguration = "compileClasspath"
)

// this expects output from running gradle with `--configuration {scope} --console plain -q`
// it accepts a scope param to initialize the dependency correctly
// if there's a lock file this should filter out the constraint lines / failed lines from bad env
func parsePackages(stdout string, scope ScopeConfiguration) []utils.JavaPackageInfo {
	var re = regexp.MustCompile(`(?m)--- (\w.+):(.+):([^\{\s]+)`)
	packages := make([]utils.JavaPackageInfo, 0, 1)

	matches := re.FindAllStringSubmatch(stdout, -1)
	for _, m := range matches {
		pi := utils.JavaPackageInfo{
			OrgName:      m[1],
			ArtifactName: m[2],
			Version:      m[3],
			Scope:        string(scope),
		}
		common.Trace("found package", "package-info", pi)
		packages = append(packages, pi)
	}

	return packages
}

func parseVersionOutput(stdout string) string {
	var re = regexp.MustCompile(`\nGradle\s+(.*)\b`)

	matches := re.FindStringSubmatch(stdout)
	if len(matches) != 2 {
		slog.Error("unexpected number of matches", "count", len(matches))
		return ""
	}

	version := matches[1]
	return version
}

// returns the names of all subprojects and root project (empty string) without ':' prefix
// example fragment of input:
//
//	...`[root project 'example-app', project ':app']`...
//
// will return the following project names:
// - "" (root)
// - "app"
func parseProjectsOutput(stdout string) []string {
	// grabbing the names without the ':' prefix, we should add it later if needed to other commands
	// this way we can filter out both other parts of the output as well as the root project's name which we should keep as empty string
	var re = regexp.MustCompile(`project\s':(.+?)'`)

	projects := make([]string, 0, 1)
	projects = append(projects, "") // adding root project manually
	// running `gradle :depenedencies` is for root, using the actual name (that is returned like 'example-app' yields error)

	matches := re.FindAllStringSubmatch(stdout, -1)
	// since we ignore the root, we might noy get any matches if for some reason it is supported

	for _, matchRes := range matches {
		// this could theoretically add dups
		rawName := matchRes[1]
		slog.Debug("ofound project", "raw-name", rawName)
		projects = append(projects, rawName)
	}

	return projects
}

// treats the first line of output as a log line gradle prints
// which tells us the current cache path with some extra elements
// tested on gradle 8.14
// IMPORTANT: no support for windows
func parseHomeDir(stdout string) string {

	var re = regexp.MustCompile(`(?i)Initialized native services in: (.+)` + string(os.PathSeparator))
	matches := re.FindStringSubmatch(stdout)
	if len(matches) != 2 {
		slog.Error("unexpected number of matches", "count", len(matches))
		return ""
	}

	cachePath := matches[1]
	return cachePath
}
