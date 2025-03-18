package api

import (
	"cli/internal/common"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

type CliJfrogServer struct {
	client  http.Client
	project string

	baseUrl string

	// using bearer
	authHeader StringPair
}

const payloadParamName = "extra"
const jfrogQueryParamLimit = 7000 // tested to work via JFrog, length of entire query string

func (s CliJfrogServer) GetPackageChunkSize() int {
	// these chunks contain a larger object, so need to be small enough to not
	// reach the GET query limit, when sending via JFrog / to our BE
	return 20
}

func (s CliJfrogServer) GetRemoteConfigChunkSize() int {
	return 100
}

func NewCliJfrogServer(client http.Client, project string, token string, baseUrl string) *CliJfrogServer {

	authHeader := BuildBearerAuthHeader(token)

	return &CliJfrogServer{
		client:  client,
		project: project,
		baseUrl: baseUrl,

		authHeader: authHeader,
	}
}

func (s CliJfrogServer) sendPayload(uri string, bodyObj any, params []StringPair, response any) (statusCode int, err error) {
	statusCode = 0

	if response == nil {
		// wrong usage by caller
		slog.Error("bad response object - nil")
		err = fmt.Errorf("response object requried for unmarshaling request")
		return
	}

	payload := payloadify(bodyObj)
	if payload == "" {
		err = fmt.Errorf("failed formatting payload for jfrog")
		return
	}

	target, err := url.JoinPath(s.baseUrl, uri)
	if err != nil {
		slog.Error("failed building target url for jfrog", "err", err, "baseUrl", s.baseUrl, "uri", uri)
		return
	}

	if params == nil {
		params = make([]StringPair, 0, 1)
	} else {
		// unlikely, but make sure caller did not use same name as our payload
		if paramExists(params, payloadParamName) {
			slog.Error("payload param name already provided", "name", payloadParamName, "params", params)
			err = fmt.Errorf("cannot override payload param")
			return
		}
	}

	params = append(params, StringPair{Name: payloadParamName, Value: payload})

	// make sure query string does not exceed limit by JFrog
	qsLen := calculateQuerystringLength(params)
	if qsLen > jfrogQueryParamLimit {
		// nothing the user can do about this
		slog.Error("jfrog payload too long", "limit", jfrogQueryParamLimit, "got", qsLen)
		err = fmt.Errorf("jfrog get request size too big")
		return
	}

	responseData, statusCode, err := sendSealRequest[any](
		s.client,
		"GET",
		target,
		nil,
		[]StringPair{s.authHeader},
		params,
	)

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return
	}

	common.Trace("received json response", "data", string(responseData), "status", statusCode)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		err = BadServerResponseCode
		return
	}

	if err = json.Unmarshal(responseData, response); err != nil {
		slog.Error("failed unmarshal response body", "body", string(responseData))
		return
	}

	return
}

func (s CliJfrogServer) QueryPackages(request *BulkCheckRequest, queryType PackageQueryType) (*Page[PackageVersion], error) {
	return nil, common.NewPrintableError("unauthenticated JFrog usage is not possible")
}

func (s CliJfrogServer) QueryPackagesAuth(request *BulkCheckRequest, queryType PackageQueryType, generateActivity bool) (*Page[PackageVersion], error) {

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

	// adding sort for deterministic order in request
	slices.SortFunc(request.Entries, func(a, b common.Dependency) int { return strings.Compare(a.Id(), b.Id()) })

	var resp Page[PackageVersion]
	if _, err := s.sendPayload(fmt.Sprintf("/scan/%s", s.project), request, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// performs the BE request to get the approved remote config
func (s CliJfrogServer) QueryRemoteConfig(query []RemoteOverrideQuery) (*Page[PackageVersion], error) {
	var resp Page[PackageVersion]
	// fail_if_disabled param will cause the server to return 405 if the remote configuration
	// is disabled. This allows us to differentiate between disabled and empty responses
	statusCode, err := s.sendPayload(fmt.Sprintf("/fixes/remote/%s", s.project), query, []StringPair{fail_if_disabled}, &resp)
	if statusCode == 404 {
		// specific case for non-existent project
		slog.Error("project not found", "err", err, "project", s.project)
		return nil, NonExistentProjectError
	}

	if statusCode == 405 {
		return nil, RemoteOverrideDisabledError
	}

	if err != nil {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, err
	}

	return &resp, nil
}

// checks auth, equivalent to:
//
//	curl -H"Authorization: Bearer {TOKEN}" -L -v "https://{host}/artifactory/{repo-key}" -X OPTIONS
func (s CliJfrogServer) CheckAuthenticationValid() error {

	defer common.ExecutionTimer().Log()
	_, statusCode, err := sendSealRequest[any](
		s.client,
		"OPTIONS",
		s.baseUrl,
		nil,
		[]StringPair{s.authHeader},
		nil,
	)

	if err != nil {
		slog.Error("failed sending request", "err", err)
		return err
	}

	if statusCode < 200 || statusCode >= 300 {
		slog.Error("jforg server returned bad status code for authentication test", "status", statusCode)
		return common.NewPrintableError("JFrog authentication failed with error %d", statusCode)
	}

	return nil
}

func payloadify(t any) string {
	bs, err := json.Marshal(t)
	if err != nil {
		slog.Error("failed dumping object json", "err", err, "data", t)
		return ""
	}

	return base64.StdEncoding.EncodeToString(bs)
}

func (s CliJfrogServer) InitializeProject(displayName string) (*ProjectDescriptor, error) {

	var resp ProjectDescriptor
	if _, err := s.sendPayload("/project", ProjectInitRequest{Tag: s.project, Name: displayName}, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s CliJfrogServer) QuerySilenceRules() ([]SilenceRule, error) {
	uri := fmt.Sprintf("/scanner_exclusions/%s", s.project)
	target, err := url.JoinPath(s.baseUrl, uri)
	if err != nil {
		slog.Error("failed building target url for jfrog", "err", err, "baseUrl", s.baseUrl, "uri", uri)
		return nil, err
	}

	resp, statusCode, err := sendSealRequest[[]byte](
		s.client,
		"GET",
		target,
		nil,
		[]StringPair{s.authHeader},
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

	var data []SilenceRule
	if err = json.Unmarshal(resp, &data); err != nil {
		slog.Error("failed unmarshal response body", "body", string(resp))
		return nil, err
	}

	return data, nil
}

func (s CliJfrogServer) QueryMavenGroupIds(lookup *MavenGroupIDLookupList) (*Page[MavenGroupIDLookupResult], error) {

	params := []StringPair{}
	var resp Page[MavenGroupIDLookupResult]
	if _, err := s.sendPayload("/authenticated/v1/maven_groupid_lookup", lookup, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
