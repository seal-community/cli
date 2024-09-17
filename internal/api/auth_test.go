package api

import (
	"testing"
)

func TestBasicTokenHeader(t *testing.T) {
	h := BuildBasicAuthHeader("fake-token")
	if h.Value != "Basic fake-token" {
		t.Fatalf("bad auth header value: `%s`", h.Value)
	}

	if h.Name != "Authorization" {
		t.Fatalf("bad auth header name: %s", h.Name)
	}
}
func TestBearerTokenHeader(t *testing.T) {
	h := BuildBearerAuthHeader("fake-token")
	if h.Value != "Bearer fake-token" {
		t.Fatalf("bad auth header value: `%s`", h.Value)
	}

	if h.Name != "Authorization" {
		t.Fatalf("bad auth header name: %s", h.Name)
	}
}
