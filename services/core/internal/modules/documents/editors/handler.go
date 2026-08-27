package editors

import (
	"encoding/json"
	"errors"
	"net/http"

	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
)

type Handler struct {
	service        *Service
	sessionService SessionService
}

func NewHandler(service *Service, sessionService SessionService) *Handler {
	return &Handler{service: service, sessionService: sessionService}
}

func (handler *Handler) Providers(writer http.ResponseWriter, request *http.Request) {
	writeEditorJSON(writer, http.StatusOK, map[string]any{"providers": handler.service.Providers(request.Context())})
}

func (handler *Handler) ProviderStatus(writer http.ResponseWriter, request *http.Request) {
	status, err := handler.service.Provider(request.Context(), ProviderID(request.PathValue("provider")))
	if errors.Is(err, ErrProviderNotFound) {
		writeEditorJSON(writer, http.StatusNotFound, map[string]string{"error": "editor_provider_not_found"})
		return
	}
	if err != nil {
		writeEditorJSON(writer, http.StatusInternalServerError, map[string]string{"error": "editor_provider_status_failed"})
		return
	}
	writeEditorJSON(writer, http.StatusOK, status)
}

func (handler *Handler) CreateSession(writer http.ResponseWriter, request *http.Request) {
	var payload struct {
		ProviderID    ProviderID `json:"providerId"`
		RequestedEdit bool       `json:"requestedEdit"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeEditorJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_editor_session_request"})
		return
	}
	if payload.ProviderID == "" || request.PathValue("id") == "" {
		writeEditorJSON(writer, http.StatusBadRequest, map[string]string{"error": "document_and_provider_are_required"})
		return
	}

	subject, ok := platformauth.FromContext(request.Context())
	if !ok {
		writeEditorJSON(writer, http.StatusUnauthorized, map[string]string{"error": "authentication_required"})
		return
	}
	session, err := handler.sessionService.CreateSession(request.Context(), SessionRequest{
		ActorID:       subject.ID,
		DocumentID:    request.PathValue("id"),
		ProviderID:    payload.ProviderID,
		RequestedEdit: payload.RequestedEdit,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionForbidden):
			writeEditorJSON(writer, http.StatusNotFound, map[string]string{"error": "document_not_found"})
		case errors.Is(err, ErrProviderUnavailable), errors.Is(err, ErrOperationDeferred):
			writeEditorJSON(writer, http.StatusServiceUnavailable, map[string]string{"error": "editor_sessions_not_available"})
		default:
			writeEditorJSON(writer, http.StatusBadGateway, map[string]string{"error": "editor_session_failed"})
		}
		return
	}
	writeEditorJSON(writer, http.StatusCreated, session)
}

func writeEditorJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
