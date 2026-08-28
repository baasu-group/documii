package integrations

import (
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGoogleOAuthUsesNarrowScopeAndOfficialEndpoints(t *testing.T) {
	connector := NewGoogleConnector(nil, nil, nil, "client", "secret", "https://documi.example/api/v1/connections/google/oauth/callback", "https://documi.example")
	if connector.oauth.Endpoint.AuthURL != "https://accounts.google.com/o/oauth2/v2/auth" || connector.oauth.Endpoint.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatal("unexpected Google OAuth endpoints")
	}
	url := connector.oauth.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", "challenge"), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	if !strings.Contains(url, "drive.file") || strings.Contains(url, "%2Fauth%2Fdrive+") {
		t.Fatalf("OAuth URL does not use narrow drive.file scope: %s", url)
	}
	if !strings.Contains(url, "code_challenge_method=S256") {
		t.Fatal("OAuth URL missing PKCE")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestGoogleDriveBoundaryUsesFixedBaseAndBearerToken(t *testing.T) {
	connector := NewGoogleConnector(nil, nil, nil, "client", "secret", "https://documi.example/callback", "https://documi.example")
	connector.driveBaseURL = "https://drive.test/drive/v3"
	connector.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://drive.test/drive/v3/files" || request.Header.Get("Authorization") != "Bearer short-lived" {
			t.Fatalf("unexpected request %s %q", request.URL, request.Header.Get("Authorization"))
		}
		encoded, _ := json.Marshal(driveFile{ID: "file-1", Name: "Plan", MIMEType: "application/vnd.google-apps.document", WebViewLink: "https://docs.google.com/document/d/file-1/edit"})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(encoded)))}, nil
	})}
	value, err := connector.driveRequest(context.Background(), "short-lived", http.MethodGet, "/files", nil)
	if err != nil || value.ID != "file-1" {
		t.Fatalf("drive request failed: %#v %v", value, err)
	}
}

func TestGoogleFormatsDoNotPretendProvidersAreInterchangeable(t *testing.T) {
	cases := map[string]string{"text": "google-docs", "spreadsheet": "google-sheets", "presentation": "google-slides"}
	for family, expected := range cases {
		provider, _, err := googleFormat(family)
		if err != nil || provider != expected {
			t.Fatalf("%s resolved to %s: %v", family, provider, err)
		}
	}
	if _, _, err := googleFormat("pdf"); err == nil {
		t.Fatal("unsupported Google family was accepted")
	}
}
