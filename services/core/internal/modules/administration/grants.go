package administration

import (
	"context"
	"errors"
	"time"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateGrantRequest struct {
	PrincipalType string `json:"principalType"`
	PrincipalID   string `json:"principalId"`
	ScopeType     string `json:"scopeType"`
	ScopeID       string `json:"scopeId"`
	Role          string `json:"role"`
}

func (service *Service) Grants(ctx context.Context, actorID, organizationID string) ([]Grant, error) {
	resource, err := service.organizationResource(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return nil, err
	}
	rows, err := service.pool.Query(ctx, `SELECT id,organization_id,principal_type::text,
		COALESCE(user_id,group_id),scope_type::text,
		CASE scope_type WHEN 'organization' THEN organization_id WHEN 'workspace' THEN workspace_id
		     WHEN 'folder' THEN folder_id WHEN 'document' THEN document_id END,
		role::text,created_at FROM authorization_grants WHERE organization_id=$1 ORDER BY created_at,id`, resource.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Grant{}
	for rows.Next() {
		var value Grant
		if err := rows.Scan(&value.ID, &value.OrganizationID, &value.PrincipalType, &value.PrincipalID, &value.ScopeType, &value.ScopeID, &value.Role, &value.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) CreateGrant(ctx context.Context, actorID, organizationID string, request CreateGrantRequest) (Grant, error) {
	organization, err := service.organizationResource(ctx, organizationID)
	if err != nil {
		return Grant{}, err
	}
	principal, err := uuid.Parse(request.PrincipalID)
	if err != nil || (request.PrincipalType != "user" && request.PrincipalType != "group") {
		return Grant{}, ErrInvalidRequest
	}
	role := permissions.Role(request.Role)
	if !permissions.ValidRole(role) || (role == permissions.RoleOwner && request.ScopeType != "organization") {
		return Grant{}, ErrInvalidRequest
	}
	target, workspaceID, folderID, documentID, err := service.scopeResource(ctx, organization.OrganizationID, request.ScopeType, request.ScopeID)
	if err != nil {
		return Grant{}, err
	}
	if err := service.require(ctx, actorID, target, permissions.CanManagePermissions); err != nil {
		return Grant{}, err
	}
	if role == permissions.RoleOwner {
		if err := service.require(ctx, actorID, organization, permissions.CanTransferOwnership); err != nil {
			return Grant{}, err
		}
	}
	if request.PrincipalType == "user" {
		var exists bool
		if err := service.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", principal).Scan(&exists); err != nil {
			return Grant{}, err
		} else if !exists {
			return Grant{}, ErrNotFound
		}
	} else {
		var exists bool
		if err := service.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM groups WHERE id=$1 AND organization_id=$2)", principal, organization.OrganizationID).Scan(&exists); err != nil {
			return Grant{}, err
		} else if !exists {
			return Grant{}, ErrNotFound
		}
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return Grant{}, err
	}
	id := uuid.New()
	var userID, groupID *uuid.UUID
	if request.PrincipalType == "user" {
		userID = &principal
	} else {
		groupID = &principal
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Grant{}, err
	}
	defer tx.Rollback(ctx)
	var createdAt time.Time
	if err = tx.QueryRow(ctx, `INSERT INTO authorization_grants(
		id,organization_id,principal_type,user_id,group_id,scope_type,workspace_id,folder_id,document_id,role,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING created_at`, id, organization.OrganizationID, request.PrincipalType, userID, groupID, request.ScopeType, workspaceID, folderID, documentID, request.Role, actor).Scan(&createdAt); err != nil {
		return Grant{}, mapConstraint(err)
	}
	var workspaceAudit *uuid.UUID
	if workspaceID != nil {
		workspaceAudit = workspaceID
	}
	if err = appendAudit(ctx, tx, "authorization.grant.created", organization.OrganizationID, workspaceAudit, actor, id, map[string]any{"principalType": request.PrincipalType, "principalId": principal, "scopeType": request.ScopeType, "scopeId": request.ScopeID, "role": request.Role}); err != nil {
		return Grant{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Grant{}, err
	}
	return Grant{ID: id.String(), OrganizationID: organization.OrganizationID.String(), PrincipalType: request.PrincipalType, PrincipalID: principal.String(), ScopeType: request.ScopeType, ScopeID: request.ScopeID, Role: request.Role, CreatedAt: createdAt}, nil
}

func (service *Service) DeleteGrant(ctx context.Context, actorID, grantID string) error {
	id, err := uuid.Parse(grantID)
	if err != nil {
		return ErrNotFound
	}
	var organizationID uuid.UUID
	var scopeType, role string
	var workspaceID, folderID, documentID *uuid.UUID
	if err = service.pool.QueryRow(ctx, `SELECT organization_id,scope_type::text,workspace_id,folder_id,document_id,role::text
		FROM authorization_grants WHERE id=$1`, id).Scan(&organizationID, &scopeType, &workspaceID, &folderID, &documentID, &role); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	resource := permissions.Resource{OrganizationID: organizationID, FolderID: folderID, DocumentID: documentID}
	if workspaceID != nil {
		resource.WorkspaceID = *workspaceID
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return err
	}
	if role == string(permissions.RoleOwner) {
		if err := service.require(ctx, actorID, permissions.Resource{OrganizationID: organizationID}, permissions.CanTransferOwnership); err != nil {
			return err
		}
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "DELETE FROM authorization_grants WHERE id=$1", id); err != nil {
		return mapConstraint(err)
	}
	if err = appendAudit(ctx, tx, "authorization.grant.deleted", organizationID, workspaceID, actor, id, map[string]any{"scopeType": scopeType, "role": role}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (service *Service) scopeResource(ctx context.Context, organizationID uuid.UUID, scopeType, scopeID string) (permissions.Resource, *uuid.UUID, *uuid.UUID, *uuid.UUID, error) {
	resource := permissions.Resource{OrganizationID: organizationID}
	if scopeType == "organization" {
		id, err := uuid.Parse(scopeID)
		if err != nil || id != organizationID {
			return resource, nil, nil, nil, ErrInvalidRequest
		}
		return resource, nil, nil, nil, nil
	}
	id, err := uuid.Parse(scopeID)
	if err != nil {
		return resource, nil, nil, nil, ErrInvalidRequest
	}
	var workspaceID uuid.UUID
	switch scopeType {
	case "workspace":
		if err := service.pool.QueryRow(ctx, "SELECT id FROM workspaces WHERE id=$1 AND organization_id=$2", id, organizationID).Scan(&workspaceID); errors.Is(err, pgx.ErrNoRows) {
			return resource, nil, nil, nil, ErrNotFound
		} else if err != nil {
			return resource, nil, nil, nil, err
		}
		resource.WorkspaceID = workspaceID
		return resource, &workspaceID, nil, nil, nil
	case "folder":
		if err := service.pool.QueryRow(ctx, "SELECT workspace_id FROM folders WHERE id=$1 AND organization_id=$2", id, organizationID).Scan(&workspaceID); errors.Is(err, pgx.ErrNoRows) {
			return resource, nil, nil, nil, ErrNotFound
		} else if err != nil {
			return resource, nil, nil, nil, err
		}
		resource.WorkspaceID, resource.FolderID = workspaceID, &id
		return resource, &workspaceID, &id, nil, nil
	case "document":
		var folderID *uuid.UUID
		if err := service.pool.QueryRow(ctx, "SELECT workspace_id,folder_id FROM documents WHERE id=$1 AND organization_id=$2", id, organizationID).Scan(&workspaceID, &folderID); errors.Is(err, pgx.ErrNoRows) {
			return resource, nil, nil, nil, ErrNotFound
		} else if err != nil {
			return resource, nil, nil, nil, err
		}
		resource.WorkspaceID, resource.FolderID, resource.DocumentID = workspaceID, folderID, &id
		return resource, &workspaceID, nil, &id, nil
	default:
		return resource, nil, nil, nil, ErrInvalidRequest
	}
}
