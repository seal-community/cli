package nuget

import (
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"strings"

	"github.com/iancoleman/orderedmap"
)

func UpdateProjectAssetsfile(assets *orderedmap.OrderedMap, fixes shared.FixMap) error {
	for _, fix := range fixes {
		name := fix.Package.Library.Name
		fromVersion := fix.Package.Version
		toVersion := fix.Package.RecommendedLibraryVersionString

		fromKey := fmt.Sprintf("%s/%s", name, fromVersion)
		toKey := fmt.Sprintf("%s/%s", name, toVersion)

		fixTargets(assets, fromKey, toKey)
		fixLibraries(assets, fromKey, toKey, fromVersion, toVersion)
		fixProjectFileDependencyGroups(assets, fromVersion, toVersion)
		fixProject(assets, name, fromVersion, toVersion)
	}

	return nil
}

func fixTargets(assets *orderedmap.OrderedMap, fromKey, toKey string) {
	targetsObj, exists := assets.Get("targets")

	if !exists {
		slog.Error("failed getting targets from project.assets.json")
		return
	}

	targets, ok := targetsObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for targets key")
		return
	}

	for key, frameworkObj := range targets.Values() {
		framework, ok := frameworkObj.(orderedmap.OrderedMap)
		if !ok {
			slog.Warn("bad object type for framework key")
			return
		}

		value, exists := framework.Get(fromKey)
		if exists {
			framework.Set(toKey, value)
			framework.Delete(fromKey)
		}

		targets.Set(key, framework)
	}
}

func fixLibraries(assets *orderedmap.OrderedMap, fromKey, toKey, fromVersion, toVersion string) {
	librariesObj, exists := assets.Get("libraries")

	if !exists {
		slog.Error("failed getting libraries from project.assets.json")
		return
	}

	libraries, ok := librariesObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for libraries key")
		return
	}

	libraryObj, exists := libraries.Get(fromKey)
	if !exists {
		slog.Warn("library not found in project.assets.json", "library", fromKey)
		return
	}

	library := libraryObj.(orderedmap.OrderedMap)
	library.Set("path", strings.ToLower(toKey))

	files, exists := library.Get("files")
	if exists {
		filesArr := files.([]interface{})
		for i, file := range filesArr {
			if strings.HasSuffix(file.(string), ".nupkg.sha512") {
				filesArr[i] = strings.ReplaceAll(file.(string), fromVersion, toVersion)
			}
		}

		library.Set("files", filesArr)
	}

	libraries.Set(toKey, library)
	libraries.Delete(fromKey)

	assets.Set("libraries", libraries)
}

func fixProjectFileDependencyGroups(assets *orderedmap.OrderedMap, fromVersion, toVersion string) {
	projectFileDependencyGroupsObj, exists := assets.Get("projectFileDependencyGroups")

	if !exists {
		slog.Error("failed getting projectFileDependencyGroups from project.assets.json")
		return
	}

	projectFileDependencyGroups, ok := projectFileDependencyGroupsObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for projectFileDependencyGroups key")
		return
	}

	for key, group := range projectFileDependencyGroups.Values() {
		newDependencies := []string{}
		for _, dep := range group.([]interface{}) {
			if strings.Contains(dep.(string), fromVersion) {
				newDependencies = append(newDependencies, strings.ReplaceAll(dep.(string), fromVersion, toVersion))
			} else {
				newDependencies = append(newDependencies, dep.(string))
			}
		}

		projectFileDependencyGroups.Set(key, newDependencies)
	}
}

func fixProject(assets *orderedmap.OrderedMap, name, fromVersion, toVersion string) {
	projectObj, exists := assets.Get("project")

	if !exists {
		slog.Error("failed getting project from project.assets.json")
		return
	}

	project, ok := projectObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for project key")
		return
	}

	frameworksObj, exists := project.Get("frameworks")

	if !exists {
		slog.Error("failed getting dependencies from project.assets.json")
		return
	}

	frameworks, ok := frameworksObj.(orderedmap.OrderedMap)
	if !ok {
		slog.Warn("bad object type for dependencies key")
		return
	}

	for key, frameworkObj := range frameworks.Values() {
		framework, ok := frameworkObj.(orderedmap.OrderedMap)
		if !ok {
			slog.Warn("bad object type for framework key")
			return
		}

		dependenciesObj, exists := framework.Get("dependencies")
		if !exists {
			continue
		}

		dependencies, ok := dependenciesObj.(orderedmap.OrderedMap)
		if !ok {
			continue
		}

		for depKey, depValue := range dependencies.Values() {
			if depKey == name {
				depMap, ok := depValue.(orderedmap.OrderedMap)
				if !ok {
					continue
				}
				
				versionObj, exists := depMap.Get("version")
				if !exists {
					continue
				}

				version, ok := versionObj.(string)
				if !ok {
					continue
				}

				depMap.Set("version", strings.ReplaceAll(version, fromVersion, toVersion))
				dependencies.Set(name, depMap)
			}
		}

		framework.Set("dependencies", dependencies)
		frameworks.Set(key, framework)
	}

	project.Set("frameworks", frameworks)
}
