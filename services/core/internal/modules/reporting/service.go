package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/documi/documi/services/core/internal/modules/documents"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrForbidden = errors.New("report forbidden")
var ErrInvalid = errors.New("invalid report request")
var ErrNotFound = errors.New("report not found")

type Definition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Formats     []string `json:"formats"`
}

var definitions = []Definition{{"organization-document-storage", "Document and storage usage", "Documents, versions, families, providers, and managed storage.", []string{"csv", "xlsx", "pdf"}}, {"membership-access-review", "Membership and access review", "Members, roles, membership state, and direct/group grant counts.", []string{"csv", "xlsx", "pdf"}}, {"audit-activity", "Audit and activity", "Append-only activity summarized by event type.", []string{"csv", "xlsx", "pdf"}}, {"external-editor-risk", "External document and editor risk", "External references and editor-provider usage.", []string{"csv", "xlsx", "pdf"}}}

type Run struct {
	ID, OrganizationID, DefinitionID, Format, Status string
	WorkspaceID                                      *string
	Filters                                          map[string]any
	DocumentID                                       *string
	ErrorCode                                        *string
	CreatedBy                                        string
	CreatedAt                                        time.Time
	CompletedAt                                      *time.Time
}

func (r Run) MarshalJSON() ([]byte, error) {
	type alias struct {
		ID             string         `json:"id"`
		OrganizationID string         `json:"organizationId"`
		WorkspaceID    *string        `json:"workspaceId,omitempty"`
		DefinitionID   string         `json:"definitionId"`
		Format         string         `json:"format"`
		Filters        map[string]any `json:"filters"`
		Status         string         `json:"status"`
		DocumentID     *string        `json:"documentId,omitempty"`
		ErrorCode      *string        `json:"errorCode,omitempty"`
		CreatedBy      string         `json:"createdBy"`
		CreatedAt      time.Time      `json:"createdAt"`
		CompletedAt    *time.Time     `json:"completedAt,omitempty"`
	}
	return json.Marshal(alias{r.ID, r.OrganizationID, r.WorkspaceID, r.DefinitionID, r.Format, r.Filters, r.Status, r.DocumentID, r.ErrorCode, r.CreatedBy, r.CreatedAt, r.CompletedAt})
}

type Service struct {
	pool      *pgxpool.Pool
	documents *documents.Service
	renderers map[string]Renderer
}

func NewService(pool *pgxpool.Pool, documentService *documents.Service) *Service {
	return &Service{pool: pool, documents: documentService, renderers: map[string]Renderer{"csv": CSVRenderer{}, "xlsx": XLSXRenderer{}, "pdf": PDFRenderer{}}}
}
func (s *Service) Definitions() []Definition { return append([]Definition(nil), definitions...) }
func definition(id string) (Definition, bool) {
	for _, value := range definitions {
		if value.ID == id {
			return value, true
		}
	}
	return Definition{}, false
}
func (s *Service) requireAdmin(ctx context.Context, actor, org string) error {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_memberships m JOIN organizations o ON o.id=m.organization_id WHERE m.organization_id=$1 AND m.user_id=$2 AND m.status='active' AND o.status='active' AND m.role IN ('owner','admin'))`, org, actor).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
func validFilters(filters map[string]any) bool {
	allowed := map[string]bool{"workspaceId": true, "family": true, "editorProvider": true, "storageMode": true, "memberId": true, "eventType": true, "dateFrom": true, "dateTo": true}
	for key, value := range filters {
		if !allowed[key] {
			return false
		}
		if _, ok := value.(string); !ok {
			return false
		}
	}
	return true
}
func (s *Service) Create(ctx context.Context, actor, org, definitionID, format, idempotencyKey string, filters map[string]any) (Run, error) {
	if err := s.requireAdmin(ctx, actor, org); err != nil {
		return Run{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if _, ok := definition(definitionID); !ok || idempotencyKey == "" || len(idempotencyKey) > 255 {
		return Run{}, ErrInvalid
	}
	if s.renderers[format] == nil || !validFilters(filters) {
		return Run{}, ErrInvalid
	}
	organizationID, err := uuid.Parse(org)
	if err != nil {
		return Run{}, ErrNotFound
	}
	actorID, err := uuid.Parse(actor)
	if err != nil {
		return Run{}, ErrForbidden
	}
	var existing uuid.UUID
	if err = s.pool.QueryRow(ctx, `SELECT resource_id FROM idempotency_keys WHERE organization_id=$1 AND actor_id=$2 AND key=$3 AND operation='report.run.create'`, organizationID, actorID, idempotencyKey).Scan(&existing); err == nil {
		return s.Get(ctx, actor, existing.String())
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Run{}, err
	}
	var workspaceID *uuid.UUID
	if value, _ := filters["workspaceId"].(string); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return Run{}, ErrInvalid
		}
		var belongs bool
		if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1 AND organization_id=$2)`, parsed, organizationID).Scan(&belongs); err != nil || !belongs {
			if err != nil {
				return Run{}, err
			}
			return Run{}, ErrInvalid
		}
		workspaceID = &parsed
	}
	encoded, _ := json.Marshal(filters)
	id := uuid.New()
	now := time.Now().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO report_runs(id,organization_id,workspace_id,definition_id,format,filters,status,created_by,completed_at) VALUES($1,$2,$3,$4,$5,$6,'completed',$7,$8)`, id, organizationID, workspaceID, definitionID, format, encoded, actorID, now); err != nil {
		return Run{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(organization_id,actor_id,key,operation,resource_id) VALUES($1,$2,$3,'report.run.create',$4)`, organizationID, actorID, idempotencyKey, id); err != nil {
		return Run{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(id,event_type,organization_id,workspace_id,actor_id,resource_id) VALUES($1,'report.run.created',$2,$3,$4,$5)`, "evt_"+uuid.NewString(), organizationID, workspaceID, actorID, id); err != nil {
		return Run{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	return Run{ID: id.String(), OrganizationID: org, WorkspaceID: stringPtr(workspaceID), DefinitionID: definitionID, Format: format, Filters: filters, Status: "completed", CreatedBy: actor, CreatedAt: now, CompletedAt: &now}, nil
}
func stringPtr(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	v := value.String()
	return &v
}
func (s *Service) Get(ctx context.Context, actor, id string) (Run, error) {
	var run Run
	var filters []byte
	err := s.pool.QueryRow(ctx, `SELECT id,organization_id,workspace_id,definition_id,format,filters,status,document_id,error_code,created_by,created_at,completed_at FROM report_runs WHERE id=$1`, id).Scan(&run.ID, &run.OrganizationID, &run.WorkspaceID, &run.DefinitionID, &run.Format, &filters, &run.Status, &run.DocumentID, &run.ErrorCode, &run.CreatedBy, &run.CreatedAt, &run.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, err
	}
	if err = s.requireAdmin(ctx, actor, run.OrganizationID); err != nil {
		return Run{}, err
	}
	_ = json.Unmarshal(filters, &run.Filters)
	return run, nil
}
func (s *Service) List(ctx context.Context, actor, org string) ([]Run, error) {
	if err := s.requireAdmin(ctx, actor, org); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,organization_id,workspace_id,definition_id,format,filters,status,document_id,error_code,created_by,created_at,completed_at FROM report_runs WHERE organization_id=$1 ORDER BY created_at DESC,id DESC LIMIT 100`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Run{}
	for rows.Next() {
		var run Run
		var filters []byte
		if err := rows.Scan(&run.ID, &run.OrganizationID, &run.WorkspaceID, &run.DefinitionID, &run.Format, &filters, &run.Status, &run.DocumentID, &run.ErrorCode, &run.CreatedBy, &run.CreatedAt, &run.CompletedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(filters, &run.Filters)
		result = append(result, run)
	}
	return result, rows.Err()
}
func (s *Service) dataset(ctx context.Context, run Run) (Dataset, error) {
	switch run.DefinitionID {
	case "organization-document-storage":
		return s.documentUsage(ctx, run)
	case "membership-access-review":
		return s.membership(ctx, run)
	case "audit-activity":
		return s.activity(ctx, run)
	case "external-editor-risk":
		return s.risk(ctx, run)
	default:
		return Dataset{}, ErrInvalid
	}
}
func (s *Service) documentUsage(ctx context.Context, run Run) (Dataset, error) {
	rows, err := s.pool.Query(ctx, `SELECT d.family,d.editor_provider,d.storage_mode,count(DISTINCT d.id),count(v.id),COALESCE(sum(o.size),0) FROM documents d JOIN document_versions v ON v.document_id=d.id JOIN storage_objects o ON o.id=v.storage_object_id WHERE d.organization_id=$1 AND ($2::uuid IS NULL OR d.workspace_id=$2) GROUP BY d.family,d.editor_provider,d.storage_mode ORDER BY d.family,d.editor_provider`, run.OrganizationID, run.WorkspaceID)
	if err != nil {
		return Dataset{}, err
	}
	defer rows.Close()
	data := Dataset{Title: "Document and storage usage", Headers: []string{"Family", "Editor provider", "Storage mode", "Documents", "Versions", "Bytes"}}
	for rows.Next() {
		var a, b, c string
		var d, e, f int64
		if err := rows.Scan(&a, &b, &c, &d, &e, &f); err != nil {
			return Dataset{}, err
		}
		data.Rows = append(data.Rows, []string{a, b, c, strconv.FormatInt(d, 10), strconv.FormatInt(e, 10), strconv.FormatInt(f, 10)})
	}
	return data, rows.Err()
}
func (s *Service) membership(ctx context.Context, run Run) (Dataset, error) {
	rows, err := s.pool.Query(ctx, `SELECT u.display_name,COALESCE(u.email,''),m.role::text,m.status,(SELECT count(*) FROM authorization_grants g WHERE g.organization_id=m.organization_id AND g.user_id=m.user_id),(SELECT count(*) FROM group_members gm JOIN groups g ON g.id=gm.group_id WHERE g.organization_id=m.organization_id AND gm.user_id=m.user_id) FROM organization_memberships m JOIN users u ON u.id=m.user_id WHERE m.organization_id=$1 ORDER BY u.display_name LIMIT 10000`, run.OrganizationID)
	if err != nil {
		return Dataset{}, err
	}
	defer rows.Close()
	data := Dataset{Title: "Membership and access review", Headers: []string{"Member", "Email", "Role", "Status", "Direct grants", "Groups"}}
	for rows.Next() {
		var a, b, c, d string
		var e, f int64
		if err := rows.Scan(&a, &b, &c, &d, &e, &f); err != nil {
			return Dataset{}, err
		}
		data.Rows = append(data.Rows, []string{a, b, c, d, strconv.FormatInt(e, 10), strconv.FormatInt(f, 10)})
	}
	return data, rows.Err()
}
func (s *Service) activity(ctx context.Context, run Run) (Dataset, error) {
	event, _ := run.Filters["eventType"].(string)
	rows, err := s.pool.Query(ctx, `SELECT event_type,count(*),min(created_at),max(created_at) FROM audit_events WHERE organization_id=$1 AND ($2='' OR event_type=$2) GROUP BY event_type ORDER BY count(*) DESC,event_type LIMIT 10000`, run.OrganizationID, event)
	if err != nil {
		return Dataset{}, err
	}
	defer rows.Close()
	data := Dataset{Title: "Audit and activity", Headers: []string{"Event type", "Count", "First event", "Latest event"}}
	for rows.Next() {
		var event string
		var count int64
		var first, last time.Time
		if err := rows.Scan(&event, &count, &first, &last); err != nil {
			return Dataset{}, err
		}
		data.Rows = append(data.Rows, []string{event, strconv.FormatInt(count, 10), first.UTC().Format(time.RFC3339), last.UTC().Format(time.RFC3339)})
	}
	return data, rows.Err()
}
func (s *Service) risk(ctx context.Context, run Run) (Dataset, error) {
	rows, err := s.pool.Query(ctx, `SELECT editor_provider,storage_mode,count(*),COALESCE(sum(CASE WHEN external_url IS NOT NULL THEN 1 ELSE 0 END),0) FROM documents WHERE organization_id=$1 GROUP BY editor_provider,storage_mode ORDER BY count(*) DESC`, run.OrganizationID)
	if err != nil {
		return Dataset{}, err
	}
	defer rows.Close()
	data := Dataset{Title: "External document and editor risk", Headers: []string{"Editor provider", "Storage mode", "Documents", "External links"}}
	for rows.Next() {
		var a, b string
		var c, d int64
		if err := rows.Scan(&a, &b, &c, &d); err != nil {
			return Dataset{}, err
		}
		data.Rows = append(data.Rows, []string{a, b, strconv.FormatInt(c, 10), strconv.FormatInt(d, 10)})
	}
	return data, rows.Err()
}
func (s *Service) Export(ctx context.Context, actor, runID, idempotencyKey string) (Run, error) {
	run, err := s.Get(ctx, actor, runID)
	if err != nil {
		return Run{}, err
	}
	if run.DocumentID != nil {
		return run, nil
	}
	data, err := s.dataset(ctx, run)
	if err != nil {
		return Run{}, err
	}
	rendered, err := s.renderers[run.Format].Render(data)
	if err != nil {
		return Run{}, err
	}
	workspace := run.WorkspaceID
	if workspace == nil {
		var value string
		if err = s.pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE organization_id=$1 ORDER BY created_at LIMIT 1`, run.OrganizationID).Scan(&value); err != nil {
			return Run{}, err
		}
		workspace = &value
	}
	definition, _ := definition(run.DefinitionID)
	document, err := s.documents.CreateGenerated(ctx, documents.GeneratedDocumentRequest{ActorID: actor, OrganizationID: run.OrganizationID, WorkspaceID: *workspace, IdempotencyKey: idempotencyKey, Title: definition.Name + " " + time.Now().UTC().Format("2006-01-02"), Family: rendered.Family, MIMEType: rendered.MIMEType, Extension: rendered.Extension, Content: rendered.Content})
	if err != nil {
		return Run{}, err
	}
	result, err := s.pool.Exec(ctx, `UPDATE report_runs SET document_id=$2,status='completed',completed_at=now() WHERE id=$1 AND document_id IS NULL`, run.ID, document.ID)
	if err != nil {
		return Run{}, err
	}
	if result.RowsAffected() == 0 {
		return s.Get(ctx, actor, runID)
	}
	return s.Get(ctx, actor, runID)
}
func DefinitionTitle(id string) string {
	value, ok := definition(id)
	if !ok {
		return strings.TrimSpace(id)
	}
	return value.Name
}
