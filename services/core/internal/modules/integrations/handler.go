package integrations

import (
	"encoding/json"
	"errors"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"io"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	if errors.Is(err, ErrForbidden) {
		status, code = http.StatusForbidden, "forbidden"
	}
	if errors.Is(err, ErrNotFound) {
		status, code = http.StatusNotFound, "not_found"
	}
	if errors.Is(err, ErrInvalid) {
		status, code = http.StatusBadRequest, "invalid_request"
	}
	if errors.Is(err, ErrAirGap) {
		status, code = http.StatusConflict, "blocked_by_air_gap"
	}
	write(w, status, map[string]string{"error": code})
}
func subject(r *http.Request) platformauth.Subject {
	value, _ := platformauth.FromContext(r.Context())
	return value
}
func (h *Handler) Statuses(w http.ResponseWriter, r *http.Request) {
	values, err := h.service.Statuses(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, map[string]any{"integrations": values})
}
func (h *Handler) Installation(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Enabled *bool `json:"enabled"`
	}
	if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.Enabled == nil {
		fail(w, ErrInvalid)
		return
	}
	value, err := h.service.Update(r.Context(), subject(r), r.PathValue("id"), *payload.Enabled)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, value)
}
func (h *Handler) Connections(w http.ResponseWriter, r *http.Request) {
	values, err := h.service.Connections(r.Context(), subject(r).ID, r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, map[string]any{"connections": values})
}
func (h *Handler) Connection(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteConnection(r.Context(), subject(r).ID, r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) GoogleOrganizationStart(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("id")
	value, err := h.service.GoogleStart(r.Context(), subject(r), &org)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, value)
}
func (h *Handler) GooglePersonalStart(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.GoogleStart(r.Context(), subject(r), nil)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, value)
}
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	_, redirect, err := h.service.GoogleCallback(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("code"))
	if err != nil {
		fail(w, err)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
func (h *Handler) GooglePicker(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.GooglePicker(r.Context(), subject(r), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, value)
}

func (h *Handler) GoogleDocument(w http.ResponseWriter, r *http.Request) {
	var payload GoogleDocumentRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		fail(w, ErrInvalid)
		return
	}
	payload.WorkspaceID = r.PathValue("id")
	payload.IdempotencyKey = r.Header.Get("Idempotency-Key")
	value, err := h.service.GoogleDocument(r.Context(), subject(r), payload, payload.ExternalFileID == "")
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusCreated, value)
}
