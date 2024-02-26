package api

import (
	"fmt"
	"net/http"
	"testing"
)

func TestAuthentication(t *testing.T) {
	authToken := "thisisjustatribute"
	fakeRoundTripper := fakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
		authValues := req.Header.Values("Authorization")
		if len(authValues) == 0 {
			t.Fatalf("no auth header")
		}

		if len(authValues) > 1 {
			t.Fatalf("multple auth headers %v", authValues)
		}
		auth := authValues[0]
		if auth == "" {
			t.Fatalf("empty auth header value")
		}

		expected := fmt.Sprintf("Basic %s", authToken)
		if auth != expected {
			t.Fatalf("bad token value in header; got %s, expected %s", auth, expected)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}
	server := Server{Client: client, AuthToken: authToken}

	err := server.CheckAuthenticationValid()
	if err != nil {
		t.Fatalf("got error %v", err)
	}

}

func TestAuthenticaionFailureOnStatusCode(t *testing.T) {
	authToken := "thisisjustatribute"
	statusCodes := []struct {
		code int
		ok   bool
	}{{100, false}, {101, false}, {200, true}, {201, true}, {300, false}, {301, false}, {400, false}, {403, false}, {404, false}, {500, false}, {501, false}, {502, false}}

	for _, testCase := range statusCodes {

		t.Run(fmt.Sprintf("code_%d", testCase.code), func(t *testing.T) {

			fakeRoundTripper := fakeRoundTripper{statusCode: testCase.code}
			client := http.Client{Transport: fakeRoundTripper}
			server := Server{Client: client, AuthToken: authToken}

			err := server.CheckAuthenticationValid()
			if testCase.ok && err != nil {
				t.Fatalf("got error %v for code %d", err, testCase.code)
			}
			if !testCase.ok && err == nil {
				t.Fatalf("expected error for code %d", testCase.code)
			}
		})
	}
}
