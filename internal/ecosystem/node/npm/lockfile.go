package npm

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/iancoleman/orderedmap"
)



var UnsupportedLockfileVersion = common.NewPrintableError("unsupported package-lock.json version")

type lockfileVersion int

// ref: https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json#file-format
const (
	lockOld lockfileVersion = iota // before npm5 - does not exist, has no value, using 0 here - seems like it's `npm-shrinkwrap.json`
	lockV1  lockfileVersion = iota // npm5 | npm6
	lockV2  lockfileVersion = iota // npm7|npm8 - compatible with v1
	lockV3  lockfileVersion = iota // npm9 and later - compatible with npm7

	lockSupportedVersion lockfileVersion = iota // only below this valid values
	lockBadVersion       lockfileVersion = -1
)

func formatUpdatedPackageName(originalName string) string {
	idx := strings.LastIndex(originalName, "/")
	if idx == -1 {
		return fmt.Sprintf("seal-%s", originalName)
	}
	
	// either nested path in node modules, or namespaces @owner/pkg
	return fmt.Sprintf("%sseal-%s", originalName[:idx+1], originalName[idx+1:])
}

func getLockfileVersion(lock *orderedmap.OrderedMap) lockfileVersion {
	versionValue, exists := lock.Get("lockfileVersion")
	if !exists {
		slog.Debug("no lock file version in json")
		return lockOld
	}

	number, ok := versionValue.(float64) // json values
	if !ok {
		slog.Warn("failed parsing version value", "value", versionValue)
		return lockBadVersion
	}

	versionInt := int(number)
	if versionInt == 0 {
		// we're using lack of version as enum value for the old lockfile version, so prevent it from being accepted as raw value
		slog.Warn("unsupported 0 version value")
		return lockBadVersion
	}

	version := lockfileVersion(versionInt)
	if version >= lockSupportedVersion {
		slog.Warn("unknown value meaning for lock file version", "value", version)
		return lockBadVersion
	}

	return version
}

// used to flip the data from FixMap to:
//
//	diskpath -> package
//
// this way we can easily find entries in the lock file that needs updating
func extractFixedLocations(fixes shared.FixMap) map[string]*api.PackageVersion {
	newmap := make(map[string]*api.PackageVersion)
	for _, entry := range fixes {
		for path, fixed := range entry.Paths {
			if !fixed {
				continue // ignoreing non fixed paths
			}

			// should not overwrite, shouldn't have dups, especially cross packages
			newmap[path] = entry.Package
		}
	}

	return newmap
}

func UpdateLockfile(lock *orderedmap.OrderedMap, fixes shared.FixMap, projectDir string) error {
	version := getLockfileVersion(lock)
	fixmap := extractFixedLocations(fixes)

	switch version {
	case lockV3:
		// v3 is backwards compatible with v2 so use v2 logic
		slog.Debug("updating lock file v3")
		return updateV2(lock, fixmap, projectDir)

	case lockV2:
		slog.Debug("updating lock file v2")
		if err := updateV2(lock, fixmap, projectDir); err != nil {
			return err
		}

		fallthrough // v2 kept backwards compatiblity with v1 so continue to its logic

	case lockV1:
		slog.Debug("updating lock file v1/v2")
		return updateV1(lock, fixmap, projectDir)

	default:
		// not supporting the old format lockOld for now
		return UnsupportedLockfileVersion
	}
}

// update v1 style package-lock file - changes the dependencies under `dependencies` keys recrusively
// this version does not keep the disk path for the dependency as the key name for it in json, so we need to build it each time
// to find it in the fixed paths map
func updateV1(node *orderedmap.OrderedMap, fixes map[string]*api.PackageVersion, root string) error {
	depObj, exists := node.Get("dependencies")
	if !exists {
		return nil
	}

	deps, ok := depObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for dependencies key", "dep-path", root)
		return nil
	}

	// pefform bfs on leafs to not modify during iteration
	keysToUpdate := orderedmap.New()

	for _, packageName := range deps.Keys() {
		obj, _ := deps.Get(packageName)
		diskPath := filepath.Join(root, "node_modules", packageName) // needs to build full path within the root node_modules to get the key

		child, ok := obj.(orderedmap.OrderedMap)
		if !ok {
			slog.Warn("bad object type for child key", "name", packageName, "root", root)
			return nil
		}

		if pv, exists := fixes[diskPath]; exists {
			keysToUpdate.Set(packageName, pv)
		}

		if err := updateV1(&child, fixes, diskPath); err != nil {
			return err
		}
	}

	for _, packageKey := range keysToUpdate.Keys() {
		pv, _ := keysToUpdate.Get(packageKey)
		newName := formatUpdatedPackageName(packageKey)
		newVersion := pv.(*api.PackageVersion).RecommendedLibraryVersionString

		child, _ := deps.Get(packageKey)

		if childNode, ok := child.(orderedmap.OrderedMap); ok {
			if _, exists := childNode.Get("version"); exists {
				slog.Debug("updating dependency", "oldName", packageKey, "newName", newName, "newVersion", newVersion)
				// set new version value
				childNode.Set("version", newVersion)

				// replace old object with new, causes the structure to change slightly since it is being appended to end
				deps.Delete(packageKey)
				deps.Set(newName, child)
			}
		}
	}

	// have to re-set the deps since its all by-value
	node.Set("dependencies", deps)

	return nil
}

// update v2 style package-lock file - changes the dependencies under `packages` keys recrusively
// this format keeps a relative path to the package dir as its key, we can use it with the project root to find it in fix map
func updateV2(node *orderedmap.OrderedMap, fixes map[string]*api.PackageVersion, root string) error {

	depObj, exists := node.Get("packages")
	if !exists {
		return nil
	}

	deps, ok := depObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for packages key")
		return nil
	}

	// pefform bfs on leafs to not modify during iteration
	keysToUpdate := orderedmap.New() // using sorted so that iteration over this later, results in the same order (especially for tests)
	for _, packagePath := range deps.Keys() {
		obj, _ := deps.Get(packagePath)
		diskPath := filepath.Join(root, packagePath)

		child, ok := obj.(orderedmap.OrderedMap)
		if !ok {
			slog.Warn("bad object type for child key", "key", packagePath)
			return nil
		}

		if pv, exists := fixes[diskPath]; exists {
			keysToUpdate.Set(packagePath, pv)
		}

		if err := updateV2(&child, fixes, diskPath); err != nil {
			return err
		}
	}

	for _, packageKey := range keysToUpdate.Keys() {
		pv, _ := keysToUpdate.Get(packageKey)
		newName := formatUpdatedPackageName(packageKey)
		newVersion := pv.(*api.PackageVersion).RecommendedLibraryVersionString

		child, _ := deps.Get(packageKey)

		if childNode, ok := child.(orderedmap.OrderedMap); ok {
			if _, exists := childNode.Get("version"); exists {
				slog.Debug("updating dependency", "oldName", packageKey, "newName", newName, "newVersion", newVersion)

				// override version
				childNode.Set("version", newVersion)

				// replace old object with new, causes the structure to change slightly since it is being appended to end
				deps.Delete(packageKey)
				deps.Set(newName, child) // will cause it to move to bottom
			}
		}
	}

	// have to re-set the deps since its all by-value
	node.Set("packages", deps)

	return nil
}
