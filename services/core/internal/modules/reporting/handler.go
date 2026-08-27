package reporting

import (
	"encoding/json"
	"errors"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"io"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func reportError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	if errors.Is(err, ErrForbidden) {
		status, code = http.StatusForbidden, "forbidden"
	}
	if errors.Is(err, ErrInvalid) {
		status, code = http.StatusBadRequest, "invalid_report_request"
	}
	if errors.Is(err, ErrNotFound) {
		status, code = http.StatusNotFound, "report_not_found"
	}
	writeJSON(w, status, map[string]string{"error": code})
}
func actor(r *http.Request) string {
	subject, _ := platformauth.FromContext(r.Context())
	return subject.ID
}
func (h *Handler) Definitions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"definitions": h.service.Definitions()})
}
func (h *Handler) Runs(w http.ResponseWriter, r *http.Request) {
	organizationID := r.PathValue("id")
	if r.Method == http.MethodGet {
		values, err := h.service.List(r.Context(), actor(r), organizationID)
		if err != nil {
			reportError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"runs": values})
		return
	}
	var payload struct {
		DefinitionID string         `json:"definitionId"`
		Format       string         `json:"format"`
		Filters      map[string]any `json:"filters"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		reportError(w, ErrInvalid)
		return
	}
	if payload.Filters == nil {
		payload.Filters = map[string]any{}
	}
	value, err := h.service.Create(r.Context(), actor(r), organizationID, payload.DefinitionID, payload.Format, r.Header.Get("Idempotency-Key"), payload.Filters)
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (h *Handler) Run(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Get(r.Context(), actor(r), r.PathValue("id"))
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, 200, value)
}
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Export(r.Context(), actor(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		reportError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
