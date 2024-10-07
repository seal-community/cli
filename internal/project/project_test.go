package project

import (
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"strings"
	"testing"
)

func TestGenerateProjectDisplayNameNoProjectName(t *testing.T) {
	manager := &shared.FakePackageManager{}
	if dispName := GenerateProjectDisplayName(manager, "src/baubau"); dispName != "baubau" {
		t.Fatalf("failed genereting display name: %s", dispName)
	}
}

func TestGenerateProjectDisplay(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "hoeh"}
	if dispName := GenerateProjectDisplayName(manager, "src/baubau"); dispName != "hoeh" {
		t.Fatalf("failed genereting display name: %s", dispName)
	}
}

func TestGenerateProjectDisplayNormalized(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "bau bau"}
	if dispName := GenerateProjectDisplayName(manager, "src/proj"); dispName != "bau-bau" {
		t.Fatalf("failed genereting display name: %s", dispName)
	}
}

func TestGenerateProjectDisplayNameNoProjectNameNormalized(t *testing.T) {
	manager := &shared.FakePackageManager{}
	if dispName := GenerateProjectDisplayName(manager, "src/bau bau"); dispName != "bau-bau" {
		t.Fatalf("failed genereting display name: %s", dispName)
	}
}

func TestNormalizeTargetNothing(t *testing.T) {
	if res := NormalizeTarget("src/package.json"); res != "src/package.json" {
		t.Fatalf("got %s instead", res)
	}
}
func TestNormalizeTargetWindows(t *testing.T) {
	if res := NormalizeTarget("src\\package.json"); res != "src/package.json" {
		t.Fatalf("got %s instead", res)
	}
}

func TestHashProjectDescriptor(t *testing.T) {
	if res := hashProjectDescriptor(""); res != "DA39A3EE5E6B4B0D3255BFEF95601890AFD80709" {
		t.Fatalf("got %s instead", res)
	}
}

func TestFindProjectIdBadInput(t *testing.T) {
	target := "\\\\share\\hello"

	projMap := map[string]config.ProjectInfo{
		"proj-id-1": {
			Targets: []string{"requirements.txt"}},
	}

	pid := findProjectIdByTarget(projMap, target)
	if pid != "" {
		t.Fatalf("should have failed - pid: %s", pid)
	}
}

func TestFindProjectIdTargetFound(t *testing.T) {
	target := "requirements.txt"

	projMap := map[string]config.ProjectInfo{
		"proj-id-1": {
			Targets: []string{"requirements.txt"}},
	}

	pid := findProjectIdByTarget(projMap, target)
	if pid == "" {
		t.Fatalf("failed -  pid: %s", pid)
	}

	if pid != "proj-id-1" {
		t.Fatalf("wrong project id %s", pid)
	}
}

func TestFormatProjectIdForRepo(t *testing.T) {
	target := "requirements.txt"
	remoteUrl := "https://github.com/seal-community/cli"

	payload := formatProjectIdForRepo(target, remoteUrl)
	if payload == "" {
		t.Fatalf("failed -  pid: %s", payload)
	}

	if payload != "seal-community/cli/requirements.txt" {
		t.Fatalf("wrong project id %s", payload)
	}
}

func TestFormatProjectIdForRepoFails(t *testing.T) {
	payload := formatProjectIdForRepo("requirements.txt", "aa\ncc") // should fail according to internal code in url.Parse
	if payload != "" {
		t.Fatalf("failed -  pid: %s", payload)
	}
}

func TestFormatProjectIdForRepoSubpath(t *testing.T) {
	target := "path/to/src/requirements.txt"
	remoteUrl := "https://github.com/seal-community/cli"

	payload := formatProjectIdForRepo(target, remoteUrl)
	if payload == "" {
		t.Fatalf("failed -  pid: %s", payload)
	}

	if payload != "seal-community/cli/path/to/src/requirements.txt" {
		t.Fatalf("wrong project id %s", payload)
	}
}

func TestFindProjectIdCaseSensitiveNotFound(t *testing.T) {
	target := "requirements.txt"

	projMap := map[string]config.ProjectInfo{
		"proj-id-1": {
			Targets: []string{"Requirements.txt"}},
	}

	pid := findProjectIdByTarget(projMap, target)
	if pid != "" {
		t.Fatalf("should have failed - pid: %s", pid)
	}
}

func TestFindProjectIdNoMap(t *testing.T) {
	target := "requirements.txt"

	projMap := map[string]config.ProjectInfo{}

	pid := findProjectIdByTarget(projMap, target)
	if pid != "" {
		t.Fatalf("should have failed - pid: %s", pid)
	}
}
func TestFindProjectIdMapNil(t *testing.T) {
	target := "requirements.txt"

	pid := findProjectIdByTarget(nil, target)
	if pid != "" {
		t.Fatalf("should have failed - pid: %s", pid)
	}
}

func TestFindProjectIdOsMismatchFromUnix(t *testing.T) {
	target := "src/requirements.txt"
	definedProjId := "proj-id-1"
	projMap := map[string]config.ProjectInfo{
		definedProjId: {
			Targets: []string{"src\\requirements.txt"}},
	}

	pid := findProjectIdByTarget(projMap, target)
	if pid != definedProjId {
		t.Fatalf("did not find correct proj id - pid: %s", pid)
	}
}

func TestFindProjectIdOsMismatchFromWindows(t *testing.T) {
	target := "src\\requirements.txt"
	definedProjId := "proj-id-1"
	projMap := map[string]config.ProjectInfo{
		definedProjId: {
			Targets: []string{"src/requirements.txt"}},
	}

	pid := findProjectIdByTarget(projMap, target)
	if pid != definedProjId {
		t.Fatalf("did not find correct proj id - pid: %s", pid)
	}
}

func TestChooseProjectIdUseGivenWithoutRepo(t *testing.T) {
	manager := &shared.FakePackageManager{}
	targetFile := "requirements.txt"
	projectDir := "/path/to/project_directory"
	projId := "my-proj-id"
	projMap := map[string]config.ProjectInfo{}

	remoteUrl := ""

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	if selectedId != projId {
		t.Fatalf("bad proj id: %v expected %v", selectedId, projId)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseRepoUseProjecInMap(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "baubau"}
	targetFile := "requirements.txt"
	projectDir := "/path/to/project_directory"
	projId := hashProjectDescriptor("seal-community/cli/requirements.txt")

	projMap := map[string]config.ProjectInfo{
		projId: {
			Targets: []string{"requirements.txt"}},
	}

	remoteUrl := "https://github.com/seal-community/cli"

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, "", projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if !isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	if selectedId != projId {
		t.Fatalf("bad proj id: %v expected %v", selectedId, projId)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseUseRepoSubpath(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "baubau"}
	targetFile := "vendor/requirements.txt"
	projectDir := "/path/to/project_directory"
	projMap := map[string]config.ProjectInfo{}
	remoteUrl := "https://github.com/seal-community/cli"

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, "", projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	expected := hashProjectDescriptor("seal-community/cli/vendor/requirements.txt")
	if selectedId != expected {
		t.Fatalf("bad proj id: %v expected %v", selectedId, expected)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseUseRepo(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "baubau"}
	targetFile := "go.mod"
	projectDir := "/path/to/project_directory"
	projMap := map[string]config.ProjectInfo{}
	remoteUrl := "https://github.com/seal-community/cli"

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, "", projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	expected := hashProjectDescriptor("seal-community/cli/go.mod") // == f250588af358d75f096c6d6f7f5b03cf994f0d62
	if selectedId != expected {
		t.Fatalf("bad proj id: %v expected %v", selectedId, expected)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseProjectIdUseGivenWithRepo(t *testing.T) {
	manager := &shared.FakePackageManager{}
	targetFile := "requirements.txt"
	projectDir := "/path/to/project_directory"
	projId := "my-proj-id"
	projMap := map[string]config.ProjectInfo{}

	remoteUrl := "https://github.com/seal-community/cli"

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	if selectedId != projId {
		t.Fatalf("bad proj id: %v expected %v", selectedId, projId)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseProjectIdUseFallbackNoManagerProject(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: ""}
	targetFile := "requirements.txt"
	projectDir := "/path/to/project_directory"
	projId := ""
	projMap := map[string]config.ProjectInfo{}

	remoteUrl := ""

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	expected := hashProjectDescriptor("project_directory/requirements.txt")
	if selectedId != expected {
		t.Fatalf("bad proj id: %v expected %v", selectedId, expected)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseProjectIdUseFallbackWithManagerProject(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "baubau-project"}
	targetFile := "requirements.txt"
	projectDir := "/path/to/project_directory"
	projId := ""
	projMap := map[string]config.ProjectInfo{}

	remoteUrl := ""

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	expected := hashProjectDescriptor("project_directory/requirements.txt/baubau-project")
	if selectedId != expected {
		t.Fatalf("bad proj id: %v expected %v", selectedId, expected)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseProjectIdUseFallbackWithManagerProjectIfRepoFailed(t *testing.T) {
	manager := &shared.FakePackageManager{ProjectName: "baubau-project"}
	targetFile := "requirements.txt"
	projectDir := "/path/to/project_directory"
	projId := ""
	projMap := map[string]config.ProjectInfo{}

	remoteUrl := "http://asd\nbbb" // should fail url.Parse

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	expected := hashProjectDescriptor("project_directory/requirements.txt/baubau-project")
	if selectedId != expected {
		t.Fatalf("bad proj id: %v expected %v", selectedId, expected)
	}

	if reason := ValidateProjectId(selectedId); reason != "" {
		t.Fatalf("chosen project id is not valid %s reason: %s", selectedId, reason)
	}
}

func TestChooseProjectIdMismatchIdFromMap(t *testing.T) {
	manager := &shared.FakePackageManager{}
	targetFile := "requirements.txt"
	projId := "provided-proj-id"
	projectDir := "/path/to/project_directory"
	projMap := map[string]config.ProjectInfo{
		"map-proj-id": {
			Targets: []string{"requirements.txt"}},
	}

	remoteUrl := "https://github.com/seal-community/cli"

	selectedId, _, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err == nil {
		t.Fatalf("failed without err %v", err)
	}

	if selectedId != "" {
		t.Fatalf("should fail genereting id %s", selectedId)
	}
}

func TestFormatProjectIdFallback(t *testing.T) {
	payload := formatProjectIdFallback("myproj", "requirements.txt", "my-proj")
	if payload != "myproj/requirements.txt/my-proj" {
		t.Fatalf("got bad payload: `%s`", payload)
	}
}

func TestFormatProjectIdPath(t *testing.T) {
	payload := formatProjectIdFallback("myproj", "src/requirements.txt", "my-proj")
	if payload != "myproj/src/requirements.txt/my-proj" {
		t.Fatalf("got bad payload: `%s`", payload)
	}
}

func TestFormatProjectIdFallbackNoProjectName(t *testing.T) {
	payload := formatProjectIdFallback("myproj", "requirements.txt", "")
	if payload != "myproj/requirements.txt" {
		t.Fatalf("got bad payload: `%s`", payload)
	}
}

func TestChooseProjectIdCalculated(t *testing.T) {
	manager := &shared.FakePackageManager{}
	projectDir := "/path/to/project_directory"
	targetFile := "requirements.txt"
	projId := ""
	projMap := map[string]config.ProjectInfo{}

	remoteUrl := "https://github.com/seal-community/cli"

	selectedId, isFound, err := ChooseProjectId(manager, projectDir, targetFile, projId, projMap, remoteUrl)
	if err != nil {
		t.Fatalf("failed with err %v", err)
	}

	if isFound {
		t.Fatalf("bad isFound %v", isFound)
	}

	if selectedId != "AA62AB9099EAE6310CDD00C074BD016F2CF9DC0C" {
		t.Fatalf("bad proj id: %v expected %v", selectedId, projId)
	}
}

func TestNormalizeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"tag", "tag"},
		{"TAG", "TAG"},
		{" tag ", "-tag-"},
		{" TAG ", "-TAG-"},
		{"!#$tag%$@", "tag"},
		{"tag.-_tag", "tag.-_tag"},
		{"com.github.seal-sec/demo", "com.github.seal-sec-demo"},
		{strings.Repeat("A", 300), strings.Repeat("A", MaxProjectNameLen)},
		{"a/b/c\\d e", "a-b-c-d-e"},
		{"åß∂ƒ®å∑ƒ®tag", "tag"},
		{"😀tag😀", "tag"},
		{"github.com/Masterminds/goutils", "github.com-Masterminds-goutils"},
		{"golang.org/x/crypto", "golang.org-x-crypto"},
		{"@group/name", "group-name"},
		{"org:java:some:artifact", "orgjavasomeartifact"},
		{"python_module", "python_module"},
		{"python-module", "python-module"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NormalizeProjectName(test.name)
			if result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}
