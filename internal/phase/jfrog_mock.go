//go:build mockserver
// +build mockserver

package phase

// these mocks will allow testing without actualy routing through JFrog

import (
	"cli/internal/api"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"net/url"
)

var cliJfrogRoute = "jfrog"

// will return url to the mocked cli server url, with the new jfrog sub-route
// immitates what would happen if we used jfrog
func getJfrogCliServerUrl(config *config.Config) (string, error) {
	return url.JoinPath(api.BaseURL, "authenticated", cliJfrogRoute)
}

// will use the normal mocked urls for the test servers
func getJfrogArtifactServerUrl(config *config.Config, manager shared.PackageManager) (string, error) {
	ecosystem := manager.GetEcosystem()
	u := ""

	switch ecosystem {
	case mappings.JavaEcosystem:
		u = api.MavenServer
	default:
		return "", fmt.Errorf("unsupported ecosystem %s", ecosystem)
	}

	return u, nil
}
