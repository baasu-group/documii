package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/documi/documi/services/core/internal/modules/documents/editors"
	"github.com/documi/documi/services/core/internal/modules/documents/editors/collabora"
	"github.com/documi/documi/services/core/internal/modules/system"
)

func TestHealthAndSystemInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := system.NewService("test", "testing", true)
	router := NewRouter(logger, Routes{System: system.NewHandler(service)}, StaticReadiness(true))

	for _, test := range []struct {
		path, contains string
	}{
		{"/health", `"status":"ok"`},
		{"/ready", `"status":"ready"`},
		{"/api/v1/system/info", `"airGapped":true`},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), test.contains) {
			t.Fatalf("GET %s = %d %s", test.path, response.Code, response.Body.String())
		}
	}
}

func TestEditorAvailabilityAndWOPIRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry, err := editors.NewRegistry(editors.DefaultProviders(editors.ProviderConfiguration{})...)
	if err != nil {
		t.Fatal(err)
	}
	editorHandler := editors.NewHandler(
		editors.NewService(registry, editors.NewResolver(registry, editors.DefaultPolicy())),
		editors.UnavailableSessionService{},
	)
	tokens, _ := collabora.NewTokenService([]byte("0123456789abcdef0123456789abcdef"), time.Minute)
	wopiHandler, _ := collabora.NewHandler(tokens, collabora.UnavailableGateway{}, 1024)
	router := NewRouter(logger, Routes{
		System:  system.NewHandler(system.NewService("test", "testing", false)),
		Editors: editorHandler,
		WOPI:    wopiHandler,
	}, StaticReadiness(true))

	availability := httptest.NewRecorder()
	router.ServeHTTP(availability, httptest.NewRequest(http.MethodGet, "/api/v1/document-editors", nil))
	if availability.Code != http.StatusOK || !strings.Contains(availability.Body.String(), `"id":"native"`) {
		t.Fatalf("GET document-editors = %d %s", availability.Code, availability.Body.String())
	}

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/wopi/files/doc_1", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("GET WOPI without token = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}
}

func TestReadinessFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(logger, Routes{System: system.NewHandler(system.NewService("test", "testing", false))}, StaticReadiness(false))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /ready = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
