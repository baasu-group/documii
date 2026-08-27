package administration

import (
	"context"
	"errors"
	"strings"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (service *Service) Audit(ctx context.Context, actorID, organizationID, workspaceFilter, eventFilter, cursorValue string, limit int) (AuditPage, error) {
	resource, err := service.organizationResource(ctx, organizationID)
	if err != nil {
		return AuditPage{}, err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return AuditPage{}, err
	}
	cursor, err := decodeCursor(cursorValue)
	if err != nil {
		return AuditPage{}, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		return AuditPage{}, ErrInvalidRequest
	}
	var workspaceID *uuid.UUID
	if workspaceFilter != "" {
		value, parseErr := uuid.Parse(workspaceFilter)
		if parseErr != nil {
			return AuditPage{}, ErrInvalidRequest
		}
		workspaceID = &value
	}
	rows, err := service.pool.Query(ctx, `SELECT id,event_type,organization_id,workspace_id,actor_id,resource_id,request_id,data,created_at
		FROM audit_events WHERE organization_id=$1
		AND ($2::uuid IS NULL OR workspace_id=$2)
		AND ($3='' OR event_type=$3)
		AND (created_at,id)<($4,$5)
		ORDER BY created_at DESC,id DESC LIMIT $6`, resource.OrganizationID, workspaceID, strings.TrimSpace(eventFilter), cursor.Timestamp, cursor.ID, limit+1)
	if err != nil {
		return AuditPage{}, err
	}
	defer rows.Close()
	events := []AuditEvent{}
	for rows.Next() {
		var value AuditEvent
		if err := rows.Scan(&value.ID, &value.Type, &value.OrganizationID, &value.WorkspaceID, &value.ActorID, &value.ResourceID, &value.RequestID, &value.Data, &value.Timestamp); err != nil {
			return AuditPage{}, err
		}
		events = append(events, value)
	}
	if err := rows.Err(); err != nil {
		return AuditPage{}, err
	}
	page := AuditPage{Events: events}
	if len(events) > limit {
		last := events[limit-1]
		page.Events = events[:limit]
		page.NextCursor = encodeCursor(auditCursor{Timestamp: last.Timestamp, ID: last.ID})
	}
	return page, nil
}

func (service *Service) EditorPolicies(ctx context.Context, actorID, organizationID string) ([]EditorPolicy, error) {
	resource, err := service.organizationResource(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanRead); err != nil {
		return nil, err
	}
	rows, err := service.pool.Query(ctx, `SELECT organization_id,family,provider_id,updated_at
		FROM document_editor_policies WHERE organization_id=$1 ORDER BY family`, resource.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []EditorPolicy{}
	for rows.Next() {
		var value EditorPolicy
		if err := rows.Scan(&value.OrganizationID, &value.Family, &value.ProviderID, &value.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) SetEditorPolicy(ctx context.Context, actorID, organizationID, family, providerID string) (EditorPolicy, error) {
	resource, err := service.organizationResource(ctx, organizationID)
	if err != nil {
		return EditorPolicy{}, err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return EditorPolicy{}, err
	}
	if service.validatePolicy == nil || service.validatePolicy(ctx, family, providerID) != nil {
		return EditorPolicy{}, ErrInvalidRequest
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return EditorPolicy{}, err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return EditorPolicy{}, err
	}
	defer tx.Rollback(ctx)
	var result EditorPolicy
	if err = tx.QueryRow(ctx, `INSERT INTO document_editor_policies(organization_id,family,provider_id,updated_by)
		VALUES($1,$2,$3,$4) ON CONFLICT(organization_id,family) DO UPDATE
		SET provider_id=excluded.provider_id,updated_by=excluded.updated_by,updated_at=now()
		RETURNING organization_id,family,provider_id,updated_at`, resource.OrganizationID, family, providerID, actor).
		Scan(&result.OrganizationID, &result.Family, &result.ProviderID, &result.UpdatedAt); err != nil {
		return EditorPolicy{}, mapConstraint(err)
	}
	if err = appendAudit(ctx, tx, "document.editor_policy.updated", resource.OrganizationID, nil, actor, resource.OrganizationID, map[string]any{"family": family, "providerId": providerID}); err != nil {
		return EditorPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return EditorPolicy{}, err
	}
	return result, nil
}

func (service *Service) EffectiveAccess(ctx context.Context, actorID, subjectID, scopeType, scopeID string) ([]permissions.Capability, error) {
	actor, err := parseActor(actorID)
	if err != nil {
		return nil, err
	}
	if subjectID == "" {
		subjectID = actor.String()
	}
	subject, err := uuid.Parse(subjectID)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	var organizationID uuid.UUID
	switch scopeType {
	case "organization":
		var resource permissions.Resource
		resource, err = service.organizationResource(ctx, scopeID)
		organizationID = resource.OrganizationID
	case "workspace":
		id, parseErr := uuid.Parse(scopeID)
		err = parseErr
		if err == nil {
			err = service.pool.QueryRow(ctx, "SELECT organization_id FROM workspaces WHERE id=$1", id).Scan(&organizationID)
		}
	case "folder":
		id, parseErr := uuid.Parse(scopeID)
		err = parseErr
		if err == nil {
			err = service.pool.QueryRow(ctx, "SELECT organization_id FROM folders WHERE id=$1", id).Scan(&organizationID)
		}
	case "document":
		id, parseErr := uuid.Parse(scopeID)
		err = parseErr
		if err == nil {
			err = service.pool.QueryRow(ctx, "SELECT organization_id FROM documents WHERE id=$1", id).Scan(&organizationID)
		}
	default:
		return nil, ErrInvalidRequest
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, ErrInvalidRequest
	}
	resource, _, _, _, err := service.scopeResource(ctx, organizationID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	if subject != actor {
		if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
			return nil, err
		}
	}
	capabilities, err := service.authorizer.Capabilities(ctx, subject.String(), resource)
	if err != nil {
		return nil, err
	}
	result := make([]permissions.Capability, 0, len(capabilities))
	for _, capability := range []permissions.Capability{permissions.CanRead, permissions.CanCreate, permissions.CanEdit, permissions.CanDownload, permissions.CanDelete, permissions.CanManagePermissions, permissions.CanManageMembers, permissions.CanTransferOwnership} {
		if capabilities[capability] {
			result = append(result, capability)
		}
	}
	return result, nil
}
