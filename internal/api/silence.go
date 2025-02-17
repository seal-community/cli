package api

import (
	"cli/internal/common"
)

func GetSilencedMap(silenced []SilenceRule, allDependencies common.DependencyMap, manager string) map[string][]string {
	silencedPackages := make(map[string][]string)
	for _, rule := range silenced {
		ruleDependencyId := common.DependencyId(manager, rule.Library, rule.Version)
		silencedPaths := []string{}
		for _, dep := range allDependencies[ruleDependencyId] {
			silencedPaths = append(silencedPaths, dep.DiskPath)
		}
		silencedPackages[ruleDependencyId] = silencedPaths
	}

	return silencedPackages
}
