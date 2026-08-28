package organizations

import (
	"encoding/json"
	"errors"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func subject(r *http.Request) platformauth.Subject {
	v, _ := platformauth.FromContext(r.Context())
	return v
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	if errors.Is(err, ErrForbidden) {
		status = http.StatusForbidden
		code = "forbidden"
	}
	if errors.Is(err, ErrNotFound) {
		status = http.StatusNotFound
		code = "not_found"
	}
	if errors.Is(err, ErrInvalid) {
		status = http.StatusBadRequest
		code = "invalid_request"
	}
	if errors.Is(err, ErrConflict) {
		status = http.StatusConflict
		code = "conflict"
	}
	write(w, status, map[string]string{"error": code})
}
func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.Session(r.Context(), subject(r))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) Organizations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		v, e := h.service.List(r.Context(), subject(r))
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, map[string]any{"organizations": v})
		return
	}
	var q struct {
		Name       string `json:"name"`
		OwnerEmail string `json:"ownerEmail"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		fail(w, ErrInvalid)
		return
	}
	o, i, e := h.service.Create(r.Context(), subject(r), q.Name, q.OwnerEmail, r.Header.Get("Idempotency-Key"))
	if e != nil {
		fail(w, e)
		return
	}
	response := map[string]any{"organization": o}
	if i.ID != "" {
		response["ownerInvitation"] = i
	}
	write(w, 201, response)
}
func (h *Handler) PlatformDocuments(w http.ResponseWriter, r *http.Request) {
	values, err := h.service.PlatformDocuments(r.Context(), subject(r), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, map[string]any{"documents": values})
}
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.AcceptInvitation(r.Context(), subject(r), r.PathValue("token"))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, v)
}
func (h *Handler) Organization(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Status *string `json:"status"`
		Share  *bool   `json:"shareDocumentTitlesWithPlatformAdmins"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		fail(w, ErrInvalid)
		return
	}
	v, e := h.service.Patch(r.Context(), subject(r), r.PathValue("id"), q.Status, q.Share)
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) PlatformDashboard(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.Dashboard(r.Context(), subject(r), "")
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) OrganizationDashboard(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.Dashboard(r.Context(), subject(r), r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	write(w, 200, v)
}
func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		value, err := h.service.Settings(r.Context(), subject(r).ID, r.PathValue("id"))
		if err != nil {
			fail(w, err)
			return
		}
		write(w, 200, value)
		return
	}
	var payload struct {
		Share *bool `json:"shareDocumentTitlesWithPlatformAdmins"`
	}
	if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.Share == nil {
		fail(w, ErrInvalid)
		return
	}
	value, err := h.service.UpdateSettings(r.Context(), subject(r).ID, r.PathValue("id"), *payload.Share)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, value)
}
func (h *Handler) Members(w http.ResponseWriter, r *http.Request) {
	values, err := h.service.Members(r.Context(), subject(r).ID, r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, map[string]any{"members": values})
}
func (h *Handler) Member(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		if err := h.service.RemoveMember(r.Context(), subject(r).ID, r.PathValue("id"), r.PathValue("userId")); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var payload struct {
		Role   *string `json:"role"`
		Status *string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&payload) != nil || (payload.Role == nil && payload.Status == nil) {
		fail(w, ErrInvalid)
		return
	}
	if err := h.service.UpdateMember(r.Context(), subject(r).ID, r.PathValue("id"), r.PathValue("userId"), payload.Role, payload.Status); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Invitations(w http.ResponseWriter, r *http.Request) {
	organizationID := r.PathValue("id")
	if r.Method == http.MethodGet {
		values, err := h.service.Invitations(r.Context(), subject(r).ID, organizationID)
		if err != nil {
			fail(w, err)
			return
		}
		write(w, 200, map[string]any{"invitations": values})
		return
	}
	var payload struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if json.NewDecoder(r.Body).Decode(&payload) != nil {
		fail(w, ErrInvalid)
		return
	}
	value, err := h.service.Invite(r.Context(), subject(r).ID, organizationID, payload.Email, payload.Role, r.Header.Get("Idempotency-Key"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusCreated, value)
}
func (h *Handler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResendInvitation(r.Context(), subject(r).ID, r.PathValue("id"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, value)
}
func (h *Handler) Invitation(w http.ResponseWriter, r *http.Request) {
	if err := h.service.RevokeInvitation(r.Context(), subject(r).ID, r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
