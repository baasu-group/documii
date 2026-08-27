package administration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("administration resource not found")
	ErrForbidden      = errors.New("administration operation forbidden")
	ErrInvalidRequest = errors.New("invalid administration request")
	ErrConflict       = errors.New("administration conflict")
)

type Folder struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	WorkspaceID    string    `json:"workspaceId"`
	Name           string    `json:"name"`
	ParentID       *string   `json:"parentId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Group struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Member struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Email       *string `json:"email,omitempty"`
}

type Grant struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	PrincipalType  string    `json:"principalType"`
	PrincipalID    string    `json:"principalId"`
	ScopeType      string    `json:"scopeType"`
	ScopeID        string    `json:"scopeId"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"createdAt"`
}

type EditorPolicy struct {
	OrganizationID string    `json:"organizationId"`
	Family         string    `json:"family"`
	ProviderID     string    `json:"providerId"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AuditEvent struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	OrganizationID string         `json:"organizationId"`
	WorkspaceID    *string        `json:"workspaceId,omitempty"`
	ActorID        *string        `json:"actorId,omitempty"`
	ResourceID     *string        `json:"resourceId,omitempty"`
	RequestID      *string        `json:"requestId,omitempty"`
	Data           map[string]any `json:"data"`
	Timestamp      time.Time      `json:"timestamp"`
}

type AuditPage struct {
	Events     []AuditEvent `json:"events"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

type PolicyValidator func(context.Context, string, string) error

type Service struct {
	pool           *pgxpool.Pool
	authorizer     *permissions.Authorizer
	validatePolicy PolicyValidator
}

func NewService(pool *pgxpool.Pool, authorizer *permissions.Authorizer, validatePolicy PolicyValidator) *Service {
	return &Service{pool: pool, authorizer: authorizer, validatePolicy: validatePolicy}
}

func validName(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && utf8.RuneCountInString(trimmed) <= 255
}

func parseActor(value string) (uuid.UUID, error) {
	actor, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, ErrForbidden
	}
	return actor, nil
}

func (service *Service) workspaceResource(ctx context.Context, workspaceID string) (permissions.Resource, error) {
	id, err := uuid.Parse(workspaceID)
	if err != nil {
		return permissions.Resource{}, ErrNotFound
	}
	var organizationID uuid.UUID
	if err := service.pool.QueryRow(ctx, "SELECT organization_id FROM workspaces WHERE id=$1", id).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) {
		return permissions.Resource{}, ErrNotFound
	} else if err != nil {
		return permissions.Resource{}, err
	}
	return permissions.Resource{OrganizationID: organizationID, WorkspaceID: id}, nil
}

func (service *Service) organizationResource(ctx context.Context, organizationID string) (permissions.Resource, error) {
	id, err := uuid.Parse(organizationID)
	if err != nil {
		return permissions.Resource{}, ErrNotFound
	}
	var exists bool
	if err := service.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)", id).Scan(&exists); err != nil {
		return permissions.Resource{}, err
	}
	if !exists {
		return permissions.Resource{}, ErrNotFound
	}
	return permissions.Resource{OrganizationID: id}, nil
}

func (service *Service) require(ctx context.Context, actorID string, resource permissions.Resource, capability permissions.Capability) error {
	allowed, err := service.authorizer.Allows(ctx, actorID, resource, capability)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, event string, organizationID uuid.UUID, workspaceID *uuid.UUID, actorID, resourceID uuid.UUID, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(id,event_type,organization_id,workspace_id,actor_id,resource_id,data)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, fmt.Sprintf("evt_%s", uuid.New()), event, organizationID, workspaceID, actorID, resourceID, encoded)
	return err
}

type auditCursor struct {
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
}

func encodeCursor(value auditCursor) string {
	encoded, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeCursor(value string) (auditCursor, error) {
	if value == "" {
		return auditCursor{Timestamp: time.Now().UTC().Add(time.Second), ID: "~"}, nil
	}
	encoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return auditCursor{}, ErrInvalidRequest
	}
	var cursor auditCursor
	if json.Unmarshal(encoded, &cursor) != nil || cursor.Timestamp.IsZero() || cursor.ID == "" {
		return auditCursor{}, ErrInvalidRequest
	}
	return cursor, nil
}
