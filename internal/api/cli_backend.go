package api

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

type CliServer struct {
	client    http.Client
	project   string
	authToken string
}

func (s CliServer) GetPackageChunkSize() int {
	return 800
}
func (s CliServer) GetRemoteConfigChunkSize() int {
	return 800
}

func NewCliServer(token string, project string, client http.Client) *CliServer {
	return &CliServer{
		client:    client,
		project:   project,
		authToken: buildAuthToken(token, project), // allowed to be empty if not auth
	}
}

func sendSealApiRequest[RequestType any, ResponseType any](client http.Client, method string, path string, body *RequestType, headers []StringPair, params []StringPair) (*ResponseType, int, error) {
	reqUrl, err := url.JoinPath(BaseURL, path) // uses the default cli server

	if err != nil {
		slog.Error("failed joining url path", "err", err)
		return nil, 0, err
	}

	return sendSealRequestJson[RequestType, ResponseType](client, method, reqUrl, body, headers, params)
}

func (s CliServer) QueryPackages(request *BulkCheckRequest, queryType PackageQueryType) (*Page[PackageVersion], error) {
	var param StringPair

	if queryType == OnlyFixed {
		param = StringPair{Name: "fixed", Value: "1"}
	} else {
		param = StringPair{Name: "fixed", Value: "0"}
	}

	var headers []StringPair

	if s.authToken != "" {
		// send token if we have it configured
		headers = []StringPair{BuildBasicAuthHeader(s.authToken)}
		common.Trace("sending auth header in bulk request")
	}

	// adding sort for deterministic order in request
	slices.SortFunc(request.Entries, func(a, b common.Dependency) int { return strings.Compare(a.Id(), b.Id()) })

	data, statusCode, err := sendSealApiRequest[BulkCheckRequest, Page[PackageVersion]](
		s.client,
		"POST",
		"/unauthenticated/v1/bulk",
		request,
		headers,
		[]StringPair{param},
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

func (s CliServer) QueryPackagesAuth(request *BulkCheckRequest, queryType PackageQueryType, generateActivity bool) (*Page[PackageVersion], error) {
	params := []StringPair{}
	if queryType == OnlyFixed {
		params = append(params, StringPair{Name: "fixed", Value: "1"})
	} else {
		params = append(params, StringPair{Name: "fixed", Value: "0"})
	}

	if generateActivity {
		common.Trace("will instruct server to generate activity for the requested packages")
		params = append(params, StringPair{Name: "store", Value: "1"}) // defaults to false
	}

	if s.authToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers := []StringPair{BuildBasicAuthHeader(s.authToken)}

	// adding sort for deterministic order in request
	slices.SortFunc(request.Entries, func(a, b common.Dependency) int { return strings.Compare(a.Id(), b.Id()) })

	data, statusCode, err := sendSealApiRequest[BulkCheckRequest, Page[PackageVersion]](
		s.client,
		"POST",
		fmt.Sprintf("/authenticated/v1/scan/%s", s.project),
		request,
		headers,
		params,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

// performs the BE request to get the approved remote config
func (s CliServer) QueryRemoteConfig(query []RemoteOverrideQuery) (*Page[PackageVersion], error) {

	var headers []StringPair

	if s.authToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers = []StringPair{BuildBasicAuthHeader(s.authToken)}
	common.Trace("sending auth header in bulk request")

	// fail_if_disabled param will cause the server to return 405 if the remote configuration
	// is disabled. This allows us to differentiate between disabled and empty responses
	data, statusCode, err := sendSealApiRequest[[]RemoteOverrideQuery, Page[PackageVersion]](
		s.client,
		"POST",
		fmt.Sprintf("/authenticated/v1/fixes/remote/%s", s.project),
		&query,
		headers,
		[]StringPair{fail_if_disabled},
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		if statusCode == 404 {
			// specific case for non-existent project
			return nil, NonExistentProjectError
		}
		if statusCode == 405 {
			return nil, RemoteOverrideDisabledError
		}

		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

func (s CliServer) QuerySilenceRules() ([]SilenceRule, error) {
	if s.authToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers := []StringPair{BuildBasicAuthHeader(s.authToken)}

	data, statusCode, err := sendSealApiRequest[any, []SilenceRule](
		s.client,
		"GET",
		fmt.Sprintf("/authenticated/v1/scanner_exclusions/%s", s.project),
		nil,
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return *data, nil
}

func (s CliServer) CheckAuthenticationValid() error {
	defer common.ExecutionTimer().Log()

	if s.authToken == "" {
		slog.Error("missing auth token for authentication")
		return MissingTokenForApiRequest
	}

	_, statusCode, err := sendSealRequest[any](
		s.client,
		"GET",
		AuthURL,
		nil,
		[]StringPair{BuildBasicAuthHeader(s.authToken)},
		nil,
	)

	if err != nil {
		slog.Error("failed sending request", "err", err)
		return err
	}

	if statusCode < 200 || statusCode >= 300 {
		slog.Error("server returned bad status code for authentication test", "status", statusCode)
		return common.NewPrintableError("authentication failed with error %d", statusCode)
	}

	return nil
}

func (s CliServer) InitializeProject(displayName string) (*ProjectDescriptor, error) {
	if s.authToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers := []StringPair{BuildBasicAuthHeader(s.authToken)}

	data, statusCode, err := sendSealApiRequest[ProjectInitRequest, ProjectDescriptor](
		s.client,
		"POST",
		"/authenticated/v1/project",
		&ProjectInitRequest{Tag: s.project, Name: displayName},
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

func (s CliServer) QueryMavenGroupIds(lookup *MavenGroupIDLookupList) (*Page[MavenGroupIDLookupResult], error) {
	var headers []StringPair
	uri := "/unauthenticated/v1/maven_groupid_lookup"

	if s.authToken != "" {
		// send token if we have it configured
		headers = []StringPair{BuildBasicAuthHeader(s.authToken)}
		common.Trace("sending auth header in bulk request")
		uri = "/authenticated/v1/maven_groupid_lookup"
	}

	data, statusCode, err := sendSealApiRequest[MavenGroupIDLookupList, Page[MavenGroupIDLookupResult]](
		s.client,
		"POST",
		uri,
		lookup,
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}

func (s CliServer) GetPublicKey() (string, error) {
	var headers []StringPair
	uri := "/unauthenticated/v1/signature/public_key"

	data, statusCode, err := sendSealApiRequest[any, map[string]string](
		s.client,
		"GET",
		uri,
		nil,
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return "", BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return "", err
	}

	if _, ok := (*data)["public_key"]; !ok {
		slog.Error("public key not found in response")
		return "", common.NewPrintableError("public key not found in response")
	}

	return (*data)["public_key"], nil
}

func (s CliServer) GetSignatures(query *ArtifactUniqueIdentifierList) ([]ArtifactMetadataResponse, error) {
	uri := "/authenticated/v1/bulk_query_artifact_metadata"

	if s.authToken == "" {
		slog.Error("missing auth token for querying artifact signatures")
		return nil, MissingTokenForApiRequest
	}

	headers := []StringPair{BuildBasicAuthHeader(s.authToken)}

	data, statusCode, err := sendSealApiRequest[ArtifactUniqueIdentifierList, []ArtifactMetadataResponse](
		s.client,
		"POST",
		uri,
		query,
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return *data, nil
}
