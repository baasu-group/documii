package administration

import (
	"context"
	"errors"
	"strings"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (service *Service) Folders(ctx context.Context, actorID, workspaceID string) ([]Folder, error) {
	resource, err := service.workspaceResource(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanRead); err != nil {
		return nil, err
	}
	rows, err := service.pool.Query(ctx, `SELECT id,organization_id,workspace_id,parent_id,name,created_at,updated_at
		FROM folders WHERE workspace_id=$1 ORDER BY name,id`, resource.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Folder{}
	for rows.Next() {
		var value Folder
		if err := rows.Scan(&value.ID, &value.OrganizationID, &value.WorkspaceID, &value.ParentID, &value.Name, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) CreateFolder(ctx context.Context, actorID, workspaceID, name string, parentID *string) (Folder, error) {
	resource, err := service.workspaceResource(ctx, workspaceID)
	if err != nil || !validName(name) {
		if err != nil {
			return Folder{}, err
		}
		return Folder{}, ErrInvalidRequest
	}
	var parent *uuid.UUID
	if parentID != nil {
		value, parseErr := uuid.Parse(*parentID)
		if parseErr != nil {
			return Folder{}, ErrInvalidRequest
		}
		parent = &value
		resource.FolderID = parent
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return Folder{}, err
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return Folder{}, err
	}
	id := uuid.New()
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Folder{}, err
	}
	defer tx.Rollback(ctx)
	var result Folder
	if err = tx.QueryRow(ctx, `INSERT INTO folders(id,organization_id,workspace_id,parent_id,name)
		VALUES($1,$2,$3,$4,$5) RETURNING id,organization_id,workspace_id,parent_id,name,created_at,updated_at`,
		id, resource.OrganizationID, resource.WorkspaceID, parent, strings.TrimSpace(name)).Scan(&result.ID, &result.OrganizationID, &result.WorkspaceID, &result.ParentID, &result.Name, &result.CreatedAt, &result.UpdatedAt); err != nil {
		return Folder{}, mapConstraint(err)
	}
	workspace := resource.WorkspaceID
	if err = appendAudit(ctx, tx, "folder.created", resource.OrganizationID, &workspace, actor, id, map[string]any{"name": result.Name}); err != nil {
		return Folder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Folder{}, err
	}
	return result, nil
}

func (service *Service) UpdateFolder(ctx context.Context, actorID, folderID string, name *string, parentSpecified bool, parentID *string) (Folder, error) {
	id, err := uuid.Parse(folderID)
	if err != nil {
		return Folder{}, ErrNotFound
	}
	var current Folder
	if err = service.pool.QueryRow(ctx, `SELECT id,organization_id,workspace_id,parent_id,name,created_at,updated_at FROM folders WHERE id=$1`, id).
		Scan(&current.ID, &current.OrganizationID, &current.WorkspaceID, &current.ParentID, &current.Name, &current.CreatedAt, &current.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return Folder{}, ErrNotFound
	} else if err != nil {
		return Folder{}, err
	}
	organizationID := uuid.MustParse(current.OrganizationID)
	workspaceID := uuid.MustParse(current.WorkspaceID)
	currentID := id
	resource := permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID, FolderID: &currentID}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return Folder{}, err
	}
	nextName := current.Name
	if name != nil {
		if !validName(*name) {
			return Folder{}, ErrInvalidRequest
		}
		nextName = strings.TrimSpace(*name)
	}
	var nextParent *uuid.UUID
	if !parentSpecified {
		if current.ParentID != nil {
			value := uuid.MustParse(*current.ParentID)
			nextParent = &value
		}
	} else if parentID != nil {
		value, parseErr := uuid.Parse(*parentID)
		if parseErr != nil || value == id {
			return Folder{}, ErrInvalidRequest
		}
		nextParent = &value
	}
	if parentSpecified {
		destination := permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID, FolderID: nextParent}
		if err := service.require(ctx, actorID, destination, permissions.CanManagePermissions); err != nil {
			return Folder{}, err
		}
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return Folder{}, err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Folder{}, err
	}
	defer tx.Rollback(ctx)
	var result Folder
	if err = tx.QueryRow(ctx, `UPDATE folders SET name=$2,parent_id=$3,updated_at=now() WHERE id=$1
		RETURNING id,organization_id,workspace_id,parent_id,name,created_at,updated_at`, id, nextName, nextParent).
		Scan(&result.ID, &result.OrganizationID, &result.WorkspaceID, &result.ParentID, &result.Name, &result.CreatedAt, &result.UpdatedAt); err != nil {
		return Folder{}, mapConstraint(err)
	}
	if err = appendAudit(ctx, tx, "folder.updated", organizationID, &workspaceID, actor, id, map[string]any{"name": result.Name, "parentId": result.ParentID}); err != nil {
		return Folder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Folder{}, err
	}
	return result, nil
}

func (service *Service) DeleteFolder(ctx context.Context, actorID, folderID string) error {
	id, err := uuid.Parse(folderID)
	if err != nil {
		return ErrNotFound
	}
	var organizationID, workspaceID uuid.UUID
	if err = service.pool.QueryRow(ctx, "SELECT organization_id,workspace_id FROM folders WHERE id=$1", id).Scan(&organizationID, &workspaceID); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	resource := permissions.Resource{OrganizationID: organizationID, WorkspaceID: workspaceID, FolderID: &id}
	if err := service.require(ctx, actorID, resource, permissions.CanManagePermissions); err != nil {
		return err
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
	if _, err = tx.Exec(ctx, "DELETE FROM folders WHERE id=$1", id); err != nil {
		return mapConstraint(err)
	}
	if err = appendAudit(ctx, tx, "folder.deleted", organizationID, &workspaceID, actor, id, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mapConstraint(err error) error {
	var postgres interface{ SQLState() string }
	if errors.As(err, &postgres) {
		switch postgres.SQLState() {
		case "23503", "23505", "23514", "P0001":
			return ErrConflict
		}
	}
	return err
}
