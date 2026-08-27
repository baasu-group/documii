package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/documi/documi/services/core/internal/modules/documents/editors"
	"github.com/documi/documi/services/core/internal/modules/documents/editors/collabora"
	"github.com/documi/documi/services/core/internal/modules/system"
)

type Readiness interface {
	Ready() bool
}

type StaticReadiness bool

func (readiness StaticReadiness) Ready() bool { return bool(readiness) }

type Routes struct {
	System  *system.Handler
	Editors *editors.Handler
	WOPI    *collabora.Handler
}

func NewRouter(logger *slog.Logger, routes Routes, readiness Readiness) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /ready", func(writer http.ResponseWriter, _ *http.Request) {
		if !readiness.Ready() {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/system/info", routes.System.SystemInfo)
	if routes.Editors != nil {
		mux.HandleFunc("GET /api/v1/document-editors", routes.Editors.Providers)
		mux.HandleFunc("GET /api/v1/document-editors/{provider}/status", routes.Editors.ProviderStatus)
		mux.HandleFunc("POST /api/v1/documents/{id}/editor-session", routes.Editors.CreateSession)
	}
	if routes.WOPI != nil {
		mux.HandleFunc("GET /api/v1/wopi/files/{id}", routes.WOPI.CheckFileInfo)
		mux.HandleFunc("GET /api/v1/wopi/files/{id}/contents", routes.WOPI.GetFile)
		mux.HandleFunc("POST /api/v1/wopi/files/{id}/contents", routes.WOPI.PutFile)
	}
	return requestLogger(logger, mux)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(writer, request)
		logger.Info("http request", "method", request.Method, "path", request.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
