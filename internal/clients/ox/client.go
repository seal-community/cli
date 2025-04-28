package ox

import (
	"cli/internal/api"
	"cli/internal/config"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	contentType = "application/json"
)

type OxClient struct {
	Client                       http.Client
	Url                          string
	Token                        string
	Application                  string
	ExcludeWhenHighCriticalFixed bool
}

func NewClient(oxConfig config.OxConfig) *OxClient {
	return &OxClient{
		Client:                       http.Client{},
		Url:                          oxConfig.Url,
		Token:                        oxConfig.Token.Value(),
		Application:                  oxConfig.Application,
		ExcludeWhenHighCriticalFixed: oxConfig.ExcludeWhenHighCriticalFixed,
	}
}

func (c *OxClient) getHeaders() []api.StringPair {
	baseHeaders := []api.StringPair{
		{Name: "Authorization", Value: c.Token},
		{Name: "Accept", Value: contentType},
		{Name: "Content-Type", Value: contentType},
	}

	return baseHeaders
}

func (c *OxClient) executeGraphQLRequest(requestBody *GraphQLRequest) ([]byte, error) {
	headers := c.getHeaders()
	res, statusCode, err := api.SendHttpRequest(
		c.Client,
		"POST",
		c.Url,
		requestBody,
		headers,
		nil,
	)

	if err != nil {
		slog.Error("failed sending request", "err", err, "status", statusCode, "url", c.Url)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("failed getting response", "status", statusCode, "url", c.Url)
		return nil, fmt.Errorf("failed sending request to %s, status: %d", c.Url, statusCode)
	}

	return res, nil
}

func (c *OxClient) GetIssues(input GetIssuesInput) (*GetIssuesResponse, error) {
	variables := map[string]interface{}{
		"getIssuesInput": input,
	}

	requestBody := GraphQLRequest{
		Query:     GetIssuesQuery,
		Variables: variables,
	}

	responseBytes, err := c.executeGraphQLRequest(&requestBody)

	if err != nil {
		slog.Error("Failed getting issues", "err", err)
		return nil, err
	}

	var resp GetIssuesResponse
	if err := json.Unmarshal(responseBytes, &resp); err != nil {
		slog.Error("Failed unmarshalling GetIssues response", "err", err)
		return nil, err
	}

	return &resp, nil
}

func (c *OxClient) ExcludeIssues(issues []ExcludedIssue) error {
	inputs := make([]ExcludeAlertInput, len(issues))
	for i, issue := range issues {
		inputs[i] = ExcludeAlertInput{
			OxIssueID: issue.Issue.IssueID,
			Comment:   issue.Issue.Comment + "\n" + issue.Reason,
		}
	}

	variables := map[string]interface{}{
		"input": inputs,
	}

	requestBody := GraphQLRequest{
		Query:     ExcludeBulkAlertsMutation,
		Variables: variables,
	}

	_, err := c.executeGraphQLRequest(&requestBody)
	if err != nil {
		slog.Error("Failed excluding issues", "err", err)
		return err
	}

	return nil
}

// Ox does not support new line support in comment
// Comment length is unknown
func (c *OxClient) AddCommentToIssue(issue Issue, comment string) error {
	input := AddCommentToIssueInput{
		IssueID: issue.IssueID,
		Comment: issue.Comment + "\n" + comment,
	}

	req := GraphQLRequest{
		Query: AddCommentToIssueMutation,
		Variables: map[string]interface{}{
			"input": input,
		},
	}

	_, err := c.executeGraphQLRequest(&req)
	if err != nil {
		slog.Error("Failed adding comment to issue", "err", err)
		return fmt.Errorf("failed to add comment to issue: %w", err)
	}
	return nil
}
