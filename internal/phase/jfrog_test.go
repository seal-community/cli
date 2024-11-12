package phase

import "testing"

func TestBuildRepoArtifactUrl(t *testing.T) {
	expected := "https://my-host.com/artifactory/seal-jfrog-repo-key"
	res := buildRepoArtifactUrl("https", "my-host.com", "seal-jfrog-repo-key")
	if res != expected {
		t.Fatalf("got `%s` expected `%s`", res, expected)
	}
}
