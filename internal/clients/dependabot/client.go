package dependabot

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	contentType    = "application/vnd.github+json"
	ghasApiVersion = "2022-11-28"
)
const per_page = 100

type DependabotClient struct {
	Client http.Client
	Url    string
	Token  string
	Owner  string
	Repo   string
}

func NewClient(dependabotConfig config.DependabotConfig) *DependabotClient {

	if dependabotConfig.Url == "" {
		slog.Debug("dependabot URL not configured. Using default " + defaultGitHubUrl)
		dependabotConfig.Url = defaultGitHubUrl
	}

	return &DependabotClient{
		Client: http.Client{},
		Url:    dependabotConfig.Url,
		Token:  dependabotConfig.Token.Value(),
		Owner:  dependabotConfig.Owner,
		Repo:   dependabotConfig.Repo,
	}
}

func (c *DependabotClient) getHeaders() ([]api.StringPair, error) {
	baseHeaders := []api.StringPair{
		{Name: "Authorization", Value: fmt.Sprintf("Bearer %s", c.Token)},
		{Name: "Accept", Value: contentType},
		{Name: "X-GitHub-Api-Version", Value: ghasApiVersion},
	}

	return baseHeaders, nil
}

func (c *DependabotClient) updateVuln(url string, update *dependabotUpdateComponentVulnerabilityRemediation) error {
	headers, err := c.getHeaders()
	if err != nil {
		slog.Error("failed getting headers", "err", err)
		return err
	}

	res, statusCode, err := api.SendHttpRequest(
		c.Client,
		"PATCH",
		url,
		update,
		headers,
		nil,
	)

	if err != nil {
		slog.Debug("failed sending request", "err", err, "status", statusCode, "url", url, "body", update)
		return err
	}

	if statusCode != 200 {
		slog.Debug("failed updating vulnerability", "status", statusCode, "url", url, "body", update)
		return fmt.Errorf("failed sending request PUT to %s, status: %d", url, statusCode)
	}

	slog.Debug("UpdateVulnerability response", "data", string(res))
	return err
}

func (c *DependabotClient) executeGet(url string, params []api.StringPair, headers []api.StringPair) ([]byte, error) {
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

func (c *DependabotClient) getProjectAlerts(_per_page, page int) (*dependabotVulnerableComponents, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/dependabot/alerts", c.Url, c.Owner, c.Repo)

	params := []api.StringPair{
		{Name: "per_page", Value: fmt.Sprintf("%d", _per_page)},
		{Name: "page", Value: fmt.Sprintf("%d", page)},
		{Name: "state", Value: "open,dismissed"},
	}

	headers, err := c.getHeaders()
	if err != nil {
		slog.Error("failed getting headers", "err", err)
		return nil, err
	}

	alerts, err := c.executeGet(
		url,
		params,
		headers,
	)
	slog.Debug("Dependabot alerts are" + string(alerts))

	if err != nil {
		slog.Debug("failed getting project versions", "err", err, "url", url)
		return nil, err
	}

	var d dependabotVulnerableComponents
	if err := json.Unmarshal(alerts, &d); err != nil {
		slog.Error("failed unmarshalling project versions", "err", err)
		return nil, err
	}

	return &d, nil
}

func (c *DependabotClient) getAllVulnsInProject(vulnsChannel chan dependabotVulnerableComponent) error {
	page := 1
	for {
		alerts, err := c.getProjectAlerts(per_page, page)
		if err != nil {
			slog.Debug("failed to get project alerts", "err", err)
			return common.NewPrintableError("failed to update Dependabot. Could not fetch list of alerts for project")
		}
		if len(*alerts) == 0 {
			slog.Debug("No more alerts. Exit the loop")
			return nil
		}
		for _, vulnerableComponent := range *alerts {
			vulnsChannel <- vulnerableComponent
		}
		page++
	}
}
