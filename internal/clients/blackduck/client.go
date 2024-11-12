package blackduck

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	contentTypeUserV4           = "application/vnd.blackducksoftware.user-4+json"
	contentTypeProjectDetailsV5 = "application/vnd.blackducksoftware.project-detail-5+json"
	contentTypeProjectDetailsV6 = "application/vnd.blackducksoftware.project-detail-6+json"
	contentTypeSBOMV6           = "application/vnd.blackducksoftware.bill-of-materials-6+json"
)

const limit = 100
const expireMinutesThreshold = 5

type BlackDuckClient struct {
	Client          http.Client
	Url             string
	Token           string
	VersionToFilter string
	ValidUntil      time.Time
	BearerToken     string
}

func NewClient(bdConfig config.BlackDuckConfig) *BlackDuckClient {
	return &BlackDuckClient{
		Client:          http.Client{},
		Url:             bdConfig.Url,
		Token:           bdConfig.Token.Value(),
		VersionToFilter: bdConfig.VersionName,
	}
}

func (c *BlackDuckClient) authenticate() error {
	url := fmt.Sprintf("%s/%s", c.Url, "api/tokens/authenticate")
	headers := []api.StringPair{
		{Name: "Authorization", Value: fmt.Sprintf("token %s", c.Token)},
		{Name: "Accept", Value: contentTypeUserV4},
	}
	res, statusCode, err := api.SendHttpRequest[any](
		c.Client,
		"POST",
		url,
		nil,
		headers,
		nil,
	)
	if err != nil {
		slog.Debug("failed to authenticate", "err", err, "status", statusCode, "url", url)
		return err
	}

	if statusCode != 200 {
		slog.Debug("failed to authenticate", "status", statusCode, "url", url)
		return fmt.Errorf("failed sending request POST to %s, status: %d", url, statusCode)
	}

	var t bdAPITokenResponse
	if err := json.Unmarshal(res, &t); err != nil {
		slog.Error("failed unmarshalling token", "err", err)
		return err
	}

	c.BearerToken = t.BearerToken
	c.ValidUntil = time.Now().Add(time.Duration(t.ExpiresInMilliseconds) * time.Millisecond)

	return nil
}

func (c *BlackDuckClient) getBearerAuth() (string, error) {
	// Valid until 5 more minutes:
	if !c.ValidUntil.Before(time.Now().Add(expireMinutesThreshold * time.Minute)) {
		slog.Debug("Bearer token is still valid, reusing")
		return c.BearerToken, nil
	}

	slog.Debug("Connecting to BlackDuck")
	if err := c.authenticate(); err != nil {
		slog.Error("failed authenticating with BlackDuck", "err", err)
		return "", common.NewPrintableError("failed authenticating to BlackDuck")
	}

	slog.Debug("Bearer token updated", "token", c.BearerToken)
	return c.BearerToken, nil
}

func (c *BlackDuckClient) getHeaders(content_type string) ([]api.StringPair, error) {
	bearerAuth, err := c.getBearerAuth()
	if err != nil {
		return nil, err
	}

	baseHeaders := []api.StringPair{
		{Name: "Authorization", Value: fmt.Sprintf("Bearer %s", bearerAuth)},
		{Name: "Accept", Value: content_type},
		{Name: "Content-Type", Value: content_type},
	}

	return baseHeaders, nil
}

func (c *BlackDuckClient) executeGet(url string, params []api.StringPair, headers []api.StringPair) ([]byte, error) {
	res, statusCode, err := api.SendHttpRequest[any](
		c.Client,
		"GET",
		url,
		nil,
		headers,
		params,
	)

	if err != nil {
		slog.Debug("failed sending request", "err", err, "status", statusCode, "url", url)
		return nil, err
	}

	if statusCode != 200 {
		slog.Debug("failed getting response", "status", statusCode, "url", url)
		return nil, fmt.Errorf("failed sending request GET to %s, status: %d", url, statusCode)
	}

	slog.Debug("received response", "data", string(res))
	return res, nil
}

func (c *BlackDuckClient) getProjects(params []api.StringPair) (*bdProjects, error) {
	url := fmt.Sprintf("%s/%s", c.Url, "api/projects")
	headers, err := c.getHeaders(contentTypeProjectDetailsV6)
	if err != nil {
		slog.Error("failed getting headers", "err", err)
		return nil, err
	}

	projects, err := c.executeGet(url, params, headers)
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
	url := fmt.Sprintf("%s/%s", project.Meta.Href, "versions")

	params := []api.StringPair{
		{Name: "limit", Value: fmt.Sprintf("%d", _limit)},
		{Name: "offset", Value: fmt.Sprintf("%d", offset)},
	}

	headers, err := c.getHeaders(contentTypeProjectDetailsV5)
	if err != nil {
		slog.Error("failed getting headers", "err", err)
		return nil, err
	}

	versions, err := c.executeGet(
		url,
		params,
		headers,
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

	headers, err := c.getHeaders(contentTypeSBOMV6)
	if err != nil {
		slog.Error("failed getting headers", "err", err)
		return nil, err
	}

	vulnerableComponents, err := c.executeGet(url, params, headers)
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

func (c *BlackDuckClient) updateVuln(url string, update *bdUpdateBOMComponentVulnerabilityRemediation) error {
	headers, err := c.getHeaders(contentTypeSBOMV6)
	if err != nil {
		slog.Error("failed getting headers", "err", err)
		return err
	}

	res, statusCode, err := api.SendHttpRequest(
		c.Client,
		"PUT",
		url,
		update,
		headers,
		nil,
	)

	if err != nil {
		slog.Debug("failed sending request", "err", err, "status", statusCode, "url", url, "body", update)
		return err
	}

	if statusCode != 202 {
		slog.Debug("failed updating vulnerability", "status", statusCode, "url", url, "body", update)
		return fmt.Errorf("failed sending request PUT to %s, status: %d", url, statusCode)
	}

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
			// VersionToFilter is the project's branch version
			if v.VersionName != c.VersionToFilter {
				slog.Debug("skipping version", "version", v.VersionName)
				continue
			}
			c.getAllVulnsInVersion(v, vulnsChannel)
		}

		offset += limit
	}

	return nil
}
