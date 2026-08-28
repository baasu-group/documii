package documents

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/documi/documi/services/core/internal/modules/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	NativeMIME       = "application/vnd.documi.document+json"
	NativeFormat     = "documi-json-v1"
	MaxNativeBytes   = 5 << 20
	MaxDocumentBytes = 100 << 20
)

var (
	ErrNotFound          = errors.New("document not found")
	ErrForbidden         = errors.New("document access forbidden")
	ErrVersionConflict   = errors.New("document version conflict")
	ErrLocked            = errors.New("document locked")
	ErrInvalidDocument   = errors.New("invalid document")
	ErrIdempotencyNeeded = errors.New("idempotency key required")
)

type Document struct {
	ID                   string          `json:"id"`
	WorkspaceID          string          `json:"workspaceId"`
	FolderID             *string         `json:"folderId,omitempty"`
	Title                string          `json:"title"`
	Family               string          `json:"family"`
	MIMEType             string          `json:"mimeType"`
	EditorProvider       string          `json:"editorProvider"`
	StorageMode          string          `json:"storageMode"`
	CurrentVersionID     string          `json:"currentVersionId"`
	ExternalProvider     *string         `json:"externalProvider,omitempty"`
	ExternalFileID       *string         `json:"externalFileId,omitempty"`
	ExternalURL          *string         `json:"externalUrl,omitempty"`
	ExternalConnectionID *string         `json:"externalConnectionId,omitempty"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
	Content              json.RawMessage `json:"content,omitempty"`
	organizationID       uuid.UUID
	storageKey           string
	size                 int64
	sha256               string
}

func (document Document) Size() int64 { return document.size }

type Version struct {
	ID              string    `json:"id"`
	DocumentID      string    `json:"documentId"`
	ParentVersionID *string   `json:"parentVersionId,omitempty"`
	VersionNumber   int64     `json:"versionNumber"`
	SHA256          string    `json:"sha256"`
	Size            int64     `json:"size"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Workspace struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Mode           string `json:"mode"`
}

type CreateRequest struct {
	ActorID, WorkspaceID, IdempotencyKey, Title string
	FolderID                                    *string
	Content                                     json.RawMessage
}

type SaveRequest struct {
	ActorID, DocumentID, BaseVersionID, IdempotencyKey string
	Content                                            json.RawMessage
}

type CreateExternalRequest struct {
	ActorID, WorkspaceID, IdempotencyKey, Title string
	FolderID                                    *string
	Family, MIMEType, Provider, ExternalFileID  string
	ExternalURL                                 string
	ExternalConnectionID                        string
}

type Service struct {
	pool       *pgxpool.Pool
	storage    storage.Provider
	authorizer *permissions.Authorizer
	editors    EditorSelector
}

type EditorSelector interface {
	Select(context.Context, string, string) (string, error)
}

type EditorSelectorFunc func(context.Context, string, string) (string, error)

func (function EditorSelectorFunc) Select(ctx context.Context, organizationID, family string) (string, error) {
	return function(ctx, organizationID, family)
}

func NewService(pool *pgxpool.Pool, provider storage.Provider, authorizer *permissions.Authorizer) *Service {
	return &Service{pool: pool, storage: provider, authorizer: authorizer}
}

func (service *Service) SetEditorSelector(selector EditorSelector) { service.editors = selector }

func (service *Service) Workspaces(ctx context.Context, actorID string) ([]Workspace, error) {
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return nil, ErrForbidden
	}
	rows, err := service.pool.Query(ctx, `SELECT DISTINCT w.id,w.organization_id,w.name FROM workspaces w
JOIN authorization_grants g ON g.organization_id=w.organization_id
JOIN organizations o ON o.id=w.organization_id AND o.status='active'
JOIN organization_memberships m ON m.organization_id=w.organization_id AND m.user_id=$1 AND m.status='active'
WHERE (g.user_id=$1 OR g.group_id IN (SELECT group_id FROM group_members WHERE user_id=$1))
AND (g.scope_type='organization' OR g.workspace_id=w.id)
ORDER BY w.name,w.id`, actor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Workspace{}
	for rows.Next() {
		var value Workspace
		value.Mode = "server"
		if err := rows.Scan(&value.ID, &value.OrganizationID, &value.Name); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) List(ctx context.Context, actorID, workspaceID string) ([]Document, error) {
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return nil, ErrForbidden
	}
	workspace, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := service.pool.Query(ctx, `
WITH RECURSIVE ancestry(document_id,folder_id,parent_id) AS (
 SELECT d.id,f.id,f.parent_id FROM documents d LEFT JOIN folders f ON f.id=d.folder_id WHERE d.workspace_id=$2
 UNION ALL SELECT a.document_id,f.id,f.parent_id FROM ancestry a JOIN folders f ON f.id=a.parent_id
)
SELECT d.id,d.workspace_id,d.folder_id,d.title,d.family,d.mime_type,d.editor_provider,d.storage_mode,
       COALESCE(d.current_version_id::text,''),d.created_at,d.updated_at,d.organization_id,
       d.external_provider,d.external_file_id,d.external_url
FROM documents d WHERE d.workspace_id=$2 AND EXISTS (
 SELECT 1 FROM authorization_grants g
 JOIN organizations o ON o.id=g.organization_id AND o.status='active'
 JOIN organization_memberships m ON m.organization_id=g.organization_id AND m.user_id=$1 AND m.status='active'
 WHERE g.organization_id=d.organization_id
 AND (g.user_id=$1 OR g.group_id IN (SELECT group_id FROM group_members WHERE user_id=$1))
 AND (g.scope_type='organization' OR (g.scope_type='workspace' AND g.workspace_id=d.workspace_id)
      OR (g.scope_type='document' AND g.document_id=d.id)
      OR (g.scope_type='folder' AND g.folder_id IN (SELECT folder_id FROM ancestry WHERE document_id=d.id)))
) ORDER BY d.updated_at DESC,d.id`, actor, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Document{}
	for rows.Next() {
		value, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) CreateNative(ctx context.Context, request CreateRequest) (Document, error) {
	if !validIdempotencyKey(request.IdempotencyKey) {
		return Document{}, ErrIdempotencyNeeded
	}
	if strings.TrimSpace(request.Title) == "" || utf8.RuneCountInString(request.Title) > 255 {
		return Document{}, ErrInvalidDocument
	}
	content, err := validateNative(request.Content)
	if err != nil {
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
	if err := service.pool.QueryRow(ctx, "SELECT organization_id FROM workspaces WHERE id=$1", workspaceID).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) {
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
	if existing, ok, err := service.idempotentDocument(ctx, organizationID, actor, request.IdempotencyKey, "document.create"); err != nil {
		return Document{}, err
	} else if ok {
		return service.Get(ctx, request.ActorID, existing.String(), true)
	}
	documentID, versionID, objectID := uuid.New(), uuid.New(), uuid.New()
	key := path.Join("documents", organizationID.String(), workspaceID.String(), documentID.String(), versionID.String(), "document.documi.json")
	object, err := service.storage.Put(ctx, key, bytes.NewReader(content))
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
	if _, err = tx.Exec(ctx, `INSERT INTO documents(id,organization_id,workspace_id,folder_id,title,family,mime_type,editor_provider,storage_mode,native_format,created_by)
VALUES($1,$2,$3,$4,$5,'native',$6,'native','managed',$7,$8)`, documentID, organizationID, workspaceID, folderID, strings.TrimSpace(request.Title), NativeMIME, NativeFormat, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO storage_objects(id,provider,storage_key,sha256,size,content_type) VALUES($1,'filesystem',$2,$3,$4,$5)", objectID, key, object.SHA256, object.Size, NativeMIME); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO document_versions(id,document_id,storage_object_id,version_number,created_by) VALUES($1,$2,$3,1,$4)", versionID, documentID, objectID, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "UPDATE documents SET current_version_id=$2 WHERE id=$1", documentID, versionID); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,'document.create',$4)", organizationID, actor, request.IdempotencyKey, documentID); err != nil {
		return Document{}, err
	}
	if err = appendAudit(ctx, tx, "document.created", organizationID, workspaceID, actor, documentID); err != nil {
		return Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	committed = true
	return service.Get(ctx, request.ActorID, documentID.String(), true)
}

func (service *Service) CreateExternal(ctx context.Context, request CreateExternalRequest) (Document, error) {
	if !validIdempotencyKey(request.IdempotencyKey) {
		return Document{}, ErrIdempotencyNeeded
	}
	title := strings.TrimSpace(request.Title)
	if title == "" || utf8.RuneCountInString(title) > 255 || request.ExternalFileID == "" {
		return Document{}, ErrInvalidDocument
	}
	parsedURL, err := url.ParseRequestURI(request.ExternalURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return Document{}, ErrInvalidDocument
	}
	validProvider := (request.Provider == "google-docs" && request.Family == "text") ||
		(request.Provider == "google-sheets" && request.Family == "spreadsheet") ||
		(request.Provider == "google-slides" && request.Family == "presentation")
	if !validProvider {
		return Document{}, ErrInvalidDocument
	}
	connectionID, err := uuid.Parse(request.ExternalConnectionID)
	if err != nil {
		return Document{}, ErrInvalidDocument
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
		value, parseErr := uuid.Parse(*request.FolderID)
		if parseErr != nil {
			return Document{}, ErrNotFound
		}
		folderID = &value
	}
	allowed, err := service.authorizer.Allows(ctx, request.ActorID, permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID, FolderID: folderID}, permissions.CanCreate)
	if err != nil || !allowed {
		if err != nil {
			return Document{}, err
		}
		return Document{}, ErrForbidden
	}
	if existing, ok, err := service.idempotentDocument(ctx, organizationID, actor, request.IdempotencyKey, "document.external.create"); err != nil {
		return Document{}, err
	} else if ok {
		return service.Get(ctx, request.ActorID, existing.String(), false)
	}
	documentID := uuid.New()
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO documents(id,organization_id,workspace_id,folder_id,title,family,mime_type,editor_provider,storage_mode,external_provider,external_file_id,external_url,external_connection_id,created_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,'external','google-workspace',$9,$10,$11,$12)`, documentID, organizationID, workspaceID, folderID, title, request.Family, request.MIMEType, request.Provider, request.ExternalFileID, request.ExternalURL, connectionID, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,'document.external.create',$4)", organizationID, actor, request.IdempotencyKey, documentID); err != nil {
		return Document{}, err
	}
	for _, event := range []string{"document.external.created", "document.google.created"} {
		if err = appendAudit(ctx, tx, event, organizationID, workspaceID, actor, documentID); err != nil {
			return Document{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return service.Get(ctx, request.ActorID, documentID.String(), false)
}

func (service *Service) Get(ctx context.Context, actorID, documentID string, includeContent bool) (Document, error) {
	id, err := uuid.Parse(documentID)
	if err != nil {
		return Document{}, ErrNotFound
	}
	row := service.pool.QueryRow(ctx, `SELECT d.id,d.workspace_id,d.folder_id,d.title,d.family,d.mime_type,d.editor_provider,d.storage_mode,
COALESCE(d.current_version_id::text,''),d.created_at,d.updated_at,d.organization_id,
COALESCE(o.storage_key,''),COALESCE(o.size,0),COALESCE(o.sha256,''),d.external_provider,d.external_file_id,d.external_url
FROM documents d LEFT JOIN document_versions v ON v.id=d.current_version_id LEFT JOIN storage_objects o ON o.id=v.storage_object_id WHERE d.id=$1`, id)
	value, err := scanDocumentWithStorage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	} else if err != nil {
		return Document{}, err
	}
	workspace := uuid.MustParse(value.WorkspaceID)
	var folder *uuid.UUID
	if value.FolderID != nil {
		v := uuid.MustParse(*value.FolderID)
		folder = &v
	}
	allowed, err := service.authorizer.Allows(ctx, actorID, permissions.Resource{OrganizationID: value.organizationID, WorkspaceID: workspace, FolderID: folder, DocumentID: &id}, permissions.CanRead)
	if err != nil {
		return Document{}, err
	}
	if !allowed {
		return Document{}, ErrNotFound
	}
	if includeContent && value.MIMEType == NativeMIME {
		body, err := service.verifiedContent(ctx, value)
		if err != nil {
			return Document{}, err
		}
		value.Content = body
	}
	return value, nil
}

func (service *Service) SaveNative(ctx context.Context, request SaveRequest) (Document, error) {
	if !validIdempotencyKey(request.IdempotencyKey) {
		return Document{}, ErrIdempotencyNeeded
	}
	content, err := validateNative(request.Content)
	if err != nil {
		return Document{}, err
	}
	current, err := service.Get(ctx, request.ActorID, request.DocumentID, false)
	if err != nil {
		return Document{}, err
	}
	documentID := uuid.MustParse(current.ID)
	workspaceID := uuid.MustParse(current.WorkspaceID)
	actor, err := uuid.Parse(request.ActorID)
	if err != nil {
		return Document{}, ErrForbidden
	}
	var folderID *uuid.UUID
	if current.FolderID != nil {
		v := uuid.MustParse(*current.FolderID)
		folderID = &v
	}
	allowed, err := service.authorizer.Allows(ctx, request.ActorID, permissions.Resource{OrganizationID: current.organizationID, WorkspaceID: workspaceID, FolderID: folderID, DocumentID: &documentID}, permissions.CanEdit)
	if err != nil {
		return Document{}, err
	}
	if !allowed {
		return Document{}, ErrForbidden
	}
	if existing, ok, err := service.idempotentDocument(ctx, current.organizationID, actor, request.IdempotencyKey, "document.save"); err != nil {
		return Document{}, err
	} else if ok {
		return service.Get(ctx, request.ActorID, existing.String(), true)
	}
	base, err := uuid.Parse(request.BaseVersionID)
	if err != nil {
		return Document{}, ErrVersionConflict
	}
	versionID, objectID := uuid.New(), uuid.New()
	key := path.Join("documents", current.organizationID.String(), workspaceID.String(), documentID.String(), versionID.String(), "document.documi.json")
	object, err := service.storage.Put(ctx, key, bytes.NewReader(content))
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
	var persistedCurrent uuid.UUID
	if err = tx.QueryRow(ctx, "SELECT current_version_id FROM documents WHERE id=$1 FOR UPDATE", documentID).Scan(&persistedCurrent); err != nil {
		return Document{}, err
	}
	if persistedCurrent != base {
		return Document{}, ErrVersionConflict
	}
	var locked bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM wopi_locks WHERE document_id=$1 AND expires_at>now())", documentID).Scan(&locked); err != nil {
		return Document{}, err
	}
	if locked {
		return Document{}, ErrLocked
	}
	var number int64
	if err = tx.QueryRow(ctx, "SELECT COALESCE(MAX(version_number),0)+1 FROM document_versions WHERE document_id=$1", documentID).Scan(&number); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO storage_objects(id,provider,storage_key,sha256,size,content_type) VALUES($1,'filesystem',$2,$3,$4,$5)", objectID, key, object.SHA256, object.Size, NativeMIME); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO document_versions(id,document_id,parent_version_id,storage_object_id,version_number,created_by) VALUES($1,$2,$3,$4,$5,$6)", versionID, documentID, base, objectID, number, actor); err != nil {
		return Document{}, err
	}
	result, err := tx.Exec(ctx, "UPDATE documents SET current_version_id=$1,updated_at=now() WHERE id=$2 AND current_version_id=$3", versionID, documentID, base)
	if err != nil {
		return Document{}, err
	}
	if result.RowsAffected() != 1 {
		return Document{}, ErrVersionConflict
	}
	if _, err = tx.Exec(ctx, "INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,'document.save',$4)", current.organizationID, actor, request.IdempotencyKey, documentID); err != nil {
		return Document{}, err
	}
	if err = appendAudit(ctx, tx, "document.editor.saved", current.organizationID, workspaceID, actor, documentID); err != nil {
		return Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	committed = true
	return service.Get(ctx, request.ActorID, request.DocumentID, true)
}

func validIdempotencyKey(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && len(trimmed) <= 255
}

func (service *Service) Versions(ctx context.Context, actorID, documentID string) ([]Version, error) {
	if _, err := service.Get(ctx, actorID, documentID, false); err != nil {
		return nil, err
	}
	id := uuid.MustParse(documentID)
	rows, err := service.pool.Query(ctx, `SELECT v.id,v.document_id,v.parent_version_id,v.version_number,o.sha256,o.size,v.created_at FROM document_versions v JOIN storage_objects o ON o.id=v.storage_object_id WHERE v.document_id=$1 ORDER BY v.version_number DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Version{}
	for rows.Next() {
		var v Version
		if err := rows.Scan(&v.ID, &v.DocumentID, &v.ParentVersionID, &v.VersionNumber, &v.SHA256, &v.Size, &v.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (service *Service) Download(ctx context.Context, actorID, documentID string) (Document, io.ReadCloser, error) {
	value, err := service.Get(ctx, actorID, documentID, false)
	if err != nil {
		return Document{}, nil, err
	}
	id := uuid.MustParse(value.ID)
	workspaceID := uuid.MustParse(value.WorkspaceID)
	var folderID *uuid.UUID
	if value.FolderID != nil {
		parsed := uuid.MustParse(*value.FolderID)
		folderID = &parsed
	}
	allowed, err := service.authorizer.Allows(ctx, actorID, permissions.Resource{OrganizationID: value.organizationID, WorkspaceID: workspaceID, FolderID: folderID, DocumentID: &id}, permissions.CanDownload)
	if err != nil {
		return Document{}, nil, err
	}
	if !allowed {
		return Document{}, nil, ErrNotFound
	}
	body, err := service.verifiedContent(ctx, value)
	if err != nil {
		return Document{}, nil, err
	}
	if err := service.RecordAudit(ctx, actorID, documentID, "document.downloaded"); err != nil {
		return Document{}, nil, err
	}
	return value, io.NopCloser(bytes.NewReader(body)), nil
}

func (service *Service) RecordAudit(ctx context.Context, actorID, documentID, event string) error {
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return ErrForbidden
	}
	id, err := uuid.Parse(documentID)
	if err != nil {
		return ErrNotFound
	}
	var organizationID, workspaceID uuid.UUID
	if err := service.pool.QueryRow(ctx, "SELECT organization_id,workspace_id FROM documents WHERE id=$1", id).Scan(&organizationID, &workspaceID); err != nil {
		return err
	}
	_, err = service.pool.Exec(ctx, "INSERT INTO audit_events(id,event_type,organization_id,workspace_id,actor_id,resource_id) VALUES($1,$2,$3,$4,$5,$6)", fmt.Sprintf("evt_%s", uuid.New()), event, organizationID, workspaceID, actor, id)
	return err
}

func validateNative(content []byte) ([]byte, error) {
	if len(content) == 0 || len(content) > MaxNativeBytes {
		return nil, ErrInvalidDocument
	}
	var envelope struct {
		Format string `json:"format"`
	}
	if json.Unmarshal(content, &envelope) != nil || envelope.Format != NativeFormat {
		return nil, ErrInvalidDocument
	}
	compact := bytes.Buffer{}
	if json.Compact(&compact, content) != nil {
		return nil, ErrInvalidDocument
	}
	return compact.Bytes(), nil
}

func (service *Service) idempotentDocument(ctx context.Context, organizationID, actor uuid.UUID, key, operation string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := service.pool.QueryRow(ctx, "SELECT resource_id FROM idempotency_keys WHERE organization_id=$1 AND actor_id=$2 AND key=$3 AND operation=$4", organizationID, actor, key, operation).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	return id, err == nil, err
}

type scanner interface{ Scan(...any) error }

func scanDocument(row scanner) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.FolderID, &d.Title, &d.Family, &d.MIMEType, &d.EditorProvider, &d.StorageMode, &d.CurrentVersionID, &d.CreatedAt, &d.UpdatedAt, &d.organizationID, &d.ExternalProvider, &d.ExternalFileID, &d.ExternalURL)
	return d, err
}
func scanDocumentWithStorage(row scanner) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.FolderID, &d.Title, &d.Family, &d.MIMEType, &d.EditorProvider, &d.StorageMode, &d.CurrentVersionID, &d.CreatedAt, &d.UpdatedAt, &d.organizationID, &d.storageKey, &d.size, &d.sha256, &d.ExternalProvider, &d.ExternalFileID, &d.ExternalURL)
	return d, err
}

func (service *Service) verifiedContent(ctx context.Context, document Document) ([]byte, error) {
	reader, _, err := service.storage.Get(ctx, document.storageKey)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	body, err := io.ReadAll(io.LimitReader(reader, MaxDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) != document.size || fmt.Sprintf("%x", sha256.Sum256(body)) != document.sha256 {
		return nil, fmt.Errorf("document content hash or size mismatch")
	}
	return body, nil
}
func appendAudit(ctx context.Context, tx pgx.Tx, event string, organizationID, workspaceID, actorID, resourceID uuid.UUID) error {
	_, err := tx.Exec(ctx, "INSERT INTO audit_events(id,event_type,organization_id,workspace_id,actor_id,resource_id) VALUES($1,$2,$3,$4,$5,$6)", fmt.Sprintf("evt_%s", uuid.New()), event, organizationID, workspaceID, actorID, resourceID)
	return err
}
