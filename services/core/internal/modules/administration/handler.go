package administration

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (handler *Handler) Folders(writer http.ResponseWriter, request *http.Request) {
	actor := actorID(request)
	workspaceID := request.PathValue("workspaceId")
	if request.Method == http.MethodGet {
		values, err := handler.service.Folders(request.Context(), actor, workspaceID)
		if err != nil {
			handler.writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"folders": values})
		return
	}
	var payload struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parentId"`
	}
	if !decodeJSON(writer, request, &payload) {
		return
	}
	value, err := handler.service.CreateFolder(request.Context(), actor, workspaceID, payload.Name, payload.ParentID)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (handler *Handler) Folder(writer http.ResponseWriter, request *http.Request) {
	actor := actorID(request)
	if request.Method == http.MethodDelete {
		if err := handler.service.DeleteFolder(request.Context(), actor, request.PathValue("id")); err != nil {
			handler.writeError(writer, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	var payload struct {
		Name     *string         `json:"name"`
		ParentID json.RawMessage `json:"parentId"`
	}
	if !decodeJSON(writer, request, &payload) {
		return
	}
	var parentID *string
	parentSpecified := payload.ParentID != nil
	if parentSpecified && string(payload.ParentID) != "null" {
		if json.Unmarshal(payload.ParentID, &parentID) != nil || parentID == nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_administration_request"})
			return
		}
	}
	value, err := handler.service.UpdateFolder(request.Context(), actor, request.PathValue("id"), payload.Name, parentSpecified, parentID)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}

func (handler *Handler) Groups(writer http.ResponseWriter, request *http.Request) {
	actor := actorID(request)
	organizationID := request.PathValue("organizationId")
	if request.Method == http.MethodGet {
		values, err := handler.service.Groups(request.Context(), actor, organizationID)
		if err != nil {
			handler.writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"groups": values})
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if !decodeJSON(writer, request, &payload) {
		return
	}
	value, err := handler.service.CreateGroup(request.Context(), actor, organizationID, payload.Name)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (handler *Handler) Group(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodPatch {
		var payload struct {
			Name string `json:"name"`
		}
		if !decodeJSON(writer, request, &payload) {
			return
		}
		value, err := handler.service.UpdateGroup(request.Context(), actorID(request), request.PathValue("id"), payload.Name)
		if err != nil {
			handler.writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, value)
		return
	}
	if err := handler.service.DeleteGroup(request.Context(), actorID(request), request.PathValue("id")); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) Members(writer http.ResponseWriter, request *http.Request) {
	actor := actorID(request)
	groupID := request.PathValue("groupId")
	if request.Method == http.MethodGet {
		values, err := handler.service.Members(request.Context(), actor, groupID)
		if err != nil {
			handler.writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"members": values})
		return
	}
	var payload struct {
		UserID string `json:"userId"`
	}
	if !decodeJSON(writer, request, &payload) {
		return
	}
	if err := handler.service.AddMember(request.Context(), actor, groupID, payload.UserID); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) Member(writer http.ResponseWriter, request *http.Request) {
	if err := handler.service.RemoveMember(request.Context(), actorID(request), request.PathValue("groupId"), request.PathValue("userId")); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) Grants(writer http.ResponseWriter, request *http.Request) {
	actor := actorID(request)
	organizationID := request.PathValue("organizationId")
	if request.Method == http.MethodGet {
		values, err := handler.service.Grants(request.Context(), actor, organizationID)
		if err != nil {
			handler.writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"grants": values})
		return
	}
	var payload CreateGrantRequest
	if !decodeJSON(writer, request, &payload) {
		return
	}
	value, err := handler.service.CreateGrant(request.Context(), actor, organizationID, payload)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (handler *Handler) Grant(writer http.ResponseWriter, request *http.Request) {
	if err := handler.service.DeleteGrant(request.Context(), actorID(request), request.PathValue("id")); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) Audit(writer http.ResponseWriter, request *http.Request) {
	limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
	if request.URL.Query().Get("limit") == "" {
		limit = 50
	} else if err != nil {
		handler.writeError(writer, ErrInvalidRequest)
		return
	}
	page, err := handler.service.Audit(request.Context(), actorID(request), request.PathValue("organizationId"), request.URL.Query().Get("workspaceId"), request.URL.Query().Get("eventType"), request.URL.Query().Get("cursor"), limit)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (handler *Handler) EditorPolicies(writer http.ResponseWriter, request *http.Request) {
	actor := actorID(request)
	organizationID := request.PathValue("organizationId")
	if request.Method == http.MethodGet {
		values, err := handler.service.EditorPolicies(request.Context(), actor, organizationID)
		if err != nil {
			handler.writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"policies": values})
		return
	}
	var payload struct {
		Family     string `json:"family"`
		ProviderID string `json:"providerId"`
	}
	if !decodeJSON(writer, request, &payload) {
		return
	}
	value, err := handler.service.SetEditorPolicy(request.Context(), actor, organizationID, payload.Family, payload.ProviderID)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}

func (handler *Handler) EffectiveAccess(writer http.ResponseWriter, request *http.Request) {
	values, err := handler.service.EffectiveAccess(request.Context(), actorID(request), request.URL.Query().Get("subjectId"), request.URL.Query().Get("scopeType"), request.URL.Query().Get("scopeId"))
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"capabilities": values})
}

func actorID(request *http.Request) string {
	subject, _ := platformauth.FromContext(request.Context())
	return subject.ID
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 32<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(destination) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_administration_request"})
		return false
	}
	return true
}

func (handler *Handler) writeError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden):
		writeJSON(writer, http.StatusNotFound, map[string]string{"error": "administration_resource_not_found"})
	case errors.Is(err, ErrInvalidRequest):
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_administration_request"})
	case errors.Is(err, ErrConflict):
		writeJSON(writer, http.StatusConflict, map[string]string{"error": "administration_conflict"})
	default:
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "administration_operation_failed"})
	}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
