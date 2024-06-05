package phase

import (
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"testing"
)

func TestCalculatedFails(t *testing.T) {
	// until implemented
	if _, err := calculateProjectId(nil, "", ""); err == nil {
		t.Fatal("should fail not implemented")
	}
}

func TestFindProjectIdBadInput(t *testing.T) {
	dir := "/tmp/proj"
	target := "\\\\share\\hello"

	projMap := map[string]config.ProjectInfo{
		"proj-id-1": {
			Targets: []string{"requirements.txt"}},
	}

	pid, err := findProjectId(projMap, dir, target)
	if err == nil || pid != "" {
		t.Fatalf("should have failed: %v pid: %s", err, pid)
	}
}

func TestFindProjectIdTargetFound(t *testing.T) {
	dir := "/tmp/proj"
	target := "/tmp/proj/requirements.txt"

	projMap := map[string]config.ProjectInfo{
		"proj-id-1": {
			Targets: []string{"requirements.txt"}},
	}

	pid, err := findProjectId(projMap, dir, target)
	if err != nil || pid == "" {
		t.Fatalf("failed: %v pid: %s", err, pid)
	}

	if pid != "proj-id-1" {
		t.Fatalf("wrong project id %s", pid)
	}
}

func TestFindProjectIdCaseSensitiveNotFound(t *testing.T) {
	dir := "/tmp/proj"
	target := "/tmp/proj/requirements.txt"

	projMap := map[string]config.ProjectInfo{
		"proj-id-1": {
			Targets: []string{"Requirements.txt"}},
	}

	pid, err := findProjectId(projMap, dir, target)
	if err != nil || pid != "" {
		t.Fatalf("should have failed: %v pid: %s", err, pid)
	}
}

func TestFindProjectIdNoMap(t *testing.T) {
	dir := "/tmp/proj"
	target := "/tmp/proj/requirements.txt"

	projMap := map[string]config.ProjectInfo{}

	pid, err := findProjectId(projMap, dir, target)
	if err != nil || pid != "" {
		t.Fatalf("should have failed: %v pid: %s", err, pid)
	}
}
func TestFindProjectIdMapNil(t *testing.T) {
	dir := "/tmp/proj"
	target := "/tmp/proj/requirements.txt"

	pid, err := findProjectId(nil, dir, target)
	if err != nil || pid != "" {
		t.Fatalf("should have failed: %v pid: %s", err, pid)
	}
}

func TestGetProjectIdLegacyFolderName(t *testing.T) {
	c := &config.Config{}
	manager := &shared.FakePackageManager{
		ProjetName: "",
	}
	targetDir := "/tmp/proj/baubau/"
	targetFile := ""

	projId, err := getProjectId(c, manager, targetDir, targetFile)
	if err != nil {
		t.Fatalf("failed getting proj id %v", err)
	}

	// getting base name since fake manager returns "" as the project name
	if projId != "baubau" {
		t.Fatalf("wrong proj id: %s", projId)
	}
}

func TestGetProjectIdLegacyWithProjName(t *testing.T) {
	c := &config.Config{}
	manager := &shared.FakePackageManager{
		ProjetName: "my-proj",
	}
	targetDir := "/tmp/proj/baubau/"
	targetFile := ""

	projId, err := getProjectId(c, manager, targetDir, targetFile)
	if err != nil {
		t.Fatalf("failed getting proj id %v", err)
	}

	// getting base name since fake manager returns "" as the project name
	if projId != "my-proj" {
		t.Fatalf("wrong proj id: %s", projId)
	}
}
