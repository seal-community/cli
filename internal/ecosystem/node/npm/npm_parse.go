package npm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/node/utils"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/exp/slices"
)

type NpmPackage struct {
	Version      string                 `json:"version"`
	Name         string                 `json:"name"`
	Path         string                 `json:"path"`
	Dependencies map[string]*NpmPackage `json:"dependencies"`
	Extraenous   bool                   `json:"extraneous"` // npm specifc,
	IntenralId   string                 `json:"_id"`        // npm specifc, exists in npm 6.14.18, 10.2.0; sanity
	Workspaces   []string               `json:"workspaces"` // npm specifc, exists in npm 6.14.18, 10.2.0; sanity
}

type dependencyParser struct {
	config *config.Config // in the future we might want to only pass the npm specific config object
}

func (parser *dependencyParser) isWorkspace(root *NpmPackage, p *NpmPackage) bool {
	// Currently we use a simple heuristic to determine if we're in a workspace
	// if we are inspecting a symlink dependency, we extract the relative path compared to the root package
	// if the relative path is in the workspaces array, we're in a workspace
	// The complete way to do this would be to go over each workspace, resolve the module name, and compare it to the current module
	rootSymLinkDest, err := filepath.EvalSymlinks(root.Path)
	if err != nil {
		slog.Warn("failed resolving symlink", "path", root.Path, "err", err)
		return false
	}
	
	packageSymLinkDest, err := filepath.EvalSymlinks(p.Path)
	if err != nil {
		slog.Warn("failed resolving symlink", "path", p.Path, "err", err)
		return false
	}
	relPath, err := filepath.Rel(rootSymLinkDest, packageSymLinkDest)
	if err != nil {
		slog.Warn("failed getting relative path", "path", p.Path, "err", err)
		return false
	}
	if !slices.Contains(root.Workspaces, relPath) {
		slog.Debug("not in workspace", "path", p.Path, "rel_path", relPath)
		return false
	}
	return true
}

func (parser *dependencyParser) shouldSkip(root *NpmPackage, p *NpmPackage) bool {
	if p.Name == "" || p.Version == "" {
		slog.Debug("empty dependency")
		return true
	}

	if p.Extraenous {
		// this will also 'catch' dependencies that are installed using `npm link {package name}`
		slog.Debug("extraneous dependency", "name", p.Name, "version", p.Version, "path", p.Path)
		if parser.config.Npm.IgnoreExtraneous {
			slog.Debug("skipping extraneous dependency")
			return true
		}
	}

	fi, err := os.Lstat(p.Path)
	if err == nil {
		// skip symlink for cases:
		//  - cli configured to parse extraneous deps
		//	- manually altered node_modules
		mode := fi.Mode()
		if mode&os.ModeSymlink != 0 {
			slog.Debug("symlink dependency")
			if !parser.isWorkspace(root, p) {
				slog.Debug("skipping symlink dependency", "name", p.Name, "version", p.Version, "path", p.Path)
				return true
			}
		}
	} else {
		// currently warn if this fails, needs to be mocked in all tests once this is implemented
		slog.Warn("failed getting stat", "path", p.Path, "err", err)
	}

	return false
}

func (parser *dependencyParser) parseDependencyNode(root *NpmPackage, node *NpmPackage, deps common.DependencyMap, depth int, parent *common.Dependency, branch string) error {
	if parent != nil {
		if branch == "" {
			// direct dep
			branch = node.IntenralId
		} else {
			branch = fmt.Sprintf("%s > %s", branch, node.IntenralId) // might be better to construct ourselves instead of using internalId
		}
	}

	for keyName, p := range node.Dependencies {
		if parser.shouldSkip(root, p) {
			slog.Warn("skipping dep", "name", p.Name, "version", p.Version, "depth", depth, "parentId", node.IntenralId)
			continue
		}

		current := addDepInstance(deps, p, keyName, parent, branch)
		err := parser.parseDependencyNode(root, p, deps, depth+1, current, branch)
		if err != nil {
			return err
		}
	}

	return nil
}

func addDepInstance(deps common.DependencyMap, p *NpmPackage, keyName string, parent *common.Dependency, branch string) *common.Dependency {
	common.Trace("adding dep", "name", p.Name, "version", p.Version, "path", p.Path, "key", keyName, "branch", branch)
	newDep := &common.Dependency{
		Name:           p.Name,
		Version:        p.Version,
		PackageManager: mappings.NpmManager,
		DiskPath:       p.Path,
		NameAlias:      keyName,
		Parent:         parent,
		Extraneous:     p.Extraenous,
		Branch:         branch,
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

func (parser *dependencyParser) Parse(lsOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)

	root := NpmPackage{}
	err := json.Unmarshal([]byte(lsOutput), &root)
	if err != nil {
		slog.Error("failed unmarshal ls output", "err", err)
		return nil, err
	}

	if root.Path != projectDir {
		// the first node in the tree is the project's package
		// use it to validate we're in the correct directory
		slog.Error("root is not the same as project dir", "root_path", root.Path, "project_dir", projectDir)
		return nil, utils.CwdWrongProjectDir
	}

	slog.Info("root package", "direct_deps", len(root.Dependencies))
	// currently dependencies can hold dupes, extranious, invalid, etc
	err = parser.parseDependencyNode(&root, &root, deps, 1, nil, "")
	if err != nil {
		return nil, err
	}

	return deps, nil
}
