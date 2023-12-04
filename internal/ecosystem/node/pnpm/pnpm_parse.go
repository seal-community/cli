package pnpm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/shared"
	"encoding/json"
	"fmt"
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

func (parser *pnpmDependencyParser) shouldSkip(p *PnpmPackage, dev bool) bool {
	if p.Name == "" || p.Version == "" {
		slog.Warn("empty dependency")
		return true
	}

	fi, err := os.Lstat(p.Path)
	if err == nil {
		// skip symlink for cases:
		//	- manually altered node_modules
		// this won't FP since using pnpm list command gives the paths within .pnpm instead of symlinks it creates for node
		mode := fi.Mode()
		if mode&os.ModeSymlink != 0 {
			slog.Warn("symlink dependency")
			return true
		}
	} else {
		// currently warn if this fails, needs to be mocked in all tests once this is implemented
		// ignore for dev-deps
		if !dev {
			slog.Error("failed getting stat", "path", p.Path, "err", err)
		}
	}

	return false
}

func (parser *pnpmDependencyParser) parseDependencyNode(node *PnpmPackage, deps common.DependencyMap, depth int, parent *common.Dependency, branch string, inDevTree bool) error {
	if parent != nil {
		parentDescriptor := fmt.Sprintf("%s@%s", node.Name, node.Version)
		if branch == "" {
			// direct dep
			branch = parentDescriptor
		} else {
			branch = fmt.Sprintf("%s > %s", branch, parentDescriptor) // might be better to construct ourselves instead of using internalId
		}
	}

	for keyName, p := range node.Dependencies {
		if parser.shouldSkip(p, inDevTree) {
			slog.Warn("skipping dep", "name", p.Name, "version", p.Version, "depth", depth, "parent", node)
			continue
		}

		current := pnpmAddDepInstance(deps, p, keyName, parent, inDevTree, branch)
		err := parser.parseDependencyNode(p, deps, depth+1, current, branch, inDevTree)
		if err != nil {
			return err
		}
	}

	// needs to explictly parse dev deps on root node
	if !parser.config.Pnpm.ProdOnlyDeps && parent == nil {
		slog.Info("parsing dev deps on root node")
		for keyName, p := range node.DevDependencies {
			// since this is the root node, all deps here on down are dev
			if parser.shouldSkip(p, true) {
				slog.Warn("skipping dev dep", "name", p.Name, "version", p.Version, "depth", depth, "parent", node)
				continue
			}

			current := pnpmAddDepInstance(deps, p, keyName, parent, true, branch)
			err := parser.parseDependencyNode(p, deps, depth+1, current, branch, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func pnpmAddDepInstance(deps common.DependencyMap, p *PnpmPackage, keyName string, parent *common.Dependency, dev bool, branch string) *common.Dependency {
	common.Trace("adding dep", "name", p.Name, "version", p.Version, "path", p.Path, "key", keyName, "branch", branch)
	newDep := &common.Dependency{
		Name:           p.Name,
		Version:        p.Version,
		PackageManager: shared.NpmManager, // using NPM here as well for the sake of the BE
		DiskPath:       p.Path,
		NameAlias:      keyName,
		Parent:         parent,
		Branch:         branch,
		Dev:            dev,
	}

	key := newDep.Id()
	if _, ok := deps[key]; !ok {
		deps[key] = make([]*common.Dependency, 0, 1)
	}

	if keyName != p.Name {
		slog.Warn("possible alias dependency", "alias", keyName, "name", p.Name, "path", p.Path, "transitive", newDep.IsTransitive())
	}

	deps[key] = append(deps[key], newDep)
	return newDep
}

func skipUntilJsonStarts(output string) string {
	// pnpm's prints to stdout a warning line for failing to replace an env variable
	//		this only happens for the first var due to uncaught exception
	// 		ref: https://github.com/pnpm/pnpm/issues/5914#issuecomment-1378997369
	// will skip input lines until reaches the first '[' character denoting start of valid json output

	if output == "" {
		return output
	}

	// this assumes --json flag outputs an array
	if string(output[0]) == "[" {
		return output
	}

	newlineIdx := strings.Index(output, "\n")
	if newlineIdx == -1 {
		slog.Warn("cant skip line of pnpm output")
		// unknown input 'format'
		return ""
	}

	slog.Info("skipped first line due to bad pnpm output", "skipped", newlineIdx)
	return output[newlineIdx+1:] // skip the \n

}

func (parser *pnpmDependencyParser) Parse(lsOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)
	roots := []PnpmPackage{}

	lsOutput = skipUntilJsonStarts(lsOutput)
	if lsOutput == "" {
		slog.Error("failed skipping bad pnpm output", "output", lsOutput)
		return nil, fmt.Errorf("failed skipping bad output from pnpm")
	}

	err := json.Unmarshal([]byte(lsOutput), &roots)
	if err != nil {
		slog.Error("failed unmarshal ls output", "err", err)
		return nil, err
	}

	if len(roots) == 0 {
		slog.Error("bad json, empty list")
		return nil, fmt.Errorf("bad output from pnpm")
	}

	if len(roots) > 1 {
		slog.Warn("got multiple roots", "count", len(roots))
	}

	root := roots[0]
	if root.Path != projectDir {
		// the first node in the tree is the project's package
		// use it to validate we're in the correct directory
		slog.Error("root is not the same as project dir", "root_path", root.Path, "project_dir", projectDir)
		return nil, utils.CwdWrongProjectDir
	}

	slog.Info("root package", "direct_deps", len(root.Dependencies))
	// currently dependencies can hold dupes, extranious, invalid, etc
	err = parser.parseDependencyNode(&root, deps, 1, nil, "", false)
	if err != nil {
		return nil, err
	}

	return deps, nil
}
