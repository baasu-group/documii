package documents

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (handler *Handler) Workspaces(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	values, err := handler.service.Workspaces(request.Context(), subject.ID)
	if err != nil {
		handler.error(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"workspaces": values})
}

func (handler *Handler) Documents(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	workspaceID := request.PathValue("workspaceId")
	if request.Method == http.MethodGet {
		values, err := handler.service.List(request.Context(), subject.ID, workspaceID)
		if err != nil {
			handler.error(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"documents": values})
		return
	}
	var payload struct {
		Title    string          `json:"title"`
		FolderID *string         `json:"folderId"`
		Family   string          `json:"family"`
		Content  json.RawMessage `json:"content"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, MaxNativeBytes+16<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_document_request"})
		return
	}
	var value Document
	var err error
	if payload.Family == "native" {
		value, err = handler.service.CreateNative(request.Context(), CreateRequest{ActorID: subject.ID, WorkspaceID: workspaceID, FolderID: payload.FolderID, Title: payload.Title, Content: payload.Content, IdempotencyKey: request.Header.Get("Idempotency-Key")})
	} else if payload.Family == "text" || payload.Family == "spreadsheet" || payload.Family == "presentation" {
		if len(payload.Content) != 0 && string(payload.Content) != "null" {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_document_request"})
			return
		}
		value, err = handler.service.CreateOffice(request.Context(), CreateOfficeRequest{ActorID: subject.ID, WorkspaceID: workspaceID, FolderID: payload.FolderID, Title: payload.Title, Family: payload.Family, IdempotencyKey: request.Header.Get("Idempotency-Key")})
	} else {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_document_request"})
		return
	}
	if err != nil {
		handler.error(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (handler *Handler) Upload(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	request.Body = http.MaxBytesReader(writer, request.Body, MaxDocumentBytes+(1<<20))
	if err := request.ParseMultipartForm(MaxDocumentBytes); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_document_upload"})
		return
	}
	defer request.MultipartForm.RemoveAll()
	file, header, err := request.FormFile("file")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "document_file_required"})
		return
	}
	defer file.Close()
	var folderID *string
	if value := request.FormValue("folderId"); value != "" {
		folderID = &value
	}
	value, err := handler.service.UploadOffice(request.Context(), UploadRequest{ActorID: subject.ID, WorkspaceID: request.PathValue("workspaceId"), FolderID: folderID, Filename: header.Filename, MIMEType: header.Header.Get("Content-Type"), Content: file, IdempotencyKey: request.Header.Get("Idempotency-Key")})
	if err != nil {
		handler.error(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (handler *Handler) Document(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	value, err := handler.service.Get(request.Context(), subject.ID, request.PathValue("id"), true)
	if err != nil {
		handler.error(writer, err)
		return
	}
	if err := handler.service.RecordAudit(request.Context(), subject.ID, value.ID, "document.editor.opened"); err != nil {
		handler.error(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}

func (handler *Handler) Content(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	value, err := handler.service.Get(request.Context(), subject.ID, request.PathValue("id"), true)
	if err != nil {
		handler.error(writer, err)
		return
	}
	writer.Header().Set("Content-Type", value.MIMEType)
	writer.Header().Set("ETag", `"`+value.CurrentVersionID+`"`)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(value.Content)
}

func (handler *Handler) Download(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	value, reader, err := handler.service.Download(request.Context(), subject.ID, request.PathValue("id"))
	if err != nil {
		handler.error(writer, err)
		return
	}
	defer reader.Close()
	writer.Header().Set("Content-Type", value.MIMEType)
	writer.Header().Set("Content-Disposition", `attachment; filename="document"`)
	writer.Header().Set("X-Documi-Version", value.CurrentVersionID)
	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, reader)
}

func (handler *Handler) Versions(writer http.ResponseWriter, request *http.Request) {
	subject, _ := platformauth.FromContext(request.Context())
	id := request.PathValue("id")
	if request.Method == http.MethodGet {
		values, err := handler.service.Versions(request.Context(), subject.ID, id)
		if err != nil {
			handler.error(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"versions": values})
		return
	}
	var payload struct {
		BaseVersionID string          `json:"baseVersionId"`
		Content       json.RawMessage `json:"content"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, MaxNativeBytes+16<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_document_version_request"})
		return
	}
	value, err := handler.service.SaveNative(request.Context(), SaveRequest{ActorID: subject.ID, DocumentID: id, BaseVersionID: payload.BaseVersionID, Content: payload.Content, IdempotencyKey: request.Header.Get("Idempotency-Key")})
	if err != nil {
		handler.error(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}

func (handler *Handler) error(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden):
		writeJSON(writer, http.StatusNotFound, map[string]string{"error": "document_not_found"})
	case errors.Is(err, ErrVersionConflict):
		writeJSON(writer, http.StatusConflict, map[string]string{"error": "document_version_conflict"})
	case errors.Is(err, ErrLocked):
		writeJSON(writer, http.StatusLocked, map[string]string{"error": "document_locked"})
	case errors.Is(err, ErrIdempotencyNeeded):
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "idempotency_key_required"})
	case errors.Is(err, ErrInvalidDocument):
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_document"})
	default:
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "document_operation_failed"})
	}
}
func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
