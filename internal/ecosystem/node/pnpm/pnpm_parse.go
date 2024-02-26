package pnpm

import (
	"bufio"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/node/utils"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	PnpmManager = "pnpm"
)

type PnpmPackage struct {
	Version         string                  `json:"version"`
	Name            string                  `json:"from"` // unknown if cannot be trusted
	Path            string                  `json:"path"`
	Dependencies    map[string]*PnpmPackage `json:"dependencies"`
	DevDependencies map[string]*PnpmPackage `json:"devDependencies"`
}

type pnpmDependencyParser struct {
	config *config.Config // in the future we might want to only pass the npm specific config object
}

func shouldSkip(d common.Dependency) bool {
	if d.Link {
		slog.Info("skipping link dependency", "path", d.DiskPath, "package", d.Name)
		return true
	}

	if d.Name == "" || d.Version == "" {
		slog.Warn("empty dependency")
		return true
	}

	fi, err := os.Lstat(d.DiskPath)
	if err != nil {
		// stat could fail, usually happens when deps were not installed (dev usually); can't distinguish between prod and dev in current implementation so only debug log for it
		slog.Debug("failed getting stat", "path", d.DiskPath, "err", err)
		return true
	}

	// skip symlink for cases:
	//	- manually altered node_modules
	// this won't FP since using pnpm list command gives the paths within .pnpm instead of symlinks it creates for node
	mode := fi.Mode()
	if mode&os.ModeSymlink != 0 {
		slog.Warn("symlink dependency", "path", d.DiskPath)
		return true
	}

	return false
}

func skipToPackages(output *bufio.Reader, projectDir string) error {
	// pnpm's prints to stdout a warning line for failing to replace an env variable
	//		this only happens for the first var due to uncaught exception
	// 		ref: https://github.com/pnpm/pnpm/issues/5914#issuecomment-1378997369
	// will skip first 'project' line, and error line if exists
	firstLine, err := output.ReadString('\n') // this should work regardless of OS, will skip \r
	if firstLine == "" || err != nil {
		// unknown input 'format'
		slog.Warn("could not read first line of pnpm output")
		return io.EOF
	}

	// handle warn message if exists
	if strings.HasPrefix(firstLine, projectDir) {
		return nil // expected output; first line should be the project itself
	}

	slog.Info("skipped first line due to bad pnpm output", "line", firstLine)
	secondLine, err := output.ReadString('\n')
	if secondLine == "" || err != nil {
		slog.Warn("could not read second line of pnpm output")
		return io.EOF
	}

	// this assumes the first listed package is the project itself, starting with the project dir
	if !strings.HasPrefix(secondLine, projectDir) {
		slog.Warn("did not find project perfix in pnpm output", "prefix", projectDir, "line", secondLine)
		return fmt.Errorf("project path prefix missing in pnpm output line %s", secondLine)
	}

	return nil
}

func parseLine(line string, projectDir string) *common.Dependency {
	// format: {diskpath}:{package}@{version}
	// 	 package - can be scoped, limitations: https://docs.npmjs.com/cli/v10/configuring-npm/package-json#name
	//	 dispath - windows/unix path, absolute, can contain may characters
	// ref for pnpm: https://github.com/btea/pnpm/blob/main/reviewing/list/src/renderParseable.ts#L24
	link := false

	verSepIdx := strings.LastIndex(line, "@") // using last index
	if verSepIdx == -1 {
		slog.Warn("did not find @ separator between name and version")
		return nil
	}

	version := line[verSepIdx+1:]
	remainder := line[:verSepIdx]

	packageSepIdx := strings.LastIndex(remainder, ":")
	if packageSepIdx == -1 {
		slog.Warn("did not find : separator between diskpath and package name")
		return nil
	}

	path := remainder[:packageSepIdx]
	pkgName := remainder[packageSepIdx+1:]

	if strings.HasPrefix(version, "link:") {
		// happens when adding folders or using link command
		// unsupported for now, seems to be replacing the version:
		//		{package}@link:{relative-path}
		slog.Warn("unsupported link/folder dependency", "line", line)
		link = true
		version = "" // unsupported, should prevent from using this
	}

	if !strings.HasPrefix(line, projectDir) {
		slog.Warn("external path to dependency", "line", line, "projectdir", projectDir)
		// could be added to struct so we could filter them out better
	}

	return &common.Dependency{
		Name:           pkgName,
		Version:        version,
		PackageManager: mappings.NpmManager, // using NPM here as well for the sake of the BE
		DiskPath:       path,
		Link:           link,
	}
}

func (parser *pnpmDependencyParser) Parse(lsOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)

	if lsOutput == "" {
		// could happen if node_modules exists but empty, before installing dependencies
		slog.Warn("empty output from pnpm, dependencies not installed")
		return nil, common.NewPrintableError("please run pnpm install before using the cli")
	}

	r := bufio.NewReader(strings.NewReader(lsOutput))
	err := skipToPackages(r, projectDir)
	if err != nil {
		slog.Error("failed skipping bad pnpm output", "output", lsOutput)
		return nil, fmt.Errorf("failed skipping bad output from pnpm %w", err) // caller should wrap this error
	}

	scanner := bufio.NewScanner(r) // handles line-endings correctly

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()

		d := parseLine(line, projectDir)
		if d == nil {
			slog.Warn("failed parsing dep", "idx", i, "line", line)
			return nil, fmt.Errorf("failed parsing pnpm output line %d", i) // returning genreal error, up to caller to use fallback
		}

		if shouldSkip(*d) {
			slog.Debug("skipping dep", "name", d.Name, "version", d.Version, "idx", i)
			continue
		}

		// compare actual vs reported version
		// this could happen if we already fixed a package on disk
		// since pnpm does not report the updated versoin
		actualVersion := utils.GetVersion(d.DiskPath)
		if actualVersion != "" && actualVersion != d.Version {
			slog.Info("overwriting version due to mismatch between pnpm report and disk", "actual", actualVersion, "reported", d.DiskPath, "package", d.Name)
			d.Version = actualVersion // using the one in disk could help differentiate between fixed and unfixed instances of same package
		}

		key := d.Id()
		if _, ok := deps[key]; !ok {
			// in case they are not dedup'd
			deps[key] = make([]*common.Dependency, 0, 1)
		}

		deps[key] = append(deps[key], d)
	}

	return deps, nil
}
