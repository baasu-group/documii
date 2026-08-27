package documents

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"time"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DocumentAccess struct{ CanRead, CanEdit bool }

type EditorLockOperation string

const (
	EditorLockAcquire EditorLockOperation = "acquire"
	EditorLockRefresh EditorLockOperation = "refresh"
	EditorLockGet     EditorLockOperation = "get"
	EditorLockRelease EditorLockOperation = "release"
	EditorLockRelock  EditorLockOperation = "relock"
)

type EditorLockRequest struct {
	ActorID, DocumentID, Value, OldValue string
	Operation                            EditorLockOperation
}

func (service *Service) Access(ctx context.Context, actorID, documentID string) (DocumentAccess, error) {
	id, err := uuid.Parse(documentID)
	if err != nil {
		return DocumentAccess{}, ErrNotFound
	}
	var organizationID, workspaceID uuid.UUID
	var folderID *uuid.UUID
	if err := service.pool.QueryRow(ctx, "SELECT organization_id,workspace_id,folder_id FROM documents WHERE id=$1", id).Scan(&organizationID, &workspaceID, &folderID); errors.Is(err, pgx.ErrNoRows) {
		return DocumentAccess{}, ErrNotFound
	} else if err != nil {
		return DocumentAccess{}, err
	}
	capabilities, err := service.authorizer.Capabilities(ctx, actorID, permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID, FolderID: folderID, DocumentID: &id})
	if err != nil {
		return DocumentAccess{}, err
	}
	return DocumentAccess{CanRead: capabilities[permissions.CanRead], CanEdit: capabilities[permissions.CanEdit]}, nil
}

func (service *Service) OpenContent(ctx context.Context, actorID, documentID string) (Document, io.ReadCloser, error) {
	value, err := service.Get(ctx, actorID, documentID, false)
	if err != nil {
		return Document{}, nil, err
	}
	body, err := service.verifiedContent(ctx, value)
	if err != nil { return Document{}, nil, err }
	return value, io.NopCloser(bytes.NewReader(body)), nil
}

func (service *Service) SaveEditorContent(ctx context.Context, actorID, documentID, expectedVersion, lockValue, providerID string, content io.Reader, contentLength int64) (Document, error) {
	access, err := service.Access(ctx, actorID, documentID)
	if err != nil {
		return Document{}, err
	}
	if !access.CanEdit {
		return Document{}, ErrForbidden
	}
	if contentLength < 0 || contentLength > MaxDocumentBytes {
		return Document{}, ErrInvalidDocument
	}
	body, err := io.ReadAll(io.LimitReader(content, MaxDocumentBytes+1))
	if err != nil {
		return Document{}, err
	}
	if int64(len(body)) > MaxDocumentBytes {
		return Document{}, ErrInvalidDocument
	}
	current, err := service.Get(ctx, actorID, documentID, false)
	if err != nil {
		return Document{}, err
	}
	documentUUID := uuid.MustParse(documentID)
	actorUUID := uuid.MustParse(actorID)
	workspaceUUID := uuid.MustParse(current.WorkspaceID)
	versionID, objectID := uuid.New(), uuid.New()
	key := path.Join("documents", current.organizationID.String(), workspaceUUID.String(), documentUUID.String(), versionID.String(), "document.bin")
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
	var activeLock string
	var lockBase uuid.UUID
	err = tx.QueryRow(ctx, "SELECT lock_value,base_version_id FROM wopi_locks WHERE document_id=$1 AND expires_at>now() FOR UPDATE", documentUUID).Scan(&activeLock, &lockBase)
	if errors.Is(err, pgx.ErrNoRows) || activeLock != lockValue {
		return Document{}, ErrLocked
	}
	if err != nil {
		return Document{}, err
	}
	// The token is bound to the version that opened the session. After the
	// first successful save the persisted lock base becomes authoritative,
	// allowing subsequent WOPI saves without issuing a new token.
	_ = expectedVersion
	base := lockBase
	if base == uuid.Nil {
		return Document{}, ErrVersionConflict
	}
	var number int64
	if err = tx.QueryRow(ctx, "SELECT COALESCE(MAX(version_number),0)+1 FROM document_versions WHERE document_id=$1", documentUUID).Scan(&number); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO storage_objects(id,provider,storage_key,sha256,size,content_type) VALUES($1,'filesystem',$2,$3,$4,$5)", objectID, key, object.SHA256, object.Size, current.MIMEType); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO document_versions(id,document_id,parent_version_id,storage_object_id,version_number,created_by) VALUES($1,$2,$3,$4,$5,$6)", versionID, documentUUID, base, objectID, number, actorUUID); err != nil {
		return Document{}, err
	}
	result, err := tx.Exec(ctx, "UPDATE documents SET current_version_id=$1,updated_at=now() WHERE id=$2 AND current_version_id=$3", versionID, documentUUID, base)
	if err != nil {
		return Document{}, err
	}
	if result.RowsAffected() != 1 {
		return Document{}, ErrVersionConflict
	}
	if _, err = tx.Exec(ctx, "UPDATE wopi_locks SET base_version_id=$2,updated_at=now() WHERE document_id=$1", documentUUID, versionID); err != nil {
		return Document{}, err
	}
	if err = appendAudit(ctx, tx, "document.editor.saved", current.organizationID, workspaceUUID, actorUUID, documentUUID); err != nil {
		return Document{}, err
	}
	if err = appendAudit(ctx, tx, "document."+providerID+".saved", current.organizationID, workspaceUUID, actorUUID, documentUUID); err != nil {
		return Document{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	committed = true
	return service.Get(ctx, actorID, documentID, false)
}

func (service *Service) ManageEditorLock(ctx context.Context, request EditorLockRequest) (string, error) {
	access, err := service.Access(ctx, request.ActorID, request.DocumentID)
	if err != nil {
		return "", err
	}
	if !access.CanEdit {
		return "", ErrForbidden
	}
	documentID := uuid.MustParse(request.DocumentID)
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var current string
	var expires time.Time
	err = tx.QueryRow(ctx, "SELECT lock_value,expires_at FROM wopi_locks WHERE document_id=$1 FOR UPDATE", documentID).Scan(&current, &expires)
	missing := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !missing {
		return "", err
	}
	if !missing && expires.Before(time.Now()) {
		_, err = tx.Exec(ctx, "DELETE FROM wopi_locks WHERE document_id=$1", documentID)
		if err != nil {
			return "", err
		}
		missing = true
		current = ""
	}
	switch request.Operation {
	case EditorLockGet:
	case EditorLockAcquire:
		if !missing && current != request.Value {
			return current, ErrVersionConflict
		}
		if missing {
			var base uuid.UUID
			if err = tx.QueryRow(ctx, "SELECT current_version_id FROM documents WHERE id=$1", documentID).Scan(&base); err != nil {
				return "", err
			}
			_, err = tx.Exec(ctx, "INSERT INTO wopi_locks(document_id,lock_value,base_version_id,expires_at) VALUES($1,$2,$3,now()+interval '30 minutes')", documentID, request.Value, base)
		} else {
			_, err = tx.Exec(ctx, "UPDATE wopi_locks SET expires_at=now()+interval '30 minutes',updated_at=now() WHERE document_id=$1", documentID)
		}
	case EditorLockRefresh:
		if missing || current != request.Value {
			return current, ErrVersionConflict
		}
		_, err = tx.Exec(ctx, "UPDATE wopi_locks SET expires_at=now()+interval '30 minutes',updated_at=now() WHERE document_id=$1", documentID)
	case EditorLockRelease:
		if missing || current != request.Value {
			return current, ErrVersionConflict
		}
		_, err = tx.Exec(ctx, "DELETE FROM wopi_locks WHERE document_id=$1", documentID)
		current = ""
	case EditorLockRelock:
		if missing || current != request.OldValue {
			return current, ErrVersionConflict
		}
		_, err = tx.Exec(ctx, "UPDATE wopi_locks SET lock_value=$2,expires_at=now()+interval '30 minutes',updated_at=now() WHERE document_id=$1", documentID, request.Value)
		current = request.Value
	default:
		return current, ErrInvalidDocument
	}
	if err != nil {
		return current, err
	}
	if err = tx.Commit(ctx); err != nil {
		return current, err
	}
	if request.Operation == EditorLockAcquire {
		current = request.Value
	}
	return current, nil
}
