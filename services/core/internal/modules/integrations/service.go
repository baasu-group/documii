package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/documi/documi/services/core/internal/modules/documents"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var ErrForbidden = errors.New("integration forbidden")
var ErrNotFound = errors.New("integration not found")
var ErrInvalid = errors.New("invalid integration request")
var ErrAirGap = errors.New("integration blocked by air gap")

type Status struct {
	ID, Name, Version, State, Description string
	Capabilities                          []string
	AirGapCompatible                      bool
}

func (s Status) MarshalJSON() ([]byte, error) {
	type value struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		Version          string   `json:"version"`
		State            string   `json:"state"`
		Description      string   `json:"description"`
		Capabilities     []string `json:"capabilities"`
		AirGapCompatible bool     `json:"airGapCompatible"`
	}
	return json.Marshal(value{s.ID, s.Name, s.Version, s.State, s.Description, s.Capabilities, s.AirGapCompatible})
}

type Connection struct {
	ID, IntegrationID, ConnectionType, Status string
	OrganizationID, UserID                    *string
	SafeMetadata                              map[string]any
	CreatedAt, UpdatedAt                      time.Time
}

func (c Connection) MarshalJSON() ([]byte, error) {
	type value struct {
		ID             string         `json:"id"`
		IntegrationID  string         `json:"integrationId"`
		OrganizationID *string        `json:"organizationId,omitempty"`
		UserID         *string        `json:"userId,omitempty"`
		ConnectionType string         `json:"connectionType"`
		Status         string         `json:"status"`
		SafeMetadata   map[string]any `json:"safeMetadata"`
		CreatedAt      time.Time      `json:"createdAt"`
		UpdatedAt      time.Time      `json:"updatedAt"`
	}
	return json.Marshal(value{c.ID, c.IntegrationID, c.OrganizationID, c.UserID, c.ConnectionType, c.Status, c.SafeMetadata, c.CreatedAt, c.UpdatedAt})
}

type PlatformAuthorizer func(context.Context, platformauth.Subject) (bool, error)
type Service struct {
	pool             *pgxpool.Pool
	airGapped        bool
	platform         PlatformAuthorizer
	googleConfigured bool
	google           *GoogleConnector
	documents        *documents.Service
}

func (s *Service) WithGoogle(connector *GoogleConnector) *Service    { s.google = connector; return s }
func (s *Service) WithDocuments(service *documents.Service) *Service { s.documents = service; return s }
func (s *Service) googleAvailable(ctx context.Context) error {
	if s.airGapped {
		return ErrAirGap
	}
	if s.google == nil {
		return ErrInvalid
	}
	var enabled bool
	if err := s.pool.QueryRow(ctx, `SELECT enabled FROM integration_installations WHERE id='google-workspace'`).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return ErrInvalid
	}
	return nil
}
func (s *Service) GoogleStart(ctx context.Context, subject platformauth.Subject, organizationID *string) (OAuthStart, error) {
	if err := s.googleAvailable(ctx); err != nil {
		return OAuthStart{}, err
	}
	if organizationID != nil {
		var allowed bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_memberships WHERE organization_id=$1 AND user_id=$2 AND status='active' AND role IN('owner','admin'))`, *organizationID, subject.ID).Scan(&allowed); err != nil {
			return OAuthStart{}, err
		}
		if !allowed {
			return OAuthStart{}, ErrForbidden
		}
	}
	return s.google.Start(ctx, subject.ID, organizationID)
}
func (s *Service) GoogleCallback(ctx context.Context, state, code string) (Connection, string, error) {
	if err := s.googleAvailable(ctx); err != nil {
		return Connection{}, "", err
	}
	return s.google.Callback(ctx, state, code)
}
func (s *Service) GooglePicker(ctx context.Context, subject platformauth.Subject, connectionID string) (PickerSession, error) {
	if err := s.googleAvailable(ctx); err != nil {
		return PickerSession{}, err
	}
	return s.google.Picker(ctx, subject, connectionID)
}

func (s *Service) GoogleDocument(ctx context.Context, subject platformauth.Subject, request GoogleDocumentRequest, create bool) (documents.Document, error) {
	if err := s.googleAvailable(ctx); err != nil {
		return documents.Document{}, err
	}
	if s.documents == nil {
		return documents.Document{}, ErrInvalid
	}
	return s.google.CreateReference(ctx, s.documents, subject, request, create)
}

func NewService(pool *pgxpool.Pool, airGapped, googleConfigured bool, platform PlatformAuthorizer) *Service {
	return &Service{pool: pool, airGapped: airGapped, googleConfigured: googleConfigured, platform: platform}
}

var manifests = map[string]Status{"native": {ID: "native", Name: "Native Documi Editor", Version: "1", Description: "Built-in Documi document editor.", Capabilities: []string{"browser", "desktop", "offline", "edit"}, AirGapCompatible: true}, "collabora": {ID: "collabora", Name: "Collabora Online", Version: "1", Description: "Browser Office editor through Documi WOPI.", Capabilities: []string{"browser", "collaborative", "edit"}, AirGapCompatible: true}, "libreoffice-desktop": {ID: "libreoffice-desktop", Name: "LibreOffice Desktop", Version: "1", Description: "Optional local desktop Office editor.", Capabilities: []string{"desktop", "offline", "external-process"}, AirGapCompatible: true}, "google-workspace": {ID: "google-workspace", Name: "Google Workspace", Version: "1", Description: "External Google Docs, Sheets, Slides and Drive connector.", Capabilities: []string{"external", "create", "import", "export"}, AirGapCompatible: false}}

func (s *Service) Statuses(ctx context.Context) ([]Status, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,version,enabled,status FROM integration_installations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Status{}
	for rows.Next() {
		var id, version, state string
		var enabled bool
		if err := rows.Scan(&id, &version, &enabled, &state); err != nil {
			return nil, err
		}
		manifest, ok := manifests[id]
		if !ok {
			continue
		}
		manifest.Version = version
		if id == "google-workspace" && s.airGapped {
			manifest.State = "blocked-by-air-gap"
		} else if id == "google-workspace" && enabled && !s.googleConfigured {
			manifest.State = "setup-required"
		} else if !enabled {
			manifest.State = "disabled"
		} else {
			manifest.State = state
		}
		result = append(result, manifest)
	}
	return result, rows.Err()
}
func (s *Service) Update(ctx context.Context, subject platformauth.Subject, id string, enabled bool) (Status, error) {
	admin, err := s.platform(ctx, subject)
	if err != nil {
		return Status{}, err
	}
	if !admin {
		return Status{}, ErrForbidden
	}
	manifest, ok := manifests[id]
	if !ok {
		return Status{}, ErrNotFound
	}
	if id == "google-workspace" && enabled && s.airGapped {
		return Status{}, ErrAirGap
	}
	state := "disabled"
	if enabled {
		state = "installed"
		if id == "google-workspace" && !s.googleConfigured {
			state = "setup-required"
		}
	}
	actor, err := uuid.Parse(subject.ID)
	if err != nil {
		return Status{}, ErrForbidden
	}
	result, err := s.pool.Exec(ctx, `UPDATE integration_installations SET enabled=$2,status=$3,updated_by=$4,updated_at=now() WHERE id=$1`, id, enabled, state, actor)
	if err != nil {
		return Status{}, err
	}
	if result.RowsAffected() == 0 {
		return Status{}, ErrNotFound
	}
	manifest.State = state
	return manifest, nil
}
func (s *Service) Connections(ctx context.Context, actor, org string) ([]Connection, error) {
	var allowed bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_memberships WHERE organization_id=$1 AND user_id=$2 AND status='active' AND role IN('owner','admin'))`, org, actor).Scan(&allowed); err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	rows, err := s.pool.Query(ctx, `SELECT id,integration_id,organization_id,user_id,connection_type,status,safe_metadata,created_at,updated_at FROM integration_connections WHERE organization_id=$1 ORDER BY created_at DESC LIMIT 200`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Connection{}
	for rows.Next() {
		var value Connection
		var metadata []byte
		if err := rows.Scan(&value.ID, &value.IntegrationID, &value.OrganizationID, &value.UserID, &value.ConnectionType, &value.Status, &metadata, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &value.SafeMetadata)
		result = append(result, value)
	}
	return result, rows.Err()
}
func (s *Service) DeleteConnection(ctx context.Context, actor, id string) error {
	connection, err := uuid.Parse(id)
	if err != nil {
		return ErrNotFound
	}
	actorID, err := uuid.Parse(actor)
	if err != nil {
		return ErrForbidden
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var org uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT organization_id FROM integration_connections WHERE id=$1 FOR UPDATE`, connection).Scan(&org); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	var allowed bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_memberships WHERE organization_id=$1 AND user_id=$2 AND status='active' AND role IN('owner','admin'))`, org, actorID).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	if _, err = tx.Exec(ctx, `DELETE FROM integration_connections WHERE id=$1`, connection); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(id,event_type,organization_id,actor_id,resource_id) VALUES($1,'integration.connection.deleted',$2,$3,$4)`, "evt_"+uuid.NewString(), org, actorID, connection); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
