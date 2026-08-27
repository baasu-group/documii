package documents

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/documi/documi/services/core/internal/modules/documents/templates"
	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UploadRequest struct {
	ActorID, WorkspaceID, IdempotencyKey, Filename, MIMEType string
	FolderID                                                 *string
	Content                                                  io.Reader
	titleOverride, operation, auditEvent                     string
}

type CreateOfficeRequest struct {
	ActorID, WorkspaceID, IdempotencyKey, Title, Family string
	FolderID                                            *string
}

func (service *Service) CreateOffice(ctx context.Context, request CreateOfficeRequest) (Document, error) {
	title := strings.TrimSpace(request.Title)
	if title == "" || utf8.RuneCountInString(title) > 255 {
		return Document{}, ErrInvalidDocument
	}
	template, err := templates.Get(request.Family)
	if err != nil {
		return Document{}, ErrInvalidDocument
	}
	return service.UploadOffice(ctx, UploadRequest{
		ActorID: request.ActorID, WorkspaceID: request.WorkspaceID, FolderID: request.FolderID,
		IdempotencyKey: request.IdempotencyKey, Filename: "document" + template.Extension,
		MIMEType: template.MIMEType, Content: bytes.NewReader(template.Content), titleOverride: title,
		operation: "document.office.create", auditEvent: "document.created",
	})
}

type officeFormat struct{ Family, MIMEType, ODFMIME string }

var officeFormats = map[string]officeFormat{
	".docx": {"text", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ""},
	".xlsx": {"spreadsheet", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ""},
	".pptx": {"presentation", "application/vnd.openxmlformats-officedocument.presentationml.presentation", ""},
	".odt":  {"text", "application/vnd.oasis.opendocument.text", "application/vnd.oasis.opendocument.text"},
	".ods":  {"spreadsheet", "application/vnd.oasis.opendocument.spreadsheet", "application/vnd.oasis.opendocument.spreadsheet"},
	".odp":  {"presentation", "application/vnd.oasis.opendocument.presentation", "application/vnd.oasis.opendocument.presentation"},
}

func (service *Service) UploadOffice(ctx context.Context, request UploadRequest) (Document, error) {
	if !validIdempotencyKey(request.IdempotencyKey) {
		return Document{}, ErrIdempotencyNeeded
	}
	if request.Filename == "" || strings.ContainsAny(request.Filename, "/\\\x00\r\n") || request.Filename != filepath.Base(request.Filename) {
		return Document{}, ErrInvalidDocument
	}
	extension := strings.ToLower(filepath.Ext(request.Filename))
	format, ok := officeFormats[extension]
	if !ok || request.MIMEType != format.MIMEType {
		return Document{}, ErrInvalidDocument
	}
	title := strings.TrimSpace(strings.TrimSuffix(request.Filename, filepath.Ext(request.Filename)))
	if request.titleOverride != "" {
		title = request.titleOverride
	}
	if title == "" || utf8.RuneCountInString(title) > 255 {
		return Document{}, ErrInvalidDocument
	}
	body, err := io.ReadAll(io.LimitReader(request.Content, MaxDocumentBytes+1))
	if err != nil || int64(len(body)) > MaxDocumentBytes {
		return Document{}, ErrInvalidDocument
	}
	if err := validateOfficeArchive(body, format); err != nil {
		return Document{}, err
	}
	actor, err := uuid.Parse(request.ActorID)
	if err != nil {
		return Document{}, ErrForbidden
	}
	workspaceID, err := uuid.Parse(request.WorkspaceID)
	if err != nil {
		return Document{}, ErrNotFound
	}
	var organizationID uuid.UUID
	if err = service.pool.QueryRow(ctx, "SELECT organization_id FROM workspaces WHERE id=$1", workspaceID).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	} else if err != nil {
		return Document{}, err
	}
	var folderID *uuid.UUID
	if request.FolderID != nil {
		parsed, err := uuid.Parse(*request.FolderID)
		if err != nil {
			return Document{}, ErrNotFound
		}
		folderID = &parsed
	}
	allowed, err := service.authorizer.Allows(ctx, request.ActorID, permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID, FolderID: folderID}, permissions.CanCreate)
	if err != nil {
		return Document{}, err
	}
	if !allowed {
		return Document{}, ErrForbidden
	}
	operation := request.operation
	if operation == "" {
		operation = "document.upload"
	}
	auditEvent := request.auditEvent
	if auditEvent == "" {
		auditEvent = "document.imported"
	}
	if existing, found, err := service.idempotentDocument(ctx, organizationID, actor, request.IdempotencyKey, operation); err != nil {
		return Document{}, err
	} else if found {
		return service.Get(ctx, request.ActorID, existing.String(), false)
	}
	documentID, versionID, objectID := uuid.New(), uuid.New(), uuid.New()
	if service.editors == nil {
		return Document{}, ErrInvalidDocument
	}
	providerID, err := service.editors.Select(ctx, organizationID.String(), format.Family)
	if err != nil {
		return Document{}, err
	}
	key := path.Join("documents", organizationID.String(), workspaceID.String(), documentID.String(), versionID.String(), "document"+extension)
	object, err := service.storage.Put(ctx, key, bytes.NewReader(body))
	if err != nil {
		return Document{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = service.storage.Delete(context.Background(), key)
		}
	}()
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO documents(id,organization_id,workspace_id,folder_id,title,family,mime_type,editor_provider,storage_mode,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'managed',$9)`, documentID, organizationID, workspaceID, folderID, title, format.Family, format.MIMEType, providerID, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO storage_objects(id,provider,storage_key,sha256,size,content_type) VALUES($1,'filesystem',$2,$3,$4,$5)", objectID, key, object.SHA256, object.Size, format.MIMEType); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO document_versions(id,document_id,storage_object_id,version_number,created_by) VALUES($1,$2,$3,1,$4)", versionID, documentID, objectID, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "UPDATE documents SET current_version_id=$2 WHERE id=$1", documentID, versionID); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,$4,$5)", organizationID, actor, request.IdempotencyKey, operation, documentID); err != nil {
		return Document{}, err
	}
	if err = appendAudit(ctx, tx, auditEvent, organizationID, workspaceID, actor, documentID); err != nil {
		return Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	committed = true
	return service.Get(ctx, request.ActorID, documentID.String(), false)
}

func validateOfficeArchive(content []byte, format officeFormat) error {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return ErrInvalidDocument
	}
	foundTypes := false
	foundODF := format.ODFMIME == ""
	var uncompressed uint64
	if len(reader.File) > 10_000 {
		return ErrInvalidDocument
	}
	for _, file := range reader.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		clean := path.Clean(name)
		if name == "" || strings.HasPrefix(name, "/") || clean == ".." || strings.HasPrefix(clean, "../") || clean != strings.TrimSuffix(name, "/") {
			return ErrInvalidDocument
		}
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "vbaproject") || strings.Contains(lowerName, "/activex/") || strings.Contains(lowerName, "/embeddings/") {
			return ErrInvalidDocument
		}
		archiveLimit := uint64(MaxDocumentBytes * 5)
		if file.UncompressedSize64 > archiveLimit || uncompressed > archiveLimit-file.UncompressedSize64 {
			return ErrInvalidDocument
		}
		uncompressed += file.UncompressedSize64
		if file.Name == "[Content_Types].xml" {
			foundTypes = true
			stream, openErr := file.Open()
			if openErr != nil {
				return ErrInvalidDocument
			}
			value, readErr := io.ReadAll(io.LimitReader(stream, (1<<20)+1))
			stream.Close()
			lowerTypes := strings.ToLower(string(value))
			if readErr != nil || len(value) > 1<<20 || strings.Contains(lowerTypes, "macroenabled") || strings.Contains(lowerTypes, "vbaproject") {
				return ErrInvalidDocument
			}
		}
		if file.Name == "mimetype" && format.ODFMIME != "" {
			stream, err := file.Open()
			if err != nil {
				return ErrInvalidDocument
			}
			value, err := io.ReadAll(io.LimitReader(stream, 256))
			stream.Close()
			if err == nil && string(value) == format.ODFMIME {
				foundODF = true
			}
		}
	}
	if format.ODFMIME == "" && !foundTypes {
		return ErrInvalidDocument
	}
	if !foundODF {
		return ErrInvalidDocument
	}
	return nil
}
