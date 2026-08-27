package permissions

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidSubject = errors.New("invalid authorization subject")

type Resource struct {
	OrganizationID uuid.UUID
	WorkspaceID    uuid.UUID
	FolderID       *uuid.UUID
	DocumentID     *uuid.UUID
}

type Authorizer struct{ pool *pgxpool.Pool }

func NewAuthorizer(pool *pgxpool.Pool) *Authorizer { return &Authorizer{pool: pool} }

func (authorizer *Authorizer) Capabilities(ctx context.Context, actorID string, resource Resource) (map[Capability]bool, error) {
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return nil, ErrInvalidSubject
	}
	rows, err := authorizer.pool.Query(ctx, `
WITH RECURSIVE ancestors(id,parent_id) AS (
  SELECT id,parent_id FROM folders WHERE id=$4
  UNION ALL SELECT f.id,f.parent_id FROM folders f JOIN ancestors a ON f.id=a.parent_id
)
SELECT DISTINCT g.role::text
FROM authorization_grants g
WHERE g.organization_id=$2
  AND (g.user_id=$1 OR g.group_id IN (SELECT group_id FROM group_members WHERE user_id=$1))
  AND (
    g.scope_type='organization'
    OR (g.scope_type='workspace' AND g.workspace_id=$3)
    OR (g.scope_type='folder' AND g.folder_id IN (SELECT id FROM ancestors))
    OR (g.scope_type='document' AND g.document_id=$5)
  )`, actor, resource.OrganizationID, resource.WorkspaceID, resource.FolderID, resource.DocumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := make([]Role, 0)
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return Capabilities(roles...), rows.Err()
}

func (authorizer *Authorizer) Allows(ctx context.Context, actorID string, resource Resource, capability Capability) (bool, error) {
	capabilities, err := authorizer.Capabilities(ctx, actorID, resource)
	return capabilities[capability], err
}
