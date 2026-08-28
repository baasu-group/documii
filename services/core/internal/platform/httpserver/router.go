package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/documi/documi/services/core/internal/modules/administration"
	"github.com/documi/documi/services/core/internal/modules/documents"
	"github.com/documi/documi/services/core/internal/modules/documents/editors"
	"github.com/documi/documi/services/core/internal/modules/documents/editors/collabora"
	"github.com/documi/documi/services/core/internal/modules/integrations"
	"github.com/documi/documi/services/core/internal/modules/organizations"
	"github.com/documi/documi/services/core/internal/modules/reporting"
	"github.com/documi/documi/services/core/internal/modules/system"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
)

type Readiness interface {
	Ready() bool
}

type StaticReadiness bool

func (readiness StaticReadiness) Ready() bool { return bool(readiness) }

type ReadinessFunc func() bool

func (readiness ReadinessFunc) Ready() bool { return readiness() }

type Routes struct {
	System         *system.Handler
	Editors        *editors.Handler
	WOPI           *collabora.Handler
	Authenticator  platformauth.Authenticator
	AuthConfig     map[string]any
	Documents      *documents.Handler
	Administration *administration.Handler
	Organizations  *organizations.Handler
	Reporting      *reporting.Handler
	Integrations   *integrations.Handler
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
	mux.HandleFunc("GET /api/v1/auth/config", func(writer http.ResponseWriter, _ *http.Request) { writeJSON(writer, http.StatusOK, routes.AuthConfig) })
	if routes.Authenticator != nil {
		mux.Handle("GET /api/v1/me", platformauth.Middleware(routes.Authenticator, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			subject, _ := platformauth.FromContext(request.Context())
			writeJSON(writer, http.StatusOK, subject)
		})))
	}
	if routes.Editors != nil {
		mux.HandleFunc("GET /api/v1/document-editors", routes.Editors.Providers)
		mux.HandleFunc("GET /api/v1/document-editors/{provider}/status", routes.Editors.ProviderStatus)
		if routes.Authenticator != nil {
			mux.Handle("POST /api/v1/documents/{id}/editor-session", platformauth.Middleware(routes.Authenticator, http.HandlerFunc(routes.Editors.CreateSession)))
		} else {
			mux.HandleFunc("POST /api/v1/documents/{id}/editor-session", routes.Editors.CreateSession)
		}
	}
	if routes.Documents != nil && routes.Authenticator != nil {
		protected := func(handler http.HandlerFunc) http.Handler {
			return platformauth.Middleware(routes.Authenticator, handler)
		}
		mux.Handle("GET /api/v1/workspaces", protected(routes.Documents.Workspaces))
		mux.Handle("GET /api/v1/workspaces/{workspaceId}/documents", protected(routes.Documents.Documents))
		mux.Handle("POST /api/v1/workspaces/{workspaceId}/documents", protected(routes.Documents.Documents))
		mux.Handle("POST /api/v1/workspaces/{workspaceId}/document-uploads", protected(routes.Documents.Upload))
		mux.Handle("GET /api/v1/documents/{id}", protected(routes.Documents.Document))
		mux.Handle("GET /api/v1/documents/{id}/content", protected(routes.Documents.Content))
		mux.Handle("GET /api/v1/documents/{id}/download", protected(routes.Documents.Download))
		mux.Handle("GET /api/v1/documents/{id}/versions", protected(routes.Documents.Versions))
		mux.Handle("POST /api/v1/documents/{id}/versions", protected(routes.Documents.Versions))
	}
	if routes.Administration != nil && routes.Authenticator != nil {
		protected := func(handler http.HandlerFunc) http.Handler {
			return platformauth.Middleware(routes.Authenticator, handler)
		}
		mux.Handle("GET /api/v1/workspaces/{workspaceId}/folders", protected(routes.Administration.Folders))
		mux.Handle("POST /api/v1/workspaces/{workspaceId}/folders", protected(routes.Administration.Folders))
		mux.Handle("PATCH /api/v1/folders/{id}", protected(routes.Administration.Folder))
		mux.Handle("DELETE /api/v1/folders/{id}", protected(routes.Administration.Folder))
		mux.Handle("GET /api/v1/organizations/{organizationId}/groups", protected(routes.Administration.Groups))
		mux.Handle("POST /api/v1/organizations/{organizationId}/groups", protected(routes.Administration.Groups))
		mux.Handle("PATCH /api/v1/groups/{id}", protected(routes.Administration.Group))
		mux.Handle("DELETE /api/v1/groups/{id}", protected(routes.Administration.Group))
		mux.Handle("GET /api/v1/groups/{groupId}/members", protected(routes.Administration.Members))
		mux.Handle("POST /api/v1/groups/{groupId}/members", protected(routes.Administration.Members))
		mux.Handle("DELETE /api/v1/groups/{groupId}/members/{userId}", protected(routes.Administration.Member))
		mux.Handle("GET /api/v1/organizations/{organizationId}/grants", protected(routes.Administration.Grants))
		mux.Handle("POST /api/v1/organizations/{organizationId}/grants", protected(routes.Administration.Grants))
		mux.Handle("DELETE /api/v1/grants/{id}", protected(routes.Administration.Grant))
		mux.Handle("GET /api/v1/organizations/{organizationId}/audit", protected(routes.Administration.Audit))
		mux.Handle("GET /api/v1/organizations/{organizationId}/editor-policies", protected(routes.Administration.EditorPolicies))
		mux.Handle("PUT /api/v1/organizations/{organizationId}/editor-policies", protected(routes.Administration.EditorPolicies))
		mux.Handle("GET /api/v1/access/effective", protected(routes.Administration.EffectiveAccess))
	}
	if routes.Organizations != nil && routes.Authenticator != nil {
		protected := func(handler http.HandlerFunc) http.Handler {
			return platformauth.Middleware(routes.Authenticator, handler)
		}
		mux.Handle("GET /api/v1/session-context", protected(routes.Organizations.Session))
		mux.Handle("GET /api/v1/platform/organizations", protected(routes.Organizations.Organizations))
		mux.Handle("POST /api/v1/platform/organizations", protected(routes.Organizations.Organizations))
		mux.Handle("PATCH /api/v1/platform/organizations/{id}", protected(routes.Organizations.Organization))
		mux.Handle("GET /api/v1/platform/organizations/{id}/documents", protected(routes.Organizations.PlatformDocuments))
		mux.Handle("GET /api/v1/platform/dashboard", protected(routes.Organizations.PlatformDashboard))
		mux.Handle("GET /api/v1/organizations/{id}/dashboard", protected(routes.Organizations.OrganizationDashboard))
		mux.Handle("GET /api/v1/organizations/{id}/settings", protected(routes.Organizations.Settings))
		mux.Handle("PATCH /api/v1/organizations/{id}/settings", protected(routes.Organizations.Settings))
		mux.Handle("GET /api/v1/organizations/{id}/members", protected(routes.Organizations.Members))
		mux.Handle("PATCH /api/v1/organizations/{id}/members/{userId}", protected(routes.Organizations.Member))
		mux.Handle("DELETE /api/v1/organizations/{id}/members/{userId}", protected(routes.Organizations.Member))
		mux.Handle("GET /api/v1/organizations/{id}/invitations", protected(routes.Organizations.Invitations))
		mux.Handle("POST /api/v1/organizations/{id}/invitations", protected(routes.Organizations.Invitations))
		mux.Handle("POST /api/v1/invitations/{id}/resend", protected(routes.Organizations.ResendInvitation))
		mux.Handle("DELETE /api/v1/invitations/{id}", protected(routes.Organizations.Invitation))
		mux.Handle("POST /api/v1/invitations/{token}/accept", protected(routes.Organizations.AcceptInvitation))
	}
	if routes.Reporting != nil && routes.Authenticator != nil {
		protected := func(handler http.HandlerFunc) http.Handler {
			return platformauth.Middleware(routes.Authenticator, handler)
		}
		mux.Handle("GET /api/v1/report-definitions", protected(routes.Reporting.Definitions))
		mux.Handle("GET /api/v1/organizations/{id}/report-runs", protected(routes.Reporting.Runs))
		mux.Handle("POST /api/v1/organizations/{id}/report-runs", protected(routes.Reporting.Runs))
		mux.Handle("GET /api/v1/report-runs/{id}", protected(routes.Reporting.Run))
		mux.Handle("POST /api/v1/report-runs/{id}/exports", protected(routes.Reporting.Export))
	}
	if routes.Integrations != nil && routes.Authenticator != nil {
		protected := func(handler http.HandlerFunc) http.Handler {
			return platformauth.Middleware(routes.Authenticator, handler)
		}
		mux.Handle("GET /api/v1/integrations", protected(routes.Integrations.Statuses))
		mux.Handle("PATCH /api/v1/platform/integrations/{id}", protected(routes.Integrations.Installation))
		mux.Handle("GET /api/v1/organizations/{id}/connections", protected(routes.Integrations.Connections))
		mux.Handle("DELETE /api/v1/connections/{id}", protected(routes.Integrations.Connection))
		mux.Handle("POST /api/v1/organizations/{id}/connections/google/oauth/start", protected(routes.Integrations.GoogleOrganizationStart))
		mux.Handle("POST /api/v1/users/me/connections/google/oauth/start", protected(routes.Integrations.GooglePersonalStart))
		mux.Handle("POST /api/v1/connections/{id}/google/picker-session", protected(routes.Integrations.GooglePicker))
		mux.Handle("POST /api/v1/workspaces/{id}/external-documents", protected(routes.Integrations.GoogleDocument))
	}
	if routes.Integrations != nil {
		mux.HandleFunc("GET /api/v1/connections/google/oauth/callback", routes.Integrations.GoogleCallback)
	}
	if routes.WOPI != nil {
		mux.HandleFunc("GET /api/v1/wopi/files/{id}", routes.WOPI.CheckFileInfo)
		mux.HandleFunc("POST /api/v1/wopi/files/{id}", routes.WOPI.FileOperation)
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
