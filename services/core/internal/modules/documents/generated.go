package documents

import (
	"bytes"
	"context"
	"errors"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type GeneratedDocumentRequest struct {
	ActorID, OrganizationID, WorkspaceID, IdempotencyKey string
	Title, Family, MIMEType, Extension                   string
	Content                                              []byte
}

// CreateGenerated creates an immutable managed artifact produced by a trusted
// server adapter. It never accepts a path or storage key from the caller.
func (service *Service) CreateGenerated(ctx context.Context, request GeneratedDocumentRequest) (Document, error) {
	if !validIdempotencyKey(request.IdempotencyKey) || len(request.Content) == 0 || len(request.Content) > MaxDocumentBytes {
		return Document{}, ErrInvalidDocument
	}
	title := strings.TrimSpace(request.Title)
	if title == "" || utf8.RuneCountInString(title) > 255 {
		return Document{}, ErrInvalidDocument
	}
	allowedFormats := map[string][2]string{
		"csv":  {"text", "text/csv"},
		"xlsx": {"spreadsheet", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		"pdf":  {"pdf", "application/pdf"},
	}
	format, ok := allowedFormats[request.Extension]
	if !ok || request.Family != format[0] || request.MIMEType != format[1] {
		return Document{}, ErrInvalidDocument
	}
	actor, err := uuid.Parse(request.ActorID)
	if err != nil {
		return Document{}, ErrForbidden
	}
	organizationID, err := uuid.Parse(request.OrganizationID)
	if err != nil {
		return Document{}, ErrNotFound
	}
	workspaceID, err := uuid.Parse(request.WorkspaceID)
	if err != nil {
		return Document{}, ErrNotFound
	}
	var belongs bool
	if err = service.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1 AND organization_id=$2)`, workspaceID, organizationID).Scan(&belongs); err != nil || !belongs {
		if err != nil {
			return Document{}, err
		}
		return Document{}, ErrNotFound
	}
	if allowed, err := service.authorizer.Allows(ctx, request.ActorID, permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID}, permissions.CanCreate); err != nil || !allowed {
		if err != nil {
			return Document{}, err
		}
		return Document{}, ErrForbidden
	}
	if existing, found, err := service.idempotentDocument(ctx, organizationID, actor, request.IdempotencyKey, "report.export"); err != nil {
		return Document{}, err
	} else if found {
		return service.Get(ctx, request.ActorID, existing.String(), false)
	}
	providerID := "native"
	if request.Family == "spreadsheet" && service.editors != nil {
		providerID, err = service.editors.Select(ctx, organizationID.String(), request.Family)
		if err != nil {
			return Document{}, err
		}
	}
	documentID, versionID, objectID, folderID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	key := path.Join("documents", organizationID.String(), workspaceID.String(), documentID.String(), versionID.String(), "report."+request.Extension)
	object, err := service.storage.Put(ctx, key, bytes.NewReader(request.Content))
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
	err = tx.QueryRow(ctx, `SELECT id FROM folders WHERE organization_id=$1 AND workspace_id=$2 AND parent_id IS NULL AND name='Reports' ORDER BY created_at LIMIT 1`, organizationID, workspaceID).Scan(&folderID)
	if errors.Is(err, pgx.ErrNoRows) {
		folderID = uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO folders(id,organization_id,workspace_id,name) VALUES($1,$2,$3,'Reports')`, folderID, organizationID, workspaceID)
	}
	if err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO documents(id,organization_id,workspace_id,folder_id,title,family,mime_type,editor_provider,storage_mode,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'managed',$9)`, documentID, organizationID, workspaceID, folderID, title, request.Family, request.MIMEType, providerID, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO storage_objects(id,provider,storage_key,sha256,size,content_type) VALUES($1,'filesystem',$2,$3,$4,$5)`, objectID, key, object.SHA256, object.Size, request.MIMEType); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO document_versions(id,document_id,storage_object_id,version_number,created_by) VALUES($1,$2,$3,1,$4)`, versionID, documentID, objectID, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE documents SET current_version_id=$2 WHERE id=$1`, documentID, versionID); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,'report.export',$4)`, organizationID, actor, request.IdempotencyKey, documentID); err != nil {
		return Document{}, err
	}
	for _, event := range []string{"report.generated", "report.exported", "document.created"} {
		if err = appendAudit(ctx, tx, event, organizationID, workspaceID, actor, documentID); err != nil {
			return Document{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	committed = true
	return service.Get(ctx, request.ActorID, documentID.String(), false)
}
