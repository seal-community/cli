package phase

import (
	"cli/internal/config"
	b64 "encoding/base64"
	"fmt"
	"testing"
)

func TestBuildToken(t *testing.T) {
	token := "abc"
	project := "123"
	builtToken := buildAuthToken(&config.Config{Token: token, Project: project})
	buffer, err := b64.StdEncoding.DecodeString(builtToken)
	if err != nil {
		t.Fatalf("failed decoding built token from base64 err:%s b64:%s", err, builtToken)
	}
	expected := fmt.Sprintf("%s:%s", project, token)
	built := string(buffer)
	if expected != built {
		t.Fatalf("wrong token format, expected :%s found:%s", expected, built)
	}
}

func TestBuildTokenMissingFields(t *testing.T) {
	token := "abc"
	project := "123"
	builtToken := buildAuthToken(&config.Config{Project: project})
	if builtToken != "" {
		t.Fatalf("should have failed building auth from missing token :%s", builtToken)
	}

	builtToken = buildAuthToken(&config.Config{Token: token})
	if builtToken != "" {
		t.Fatalf("should have failed building auth from missing project :%s", builtToken)
	}

}
