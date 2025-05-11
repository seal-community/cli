package gradle

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
)

const globalGradleExe = "gradle"
const projectGradlewExe = "./gradlew" // unsupported on windows

type gradleRunner struct {
	gradleExe string
	targetDir string
}

// used to bind it to a target dir (either gradlew or use global gradle from path)
func NewGradleRunner(targetDir string, useGradleW bool) *gradleRunner {
	exe := globalGradleExe
	if useGradleW {
		exe = projectGradlewExe
	}

	return &gradleRunner{
		gradleExe: exe,
		targetDir: targetDir,
	}
}

func (g *gradleRunner) getStdout(params ...string) string {
	res, err := common.RunCmdWithArgs(g.targetDir, g.gradleExe, params...)
	if err != nil {
		slog.Error("failed running gradle command", "err", err, "dir", g.targetDir, "exe", g.gradleExe, "params", params)
		return ""
	}

	if res.Code != 0 {
		slog.Error("gradle returned non zero return code", "result", res, "exitcode", res.Code, "exe", g.gradleExe, "params", params)
		return ""
	}

	return res.Stdout
}

// root project should be empty string
func formatTaskForProject(project, task string) string {
	return fmt.Sprintf("%s:%s", project, task)
}

// tested on 8.14
// tested on 7.6.4 (--no-continue is not supported)
// if a project has no dependencies defined with a given scope this could fail, and return empty string
func (g *gradleRunner) Dependencies(project string, scope ScopeConfiguration) string {
	// https://docs.gradle.org/current/userguide/viewing_debugging_dependencies.html#understanding_output_annotations
	projectTask := formatTaskForProject(project, "dependencies")
	return g.getStdout(
		projectTask,
		"--configuration", string(scope),
		"--console", "plain",
		"-q",
	)
}

func (g *gradleRunner) Version() string {
	return g.getStdout("--version", "--console", "plain")
}

// used to grab the current cache gradle is using from its output
// tested on 8.14
// tested on 7.6.4 (--no-continue is not supported)
func (g *gradleRunner) Status() string {

	return g.getStdout(
		"--info",   // will cause log lines to be emitted that tell us where this gradle's cache dir is
		"--status", // seems harmless and prints a short output
		"--console", "plain",
	)
}

// output for the querying the allprojects property
// the root project seems to be the first, but behaves the same if explicitly named in gradle.settings or not
// tested on 8.14
// tested on 7.6.4 (--no-continue is not supported)
func (g *gradleRunner) Projects() string {
	return g.getStdout(
		"properties",
		"--property",
		"allprojects",
		"--console", "plain", // no colors
		"-q", // less output
	)
}
