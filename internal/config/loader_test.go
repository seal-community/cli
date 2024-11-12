package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

var emptyEnv = EnvMap{}

func TestEmptyConfigFile(t *testing.T) {
	content := ``
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}
}

func TestEmptyConfigFileOverriddenByEnv(t *testing.T) {
	content := ``
	envToken := "abc"
	config, err := Load(strings.NewReader(content), EnvMap{"SEAL_TOKEN": envToken})

	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Token.Value() != envToken {
		t.Fatalf("failed override with env - got %s expected %s", config.Token, envToken)
	}
}

func TestNoExtraFields(t *testing.T) {
	content := `aaa:123`
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config != nil {
		t.Fatalf("allowed extraneous field in config: %v", config)
	}

	if err != FailedParsingConfYaml {
		t.Fatalf("should fail parsing yaml with extraneous field: %v", err)
	}
}

func TestNoDupFields(t *testing.T) {
	firstVal := "123"
	secondVal := "456"
	content := fmt.Sprintf("token: %s\token: %s", firstVal, secondVal)
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config != nil {
		t.Fatalf("allowed duplicate field in config: %v", config)
	}

	if err != FailedParsingConfYaml {
		t.Fatalf("should fail parsing yaml with dup fields field: %v", err)
	}
}

func TestNonemptyConfigFile(t *testing.T) {
	tokenValue := "abcd"
	content := fmt.Sprintf("token: %s", tokenValue)
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Token.Value() != tokenValue {
		t.Fatalf("failed parsing content - got %s expected %s", config.Token, tokenValue)
	}
}

func TestNonemptyConfigFileOverriddenByEnv(t *testing.T) {
	content := `token: abcd`
	envToken := "123"
	config, err := Load(strings.NewReader(content), EnvMap{"SEAL_TOKEN": envToken})

	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Token.Value() != envToken {
		t.Fatalf("failed override exsiting value with env - got %s expected %s", config.Token, envToken)
	}
}

func TestDefaultValue(t *testing.T) {
	content := ``
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Npm.ProdOnlyDeps != false {
		t.Fatalf("wrong default value in config - got %v expected %v", config.Npm.ProdOnlyDeps, false)
	}
}

func TestDefaultValueOverriddenByConfig(t *testing.T) {
	content := "npm:\n  prod-only: true"
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Npm.ProdOnlyDeps != true {
		t.Fatalf("wrong default value in config - got %v expected %v", config.Npm.ProdOnlyDeps, true)
	}
}

func TestDefaultValueOverriddenByEnv(t *testing.T) {
	content := ``
	config, err := Load(strings.NewReader(content), EnvMap{"SEAL_NPM_PROD_ONLY": "1"})

	if config == nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Npm.ProdOnlyDeps != true {
		t.Fatalf("wrong default value in config - got %v expected %v", config.Npm.ProdOnlyDeps, true)
	}
}

func TestProjectId(t *testing.T) {
	content := "project: proj-id-1"
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Project != "proj-id-1" {
		t.Fatalf("wrong project value - got %v", config.Project)
	}
}

func TestProjectIdEmpty(t *testing.T) {
	content := "project: \n"
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Project != "" {
		t.Fatalf("wrong project value - got %v", config.Project)
	}
}

func TestProjectsMap(t *testing.T) {
	content := "projects: \n  proj-id-1:\n    targets:\n      - package.json"
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Project != "" {
		t.Fatalf("wrong project value - got %v", config.Project)
	}

	if len(config.ProjectMap) != 1 {
		t.Fatalf("wrong projct map size: %d", len(config.ProjectMap))
	}

	projInfo, ok := config.ProjectMap["proj-id-1"]
	if !ok {
		t.Fatalf("proj id not found")
	}

	if len(projInfo.Targets) != 1 {
		t.Fatalf("wrong number of targets: %d", len(projInfo.Targets))
	}

	if projInfo.Targets[0] != "package.json" {
		t.Fatalf("wrong target: %s", projInfo.Targets[0])
	}
}

func TestProjectsMapAndProject(t *testing.T) {
	// not really supported

	content := "project: proj-id-3\nprojects: \n  proj-id-1:\n    targets:\n      - package.json"
	config, err := Load(strings.NewReader(content), emptyEnv)

	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if config.Project != "proj-id-3" {
		t.Fatalf("wrong project value - got %v", config.Project)
	}

	if len(config.ProjectMap) != 1 {
		t.Fatalf("wrong projct map size: %d", len(config.ProjectMap))
	}

	projInfo, ok := config.ProjectMap["proj-id-1"]
	if !ok {
		t.Fatalf("proj id not found")
	}

	if len(projInfo.Targets) != 1 {
		t.Fatalf("wrong number of targets: %d", len(projInfo.Targets))
	}

	if projInfo.Targets[0] != "package.json" {
		t.Fatalf("wrong target: %s", projInfo.Targets[0])
	}
}

func TestStringRedacted(t *testing.T) {
	content := "token: this-is-my-secret-token\nproject: proj-id-3\nprojects: \n  proj-id-1:\n    targets:\n      - package.json"
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if tokenStr := config.Token.String(); tokenStr != redactedString {
		t.Fatalf("token str not redacted `%s`", tokenStr)
	}
}

func TestFmtRedacted(t *testing.T) {
	token := "this-is-my-secret-token"
	content := fmt.Sprintf("token: %s\nproject: proj-id-3\nprojects: \n  proj-id-1:\n    targets:\n      - package.json", token)
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	res := fmt.Sprintf("%v", config)
	if res == "" {
		t.Fatalf("failed formatting config")
	}

	if strings.Contains(res, token) {
		t.Fatalf("token not redacted in `%s`", res)
	}
}

func TestLogRedacted(t *testing.T) {
	token := "this-is-my-secret-token"
	content := fmt.Sprintf("token: %s\nproject: proj-id-3\nprojects: \n  proj-id-1:\n    targets:\n      - package.json", token)
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	var b bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&b,
		&slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("test sensitive", "config", config)

	logData := b.String()

	if strings.Contains(logData, token) {
		t.Fatalf("log data contains token:\n`%s`", logData)
	}
}

func TestLogNotTruncated(t *testing.T) {
	token := "this-is-my-secret-token"
	projId := "proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-123-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-proj-idproj-idproj-idproj-123"
	content := fmt.Sprintf(`token: %s
project: %s
`, token, projId)
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	var b bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&b,
		&slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("test sensitive", "config", config)

	logData := b.String()

	if !strings.Contains(logData, projId) {
		t.Fatalf("output truncated:\n`%s`", logData)
	}
}

func TestValidateJfrogSanity(t *testing.T) {
	content := `jfrog:
  enabled: true
  host: "my-domain.com"
`
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config == nil || err != nil {
		t.Fatalf("failed loading config: %v", err)
	}
}

func TestValidateJfrogBadHostScheme(t *testing.T) {
	content := `jfrog:
  enabled: true
  host: "http://my-domain.com"
`
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config != nil || err == nil {
		t.Fatalf("should fail loading config: %v", config)
	}

	if err != InvalidJFrogHostScheme {
		t.Fatalf("wrong err: `%v` exepcted `%v`", err, InvalidJFrogHostScheme)
	}
}

func TestValidateJfrogBadHost(t *testing.T) {
	content := `jfrog:
  enabled: true
  host: "www\n.exe"
`
	config, err := Load(strings.NewReader(content), emptyEnv)
	if config != nil || err == nil {
		t.Fatalf("should fail loading config: %v", config)
	}

	if err != InvalidJFrogHost {
		t.Fatalf("wrong err: `%v` exepcted `%v`", err, InvalidJFrogHost)
	}
}
