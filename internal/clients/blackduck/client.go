package blackduck

import (
	"cli/internal/api"
	"cli/internal/common"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	headerVersion4 = "application/vnd.blackducksoftware.user-4+json"
	headerVersion5 = "application/vnd.blackducksoftware.project-detail-5+json"
	headerVersion6 = "application/vnd.blackducksoftware.bill-of-materials-6+json"
)

const limit = 100

type BlackDuckClient struct {
	Client http.Client
	Url    string
	Token  string
}

func NewClient(url, token string) *BlackDuckClient {
	return &BlackDuckClient{
		Client: http.Client{},
		Url:    url,
		Token:  token,
	}
}

func (c *BlackDuckClient) getHeaders(api_version string, overrideContentType bool) []api.StringPair {
	baseHeaders := []api.StringPair{
		{Name: "Authorization", Value: fmt.Sprintf("Bearer %s", c.Token)},
		{Name: "Accept", Value: api_version},
	}

	if overrideContentType {
		return append(baseHeaders, api.StringPair{
			Name: "Content-Type", Value: api_version,
		})
	}

	return append(baseHeaders, api.StringPair{
		Name: "Content-Type", Value: "application/json",
	})
}

func (c *BlackDuckClient) executeGet(url string, params []api.StringPair, headers []api.StringPair) ([]byte, error) {
	res, statusCode, err := api.BaseSendRequest[any](
		c.Client,
		"GET",
		url,
		nil,
		headers,
		params,
	)
	if err != nil || statusCode != 200 {
		slog.Debug("failed sending request", "err", err, "status", statusCode, "url", url)
		return nil, err
	}

	slog.Debug("received response", "data", string(res))
	return res, nil
}

func (c *BlackDuckClient) getProjects(params []api.StringPair) (*bdProjects, error) {
	url := fmt.Sprintf("%s/%s", c.Url, "api/projects")
	projects, err := c.executeGet(url, params, c.getHeaders(headerVersion6, true))
	if err != nil {
		slog.Debug("failed getting projects", "err", err, "url", url)
		return nil, err
	}

	var p bdProjects
	if err := json.Unmarshal(projects, &p); err != nil {
		slog.Error("failed unmarshalling projects", "err", err)
		return nil, err
	}

	return &p, nil
}

func (c *BlackDuckClient) getProjectByName(name string) (*bdProject, error) {
	amount := limit
	offset := 0

	for offset*limit < amount {
		params := []api.StringPair{
			{Name: "limit", Value: fmt.Sprintf("%d", limit)},
			{Name: "offset", Value: fmt.Sprintf("%d", offset)},
			{Name: "q", Value: fmt.Sprintf("name:%s", name)},
		}

		projects, err := c.getProjects(params)
		if err != nil {
			return nil, common.NewPrintableError("failed to update BlackDuck. Could not fetch project %s", name)
		}

		for _, project := range projects.Items {
			if project.Name == name {
				slog.Debug("found project", "project", project.Name)
				return &project, nil
			}
		}
		amount = projects.TotalCount
		offset++
	}

	return nil, common.NewPrintableError("failed to update BlackDuck. Project %s was not found", name)
}

func (c *BlackDuckClient) getProjectVersions(project *bdProject, _limit, offset int) (*bdVersions, error) {
	url := project.Meta.Href

	params := []api.StringPair{
		{Name: "limit", Value: fmt.Sprintf("%d", _limit)},
		{Name: "offset", Value: fmt.Sprintf("%d", offset)},
	}

	versions, err := c.executeGet(
		fmt.Sprintf("%s/%s", url, "versions"),
		params,
		c.getHeaders(headerVersion5, true),
	)

	if err != nil {
		slog.Debug("failed getting project versions", "err", err, "url", url)
		return nil, err
	}

	var v bdVersions
	if err := json.Unmarshal(versions, &v); err != nil {
		slog.Error("failed unmarshalling project versions", "err", err)
		return nil, err
	}

	return &v, nil
}

func (c *BlackDuckClient) getLink(version bdVersion, linkName string) string {
	links := version.Meta.Links
	for _, link := range links {
		// if link has a rel attribute that matches the linkName, return the href
		if link.Rel == linkName {
			return link.Href
		}
	}

	return ""
}

func (c *BlackDuckClient) getVulnerableComponents(url string, _limit, offset int) (*bdVulnerableBOMComponents, error) {
	params := []api.StringPair{
		{Name: "limit", Value: fmt.Sprintf("%d", _limit)},
		{Name: "offset", Value: fmt.Sprintf("%d", offset)},
	}

	vulnerableComponents, err := c.executeGet(url, params, c.getHeaders(headerVersion6, true))
	if err != nil {
		slog.Debug("failed getting vulnerable components", "err", err, "url", url)
		return nil, err
	}

	var v bdVulnerableBOMComponents
	if err := json.Unmarshal(vulnerableComponents, &v); err != nil {
		slog.Error("failed unmarshalling vulnerable components", "err", err)
		return nil, err
	}

	return &v, nil
}

func (c *BlackDuckClient) executePut(url string, body []byte, params []api.StringPair, headers []api.StringPair) ([]byte, error) {
	res, statusCode, err := api.BaseSendRequest(
		c.Client,
		"PUT",
		url,
		&body,
		headers,
		params,
	)
	if err != nil || statusCode != 202 {
		slog.Debug("failed sending request", "err", err, "status", statusCode, "url", url, "body", string(body))
		return nil, err
	}

	return res, nil
}

func (c *BlackDuckClient) updateVuln(url string, update bdUpdateBOMComponentVulnerabilityRemediation) error {
	body, err := json.Marshal(update)
	if err != nil {
		slog.Error("failed marshalling update", "err", err)
		return err
	}

	res, err := c.executePut(url, body, nil, c.getHeaders(headerVersion6, false))
	slog.Debug("UpdateVulnerability response", "data", string(res))
	return err
}

func (c *BlackDuckClient) getAllVulnsInVersion(version bdVersion, vulnChan chan bdVulnerableBOMComponent) {
	amount := limit
	offset := 0
	link := c.getLink(version, "vulnerable-components")
	if link == "" {
		slog.Warn("failed getting link", "offset", offset)
		return
	}

	for offset < amount {
		vulns, err := c.getVulnerableComponents(link, limit, offset)
		if err != nil {
			slog.Warn("failed getting vulnerable components", "err", err, "link", link, "offset", offset)
			return
		}

		amount = vulns.TotalCount
		if amount == 0 {
			slog.Debug("no vulnerabilities found for version", "version", version.VersionName)
			return
		}

		for _, vulnerableComponent := range vulns.Items {
			vulnChan <- vulnerableComponent
		}

		offset += limit
	}
}

func (c *BlackDuckClient) getAllVulnsInProject(project *bdProject, vulnsChannel chan bdVulnerableBOMComponent) error {
	amount := limit
	offset := 0

	for offset < amount {
		versions, err := c.getProjectVersions(project, limit, offset)
		if err != nil {
			slog.Debug("failed to get project versions", "err", err)
			return common.NewPrintableError("failed to update BlackDuck. Could not fetch list of vulnerable components for project version")
		}

		amount = versions.TotalCount
		if amount == 0 {
			slog.Info("no versions found for project", "project", project.Name)
			return nil
		}

		for _, v := range versions.Items {
			c.getAllVulnsInVersion(v, vulnsChannel)
		}

		offset += limit
	}

	return nil
}
