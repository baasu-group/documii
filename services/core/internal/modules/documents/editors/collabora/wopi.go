package collabora

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var (
	ErrForbidden = errors.New("document access forbidden")
	ErrNotFound  = errors.New("document not found")
	ErrConflict  = errors.New("document version conflict")
)

type Access struct {
	CanRead bool
	CanEdit bool
}

type File struct {
	ID          string
	Name        string
	VersionID   string
	Size        int64
	ContentType string
}

type SaveRequest struct {
	FileID          string
	ActorID         string
	ExpectedVersion string
	Content         io.Reader
	ContentLength   int64
}

// DocumentGateway is implemented by the Documents application service. Its
// methods must enforce permissions and append audit events; SaveNewVersion must
// never overwrite an existing business document version.
type DocumentGateway interface {
	Access(context.Context, string, string) (Access, error)
	Info(context.Context, string, string) (File, error)
	OpenContent(context.Context, string, string) (io.ReadCloser, File, error)
	SaveNewVersion(context.Context, SaveRequest) (File, error)
}

type Handler struct {
	tokens         *TokenService
	documents      DocumentGateway
	maxUploadBytes int64
}

func NewHandler(tokens *TokenService, documents DocumentGateway, maxUploadBytes int64) (*Handler, error) {
	if tokens == nil || documents == nil {
		return nil, fmt.Errorf("WOPI token service and document gateway are required")
	}
	if maxUploadBytes <= 0 {
		return nil, fmt.Errorf("WOPI maximum upload size must be positive")
	}
	return &Handler{tokens: tokens, documents: documents, maxUploadBytes: maxUploadBytes}, nil
}

func (handler *Handler) CheckFileInfo(writer http.ResponseWriter, request *http.Request) {
	claims, access, ok := handler.authorize(writer, request, false)
	if !ok {
		return
	}
	file, err := handler.documents.Info(request.Context(), claims.ActorID, claims.FileID)
	if err != nil {
		handler.writeGatewayError(writer, err)
		return
	}
	if !safeFileName(file.Name) {
		writeWOPIJSON(writer, http.StatusInternalServerError, map[string]string{"error": "unsafe_document_filename"})
		return
	}
	writeWOPIJSON(writer, http.StatusOK, map[string]any{
		"BaseFileName":            file.Name,
		"OwnerId":                 "documi",
		"Size":                    file.Size,
		"UserCanWrite":            access.CanEdit && claims.CanEdit,
		"Version":                 file.VersionID,
		"SupportsUpdate":          true,
		"SupportsLocks":           false,
		"UserCanNotWriteRelative": true,
	})
}

func (handler *Handler) GetFile(writer http.ResponseWriter, request *http.Request) {
	claims, _, ok := handler.authorize(writer, request, false)
	if !ok {
		return
	}
	content, file, err := handler.documents.OpenContent(request.Context(), claims.ActorID, claims.FileID)
	if err != nil {
		handler.writeGatewayError(writer, err)
		return
	}
	defer content.Close()
	writer.Header().Set("Content-Type", file.ContentType)
	writer.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	writer.Header().Set("X-WOPI-ItemVersion", file.VersionID)
	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, content)
}

func (handler *Handler) PutFile(writer http.ResponseWriter, request *http.Request) {
	claims, _, ok := handler.authorize(writer, request, true)
	if !ok {
		return
	}
	if request.ContentLength > handler.maxUploadBytes {
		writeWOPIJSON(writer, http.StatusRequestEntityTooLarge, map[string]string{"error": "document_too_large"})
		return
	}
	expectedVersion := claims.VersionID
	if expectedVersion == "" {
		expectedVersion = request.Header.Get("X-Documi-Base-Version")
	}
	if expectedVersion == "" {
		writeWOPIJSON(writer, http.StatusPreconditionRequired, map[string]string{"error": "base_version_required"})
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, handler.maxUploadBytes)
	file, err := handler.documents.SaveNewVersion(request.Context(), SaveRequest{
		FileID:          claims.FileID,
		ActorID:         claims.ActorID,
		ExpectedVersion: expectedVersion,
		Content:         request.Body,
		ContentLength:   request.ContentLength,
	})
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeWOPIJSON(writer, http.StatusRequestEntityTooLarge, map[string]string{"error": "document_too_large"})
			return
		}
		handler.writeGatewayError(writer, err)
		return
	}
	writeWOPIJSON(writer, http.StatusOK, map[string]string{"LastModifiedTime": file.VersionID})
}

func (handler *Handler) authorize(writer http.ResponseWriter, request *http.Request, edit bool) (Claims, Access, bool) {
	token := request.URL.Query().Get("access_token")
	claims, err := handler.tokens.Verify(token)
	if err != nil || claims.FileID != request.PathValue("id") {
		writeWOPIJSON(writer, http.StatusUnauthorized, map[string]string{"error": "invalid_editor_token"})
		return Claims{}, Access{}, false
	}
	access, err := handler.documents.Access(request.Context(), claims.ActorID, claims.FileID)
	if err != nil {
		handler.writeGatewayError(writer, err)
		return Claims{}, Access{}, false
	}
	if !access.CanRead || (edit && (!access.CanEdit || !claims.CanEdit)) {
		writeWOPIJSON(writer, http.StatusForbidden, map[string]string{"error": "document_permission_denied"})
		return Claims{}, Access{}, false
	}
	return claims, access, true
}

func (handler *Handler) writeGatewayError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeWOPIJSON(writer, http.StatusForbidden, map[string]string{"error": "document_permission_denied"})
	case errors.Is(err, ErrNotFound):
		writeWOPIJSON(writer, http.StatusNotFound, map[string]string{"error": "document_not_found"})
	case errors.Is(err, ErrConflict):
		writeWOPIJSON(writer, http.StatusConflict, map[string]string{"error": "document_version_conflict"})
	default:
		writeWOPIJSON(writer, http.StatusInternalServerError, map[string]string{"error": "document_operation_failed"})
	}
}

func safeFileName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00\r\n")
}

func writeWOPIJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
