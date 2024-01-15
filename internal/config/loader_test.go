package config

import (
	"fmt"
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

	if config.Token != envToken {
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

	if config.Token != tokenValue {
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

	if config.Token != envToken {
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
