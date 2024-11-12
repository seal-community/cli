//go:build !mockserver
// +build !mockserver

package phase

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"net/url"
)

// builds url as:
//
//	https://{host}/artifactory/{repo-key}
func buildRepoArtifactUrl(scheme, host, repoKey string) string {
	if scheme == "" || host == "" || repoKey == "" {
		slog.Error("missing information for building repo url", "scheme", scheme, "host", host, "repoKey", repoKey)
		return ""
	}

	uri, err := url.JoinPath("artifactory", repoKey)
	if err != nil {
		slog.Error("failed building base url", "err", err, "host", host, "repoKey", repoKey)
		return ""
	}

	artifactUrl := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   uri,
	}

	return artifactUrl.String()
}

func getJfrogArtifactRepo(manager shared.PackageManager, conf *config.Config) string {
	ecosystem := manager.GetEcosystem()

	switch ecosystem {
	case mappings.JavaEcosystem:
		return conf.JFrog.MavenRepository
	default:
		return ""
	}
}

func getJfrogCliServerUrl(config *config.Config) (string, error) {
	jfrogConf := config.JFrog
	baseUrl := buildRepoArtifactUrl(jfrogConf.Scheme, jfrogConf.Host, jfrogConf.CliRepository)
	if baseUrl == "" {
		slog.Error("could not initialize cli jfrog server", "scheme", jfrogConf.Scheme, "host", jfrogConf.Host, "repoKey", jfrogConf.CliRepository)
		return "", common.NewPrintableError("misconfigured CLI JFrog server")
	}

	return baseUrl, nil
}

func getJfrogArtifactServerUrl(config *config.Config, manager shared.PackageManager) (string, error) {
	jfrogConf := config.JFrog

	artifactRepo := getJfrogArtifactRepo(manager, config)
	if artifactRepo == "" {
		slog.Error("failed init artifact repo with jfrog", "manager", manager.Name())
		return "", common.NewPrintableError("unsupported ecosystem for JFrog - %s", manager.GetEcosystem())
	}

	baseUrl := buildRepoArtifactUrl(jfrogConf.Scheme, jfrogConf.Host, jfrogConf.MavenRepository)
	if baseUrl == "" {
		slog.Error("could not initialize artifact jfrog server", "scheme", jfrogConf.Scheme, "host", jfrogConf.Host, "repoKey", jfrogConf.CliRepository)
		return "", common.NewPrintableError("bad JFrog server configuration")
	}

	return baseUrl, nil
}
