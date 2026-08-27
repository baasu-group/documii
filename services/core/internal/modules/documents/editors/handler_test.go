package editors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
)

func TestAvailabilityHandlerDoesNotExposeProviderConfiguration(t *testing.T) {
	registry, err := NewRegistry(NewNativeProvider(), NewStaticProvider(
		ProviderOnlyOffice,
		Capabilities{Operations: []Capability{CapabilityEdit}, Families: []DocumentFamily{FamilyText}},
		Availability{Available: false, Reason: "disabled"},
	))
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(NewService(registry, NewResolver(registry, DefaultPolicy())), UnavailableSessionService{})
	response := httptest.NewRecorder()
	handler.Providers(response, httptest.NewRequest(http.MethodGet, "/api/v1/document-editors", nil))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"native"`) {
		t.Fatalf("Providers() = %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret") || strings.Contains(response.Body.String(), "url") {
		t.Fatalf("Providers() leaked provider configuration: %s", response.Body.String())
	}
}

func TestSessionHandlerFailsClosedWithoutDocumentsService(t *testing.T) {
	registry, _ := NewRegistry(NewNativeProvider())
	handler := NewHandler(NewService(registry, NewResolver(registry, DefaultPolicy())), UnavailableSessionService{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/doc_1/editor-session", strings.NewReader(`{"providerId":"collabora","requestedEdit":true}`))
	request = request.WithContext(platformauth.WithSubject(request.Context(), platformauth.Subject{ID: "user_1"}))
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.CreateSession(response, request)

	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "editor_sessions_not_available") {
		t.Fatalf("CreateSession() = %d %s", response.Code, response.Body.String())
	}
}
