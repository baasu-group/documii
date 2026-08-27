package organizations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrForbidden = errors.New("organization operation forbidden")
var ErrNotFound = errors.New("organization not found")
var ErrInvalid = errors.New("invalid organization request")
var ErrConflict = errors.New("organization conflict")

type Service struct {
	pool           *pgxpool.Pool
	platformGroups map[string]bool
	development    bool
	publicURL      string
}

func NewService(pool *pgxpool.Pool, groups []string, development bool, publicURL string) *Service {
	g := map[string]bool{}
	for _, value := range groups {
		g[value] = true
	}
	return &Service{pool: pool, platformGroups: g, development: development, publicURL: strings.TrimRight(publicURL, "/")}
}

type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	ShareTitles bool      `json:"shareDocumentTitlesWithPlatformAdmins"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type Membership struct {
	UserID      string    `json:"userId"`
	Email       string    `json:"email,omitempty"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	JoinedAt    time.Time `json:"joinedAt"`
}
type Invitation struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expiresAt"`
	CreatedAt      time.Time `json:"createdAt"`
	AcceptanceURL  string    `json:"acceptanceUrl,omitempty"`
}
type Dashboard struct {
	Organizations          int64 `json:"organizations,omitempty"`
	Members                int64 `json:"members"`
	Documents              int64 `json:"documents"`
	Versions               int64 `json:"versions"`
	StorageBytes           int64 `json:"storageBytes"`
	ExternalDocuments      int64 `json:"externalDocuments"`
	PendingInvitations     int64 `json:"pendingInvitations"`
	SuspendedOrganizations int64 `json:"suspendedOrganizations,omitempty"`
	ProviderErrors         int64 `json:"providerErrors,omitempty"`
	ReportFailures         int64 `json:"reportFailures,omitempty"`
}
type SessionOrganization struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}
type SessionContext struct {
	User                 platformauth.Subject  `json:"user"`
	PlatformAdmin        bool                  `json:"platformAdmin"`
	Organizations        []SessionOrganization `json:"organizations"`
	ActiveOrganizationID string                `json:"activeOrganizationId,omitempty"`
	Workspaces           []map[string]string   `json:"workspaces"`
	Capabilities         []string              `json:"platformCapabilities"`
}

func (s *Service) IsPlatformAdmin(ctx context.Context, subject platformauth.Subject) (bool, error) {
	for _, group := range subject.Groups {
		if s.platformGroups[group] {
			return true, nil
		}
	}
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM platform_authorization_grants WHERE issuer=$1 AND subject=$2)`, subject.Issuer, subject.ExternalSubject).Scan(&ok)
	return ok, err
}
func (s *Service) requireOrgAdmin(ctx context.Context, actorID, orgID string) error {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_memberships WHERE organization_id=$1 AND user_id=$2 AND status='active' AND role IN ('owner','admin'))`, orgID, actorID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
func (s *Service) Session(ctx context.Context, subject platformauth.Subject) (SessionContext, error) {
	admin, err := s.IsPlatformAdmin(ctx, subject)
	if err != nil {
		return SessionContext{}, err
	}
	result := SessionContext{User: subject, PlatformAdmin: admin, Organizations: []SessionOrganization{}, Workspaces: []map[string]string{}}
	rows, err := s.pool.Query(ctx, `SELECT o.id,o.name,m.role::text,o.status FROM organization_memberships m JOIN organizations o ON o.id=m.organization_id WHERE m.user_id=$1 AND m.status='active' ORDER BY o.name`, subject.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var v SessionOrganization
		if err := rows.Scan(&v.ID, &v.Name, &v.Role, &v.Status); err != nil {
			return result, err
		}
		result.Organizations = append(result.Organizations, v)
		if result.ActiveOrganizationID == "" && v.Status == "active" {
			result.ActiveOrganizationID = v.ID
		}
	}
	if result.ActiveOrganizationID != "" {
		wr, err := s.pool.Query(ctx, `SELECT id,name FROM workspaces WHERE organization_id=$1 ORDER BY name`, result.ActiveOrganizationID)
		if err != nil {
			return result, err
		}
		defer wr.Close()
		for wr.Next() {
			var id, name string
			if err := wr.Scan(&id, &name); err != nil {
				return result, err
			}
			result.Workspaces = append(result.Workspaces, map[string]string{"id": id, "organizationId": result.ActiveOrganizationID, "name": name, "mode": "server"})
		}
	}
	if admin {
		result.Capabilities = []string{"manage_platform", "inspect_document_metadata"}
	}
	return result, rows.Err()
}
func (s *Service) List(ctx context.Context, subject platformauth.Subject) ([]Organization, error) {
	ok, err := s.IsPlatformAdmin(ctx, subject)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	rows, err := s.pool.Query(ctx, `SELECT o.id,o.name,o.status,COALESCE(st.share_document_titles_with_platform_admins,false),o.created_at,o.updated_at FROM organizations o LEFT JOIN organization_settings st ON st.organization_id=o.id ORDER BY o.created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Organization{}
	for rows.Next() {
		var v Organization
		if err := rows.Scan(&v.ID, &v.Name, &v.Status, &v.ShareTitles, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func audit(ctx context.Context, tx pgx.Tx, event string, org, actor, resource uuid.UUID, data map[string]any) error {
	raw, _ := json.Marshal(data)
	_, err := tx.Exec(ctx, `INSERT INTO audit_events(id,event_type,organization_id,actor_id,resource_id,data) VALUES($1,$2,$3,$4,$5,$6)`, "evt_"+uuid.NewString(), event, org, actor, resource, raw)
	return err
}
func (s *Service) Create(ctx context.Context, subject platformauth.Subject, name, ownerEmail, idempotencyKey string) (Organization, Invitation, error) {
	ok, err := s.IsPlatformAdmin(ctx, subject)
	if err != nil || !ok {
		if err == nil {
			err = ErrForbidden
		}
		return Organization{}, Invitation{}, err
	}
	name = strings.TrimSpace(name)
	ownerEmail = strings.ToLower(strings.TrimSpace(ownerEmail))
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if name == "" || !strings.Contains(ownerEmail, "@") || idempotencyKey == "" || len(idempotencyKey) > 255 {
		return Organization{}, Invitation{}, ErrInvalid
	}
	actor, err := uuid.Parse(subject.ID)
	if err != nil {
		return Organization{}, Invitation{}, ErrForbidden
	}
	var existingID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT resource_id FROM idempotency_keys WHERE actor_id=$1 AND key=$2 AND operation='organization.create'`, actor, idempotencyKey).Scan(&existingID); err == nil {
		value, getErr := s.get(ctx, existingID.String())
		return value, Invitation{}, getErr
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, Invitation{}, err
	}
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return Organization{}, Invitation{}, err
	}
	hash := sha256.Sum256(token)
	orgID, workspaceID, inviteID := uuid.New(), uuid.New(), uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Organization{}, Invitation{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO organizations(id,name,status) VALUES($1,$2,'pending')`, orgID, name); err != nil {
		return Organization{}, Invitation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO organization_settings(organization_id) VALUES($1)`, orgID); err != nil {
		return Organization{}, Invitation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO workspaces(id,organization_id,name) VALUES($1,$2,'General')`, workspaceID, orgID); err != nil {
		return Organization{}, Invitation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO organization_invitations(id,organization_id,email,role,token_hash,expires_at,created_by) VALUES($1,$2,$3,'owner',$4,now()+interval '7 days',$5)`, inviteID, orgID, ownerEmail, hash[:], actor); err != nil {
		return Organization{}, Invitation{}, err
	}
	if err = audit(ctx, tx, "organization.created", orgID, actor, orgID, map[string]any{"status": "pending"}); err != nil {
		return Organization{}, Invitation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,'organization.create',$1)`, orgID, actor, idempotencyKey); err != nil {
		return Organization{}, Invitation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Organization{}, Invitation{}, err
	}
	now := time.Now().UTC()
	inv := Invitation{ID: inviteID.String(), OrganizationID: orgID.String(), Email: ownerEmail, Role: "owner", Status: "pending", ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now}
	if s.development {
		inv.AcceptanceURL = s.publicURL + "/invitations/" + hex.EncodeToString(token)
	}
	return Organization{ID: orgID.String(), Name: name, Status: "pending", CreatedAt: now, UpdatedAt: now}, inv, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, subject platformauth.Subject, token string) (SessionOrganization, error) {
	if !subject.EmailVerified || subject.Email == "" {
		return SessionOrganization{}, ErrForbidden
	}
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return SessionOrganization{}, ErrNotFound
	}
	hash := sha256.Sum256(decoded)
	actor, err := uuid.Parse(subject.ID)
	if err != nil {
		return SessionOrganization{}, ErrForbidden
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionOrganization{}, err
	}
	defer tx.Rollback(ctx)
	var invitationID, orgID uuid.UUID
	var role, name, email, status string
	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT i.id,i.organization_id,i.role::text,o.name,i.email,i.status,i.expires_at FROM organization_invitations i JOIN organizations o ON o.id=i.organization_id WHERE i.token_hash=$1 FOR UPDATE`, hash[:]).Scan(&invitationID, &orgID, &role, &name, &email, &status, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionOrganization{}, ErrNotFound
	}
	if err != nil {
		return SessionOrganization{}, err
	}
	if status != "pending" || time.Now().After(expires) {
		return SessionOrganization{}, ErrConflict
	}
	if !strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(subject.Email)) {
		return SessionOrganization{}, ErrForbidden
	}
	if _, err = tx.Exec(ctx, `INSERT INTO organization_memberships(organization_id,user_id,role,status) VALUES($1,$2,$3,'active') ON CONFLICT(organization_id,user_id) DO UPDATE SET role=$3,status='active',updated_at=now()`, orgID, actor, role); err != nil {
		return SessionOrganization{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO authorization_grants(id,organization_id,principal_type,user_id,scope_type,role,created_by) SELECT $1,$2,'user',$3,'organization',$4,$3 WHERE NOT EXISTS(SELECT 1 FROM authorization_grants WHERE organization_id=$2 AND user_id=$3 AND scope_type='organization')`, uuid.New(), orgID, actor, role); err != nil {
		return SessionOrganization{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE organization_invitations SET status='accepted',accepted_by=$2,updated_at=now() WHERE id=$1`, invitationID, actor); err != nil {
		return SessionOrganization{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE organizations SET status='active',updated_at=now() WHERE id=$1 AND status='pending'`, orgID); err != nil {
		return SessionOrganization{}, err
	}
	if err = audit(ctx, tx, "organization.invitation.accepted", orgID, actor, invitationID, map[string]any{"role": role}); err != nil {
		return SessionOrganization{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SessionOrganization{}, err
	}
	return SessionOrganization{ID: orgID.String(), Name: name, Role: role, Status: "active"}, nil
}
func (s *Service) Patch(ctx context.Context, subject platformauth.Subject, id string, status *string, share *bool) (Organization, error) {
	admin, err := s.IsPlatformAdmin(ctx, subject)
	if err != nil || !admin {
		if err == nil {
			err = ErrForbidden
		}
		return Organization{}, err
	}
	actor, _ := uuid.Parse(subject.ID)
	org, err := uuid.Parse(id)
	if err != nil {
		return Organization{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Organization{}, err
	}
	defer tx.Rollback(ctx)
	if status != nil {
		if *status != "active" && *status != "suspended" {
			return Organization{}, ErrInvalid
		}
		_, err = tx.Exec(ctx, `UPDATE organizations SET status=$2,updated_at=now() WHERE id=$1`, org, *status)
	}
	if err == nil && share != nil {
		_, err = tx.Exec(ctx, `INSERT INTO organization_settings(organization_id,share_document_titles_with_platform_admins,updated_by) VALUES($1,$2,$3) ON CONFLICT(organization_id) DO UPDATE SET share_document_titles_with_platform_admins=$2,updated_by=$3,updated_at=now()`, org, *share, actor)
	}
	if err != nil {
		return Organization{}, err
	}
	if err = audit(ctx, tx, "organization.settings.updated", org, actor, org, map[string]any{"status": status, "shareTitles": share}); err != nil {
		return Organization{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Organization{}, err
	}
	return s.get(ctx, id)
}
func (s *Service) get(ctx context.Context, id string) (Organization, error) {
	var v Organization
	err := s.pool.QueryRow(ctx, `SELECT o.id,o.name,o.status,COALESCE(st.share_document_titles_with_platform_admins,false),o.created_at,o.updated_at FROM organizations o LEFT JOIN organization_settings st ON st.organization_id=o.id WHERE o.id=$1`, id).Scan(&v.ID, &v.Name, &v.Status, &v.ShareTitles, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *Service) Dashboard(ctx context.Context, subject platformauth.Subject, orgID string) (Dashboard, error) {
	if orgID == "" {
		ok, err := s.IsPlatformAdmin(ctx, subject)
		if err != nil || !ok {
			if err == nil {
				err = ErrForbidden
			}
			return Dashboard{}, err
		}
		var d Dashboard
		err = s.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE status='suspended'),(SELECT count(*) FROM organization_memberships WHERE status='active'),(SELECT count(*) FROM documents),(SELECT count(*) FROM document_versions),(SELECT COALESCE(sum(size),0) FROM storage_objects),(SELECT count(*) FROM report_runs WHERE status='failed') FROM organizations`).Scan(&d.Organizations, &d.SuspendedOrganizations, &d.Members, &d.Documents, &d.Versions, &d.StorageBytes, &d.ReportFailures)
		return d, err
	}
	if err := s.requireOrgAdmin(ctx, subject.ID, orgID); err != nil {
		return Dashboard{}, err
	}
	var d Dashboard
	err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM organization_memberships WHERE organization_id=$1 AND status='active'),(SELECT count(*) FROM documents WHERE organization_id=$1),(SELECT count(*) FROM document_versions v JOIN documents d ON d.id=v.document_id WHERE d.organization_id=$1),(SELECT COALESCE(sum(s.size),0) FROM storage_objects s JOIN document_versions v ON v.storage_object_id=s.id JOIN documents d ON d.id=v.document_id WHERE d.organization_id=$1),(SELECT count(*) FROM documents WHERE organization_id=$1 AND storage_mode='external'),(SELECT count(*) FROM organization_invitations WHERE organization_id=$1 AND status='pending')`, orgID).Scan(&d.Members, &d.Documents, &d.Versions, &d.StorageBytes, &d.ExternalDocuments, &d.PendingInvitations)
	return d, err
}
