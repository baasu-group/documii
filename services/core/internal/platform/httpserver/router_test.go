package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/documi/documi/services/core/internal/modules/system"
)

func TestHealthAndSystemInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := system.NewService("test", "testing", true)
	router := NewRouter(logger, system.NewHandler(service), StaticReadiness(true))

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

func TestReadinessFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(logger, system.NewHandler(system.NewService("test", "testing", false)), StaticReadiness(false))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /ready = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
