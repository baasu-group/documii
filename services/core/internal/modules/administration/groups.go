package administration

import (
	"context"
	"errors"
	"strings"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (service *Service) Groups(ctx context.Context, actorID, organizationID string) ([]Group, error) {
	resource, err := service.organizationResource(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManageMembers); err != nil {
		return nil, err
	}
	rows, err := service.pool.Query(ctx, "SELECT id,organization_id,name,created_at FROM groups WHERE organization_id=$1 ORDER BY name,id", resource.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Group{}
	for rows.Next() {
		var value Group
		if err := rows.Scan(&value.ID, &value.OrganizationID, &value.Name, &value.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) CreateGroup(ctx context.Context, actorID, organizationID, name string) (Group, error) {
	resource, err := service.organizationResource(ctx, organizationID)
	if err != nil || !validName(name) {
		if err != nil {
			return Group{}, err
		}
		return Group{}, ErrInvalidRequest
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManageMembers); err != nil {
		return Group{}, err
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return Group{}, err
	}
	id := uuid.New()
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback(ctx)
	var result Group
	if err = tx.QueryRow(ctx, `INSERT INTO groups(id,organization_id,name) VALUES($1,$2,$3)
		RETURNING id,organization_id,name,created_at`, id, resource.OrganizationID, strings.TrimSpace(name)).
		Scan(&result.ID, &result.OrganizationID, &result.Name, &result.CreatedAt); err != nil {
		return Group{}, mapConstraint(err)
	}
	if err = appendAudit(ctx, tx, "group.created", resource.OrganizationID, nil, actor, id, map[string]any{"name": result.Name}); err != nil {
		return Group{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Group{}, err
	}
	return result, nil
}

func (service *Service) DeleteGroup(ctx context.Context, actorID, groupID string) error {
	id, err := uuid.Parse(groupID)
	if err != nil {
		return ErrNotFound
	}
	var organizationID uuid.UUID
	if err = service.pool.QueryRow(ctx, "SELECT organization_id FROM groups WHERE id=$1", id).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	resource := permissions.Resource{OrganizationID: organizationID}
	if err := service.require(ctx, actorID, resource, permissions.CanManageMembers); err != nil {
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
	if _, err = tx.Exec(ctx, "DELETE FROM groups WHERE id=$1", id); err != nil {
		return mapConstraint(err)
	}
	if err = appendAudit(ctx, tx, "group.deleted", organizationID, nil, actor, id, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (service *Service) Members(ctx context.Context, actorID, groupID string) ([]Member, error) {
	group, resource, err := service.groupResource(ctx, groupID)
	if err != nil {
		return nil, err
	}
	_ = group
	if err := service.require(ctx, actorID, resource, permissions.CanManageMembers); err != nil {
		return nil, err
	}
	rows, err := service.pool.Query(ctx, `SELECT u.id,u.display_name,u.email FROM group_members gm
		JOIN users u ON u.id=gm.user_id WHERE gm.group_id=$1 ORDER BY u.display_name,u.id`, uuid.MustParse(groupID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Member{}
	for rows.Next() {
		var value Member
		if err := rows.Scan(&value.ID, &value.DisplayName, &value.Email); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (service *Service) AddMember(ctx context.Context, actorID, groupID, userID string) error {
	group, resource, err := service.groupResource(ctx, groupID)
	if err != nil {
		return err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManageMembers); err != nil {
		return err
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return err
	}
	user, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidRequest
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, "INSERT INTO group_members(group_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING", group, user)
	if err != nil {
		return mapConstraint(err)
	}
	if result.RowsAffected() > 0 {
		if err = appendAudit(ctx, tx, "group.member.added", resource.OrganizationID, nil, actor, group, map[string]any{"userId": user}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (service *Service) RemoveMember(ctx context.Context, actorID, groupID, userID string) error {
	group, resource, err := service.groupResource(ctx, groupID)
	if err != nil {
		return err
	}
	if err := service.require(ctx, actorID, resource, permissions.CanManageMembers); err != nil {
		return err
	}
	actor, err := parseActor(actorID)
	if err != nil {
		return err
	}
	user, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidRequest
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, "DELETE FROM group_members WHERE group_id=$1 AND user_id=$2", group, user)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err = appendAudit(ctx, tx, "group.member.removed", resource.OrganizationID, nil, actor, group, map[string]any{"userId": user}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (service *Service) groupResource(ctx context.Context, groupID string) (uuid.UUID, permissions.Resource, error) {
	id, err := uuid.Parse(groupID)
	if err != nil {
		return uuid.Nil, permissions.Resource{}, ErrNotFound
	}
	var organizationID uuid.UUID
	if err := service.pool.QueryRow(ctx, "SELECT organization_id FROM groups WHERE id=$1", id).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, permissions.Resource{}, ErrNotFound
	} else if err != nil {
		return uuid.Nil, permissions.Resource{}, err
	}
	return id, permissions.Resource{OrganizationID: organizationID}, nil
}
