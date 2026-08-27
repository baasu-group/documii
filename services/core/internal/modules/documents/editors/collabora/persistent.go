package collabora

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/documi/documi/services/core/internal/modules/documents"
	"github.com/documi/documi/services/core/internal/modules/documents/editors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PersistentGateway struct{ documents *documents.Service }

func NewPersistentGateway(service *documents.Service) *PersistentGateway {
	return &PersistentGateway{documents: service}
}

func (gateway *PersistentGateway) SessionAccess(ctx context.Context, sessionID, actorID, fileID string) (Access, error) {
	sessionCanEdit, err := gateway.documents.ValidateEditorSession(ctx, sessionID, actorID, fileID)
	if err != nil {
		return Access{}, mapDocumentError(err)
	}
	access, err := gateway.Access(ctx, actorID, fileID)
	access.CanEdit = access.CanEdit && sessionCanEdit
	return access, err
}

func (gateway *PersistentGateway) Access(ctx context.Context, actorID, fileID string) (Access, error) {
	value, err := gateway.documents.Access(ctx, actorID, fileID)
	return Access{CanRead: value.CanRead, CanEdit: value.CanEdit}, mapDocumentError(err)
}
func (gateway *PersistentGateway) Info(ctx context.Context, actorID, fileID string) (File, error) {
	value, err := gateway.documents.Get(ctx, actorID, fileID, false)
	if err != nil {
		return File{}, mapDocumentError(err)
	}
	return fileFromDocument(value), nil
}
func (gateway *PersistentGateway) OpenContent(ctx context.Context, actorID, fileID string) (io.ReadCloser, File, error) {
	value, reader, err := gateway.documents.OpenContent(ctx, actorID, fileID)
	if err != nil {
		return nil, File{}, mapDocumentError(err)
	}
	return reader, fileFromDocument(value), nil
}
func (gateway *PersistentGateway) SaveNewVersion(ctx context.Context, request SaveRequest) (File, error) {
	value, err := gateway.documents.SaveEditorContent(ctx, request.ActorID, request.FileID, request.ExpectedVersion, request.LockValue, "collabora", request.Content, request.ContentLength)
	if err != nil {
		return File{}, mapDocumentError(err)
	}
	return fileFromDocument(value), nil
}
func (gateway *PersistentGateway) ManageLock(ctx context.Context, request LockRequest) (LockResult, error) {
	operations := map[LockOperation]documents.EditorLockOperation{LockAcquire: documents.EditorLockAcquire, LockRefresh: documents.EditorLockRefresh, LockGet: documents.EditorLockGet, LockRelease: documents.EditorLockRelease, LockRelock: documents.EditorLockRelock}
	value, err := gateway.documents.ManageEditorLock(ctx, documents.EditorLockRequest{ActorID: request.ActorID, DocumentID: request.FileID, ExpectedVersion: request.ExpectedVersion, Value: request.LockValue, OldValue: request.OldLockValue, Operation: operations[request.Operation]})
	return LockResult{CurrentLock: value}, mapDocumentError(err)
}

func mapDocumentError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, documents.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, documents.ErrForbidden):
		return ErrForbidden
	case errors.Is(err, documents.ErrVersionConflict), errors.Is(err, documents.ErrLocked):
		return ErrConflict
	default:
		return err
	}
}
func fileFromDocument(value documents.Document) File {
	return File{ID: value.ID, Name: safeOfficeName(value.Title, value.MIMEType), VersionID: value.CurrentVersionID, Size: value.Size(), ContentType: value.MIMEType}
}
func safeOfficeName(title, mime string) string {
	extension := map[string]string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ".xlsx", "application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx", "application/vnd.oasis.opendocument.text": ".odt", "application/vnd.oasis.opendocument.spreadsheet": ".ods", "application/vnd.oasis.opendocument.presentation": ".odp"}[mime]
	name := strings.Map(func(character rune) rune {
		if character < ' ' || strings.ContainsRune(`<>:"/\|?*`, character) {
			return '-'
		}
		return character
	}, strings.TrimSpace(title))
	runes := []rune(name)
	if len(runes) > 180 {
		name = string(runes[:180])
	}
	name = strings.TrimRight(strings.TrimSuffix(name, filepath.Ext(name)), " .")
	if name == "" {
		name = "document"
	}
	upper := strings.ToUpper(name)
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" ||
		(len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9') {
		name = "_" + name
	}
	return name + extension
}

type PersistentSessionService struct {
	pool            *pgxpool.Pool
	documents       *documents.Service
	tokens          *TokenService
	discovery       *Discovery
	publicURL       *url.URL
	collaboraPublic *url.URL
}

func NewPersistentSessionService(pool *pgxpool.Pool, service *documents.Service, tokens *TokenService, discovery *Discovery, publicURL, collaboraPublic string) (*PersistentSessionService, error) {
	documi, err := url.Parse(publicURL)
	if err != nil {
		return nil, err
	}
	office, err := url.Parse(collaboraPublic)
	if err != nil {
		return nil, err
	}
	return &PersistentSessionService{pool: pool, documents: service, tokens: tokens, discovery: discovery, publicURL: documi, collaboraPublic: office}, nil
}
func (service *PersistentSessionService) CreateSession(ctx context.Context, request editors.SessionRequest) (editors.Session, error) {
	if request.ProviderID != editors.ProviderCollabora {
		return editors.Session{}, editors.ErrProviderUnavailable
	}
	access, err := service.documents.Access(ctx, request.ActorID, request.DocumentID)
	if err != nil || !access.CanRead {
		return editors.Session{}, editors.ErrSessionForbidden
	}
	document, err := service.documents.Get(ctx, request.ActorID, request.DocumentID, false)
	if err != nil {
		return editors.Session{}, err
	}
	mode := "view"
	canEdit := request.RequestedEdit && access.CanEdit
	if canEdit {
		mode = "edit"
	}
	extension := strings.TrimPrefix(filepath.Ext(safeOfficeName(document.Title, document.MIMEType)), ".")
	action, err := service.discovery.Action(ctx, extension, mode)
	if err != nil {
		return editors.Session{}, err
	}
	actionURL, err := url.Parse(action)
	if err != nil {
		return editors.Session{}, err
	}
	actionURL.Scheme = service.collaboraPublic.Scheme
	actionURL.Host = service.collaboraPublic.Host
	wopi := service.publicURL.ResolveReference(&url.URL{Path: "/api/v1/wopi/files/" + url.PathEscape(document.ID)})
	query := actionURL.Query()
	query.Set("WOPISrc", wopi.String())
	actionURL.RawQuery = query.Encode()
	sessionID := uuid.New()
	token, expires, err := service.tokens.IssueForSession(document.ID, document.CurrentVersionID, request.ActorID, sessionID.String(), true, canEdit)
	if err != nil {
		return editors.Session{}, err
	}
	actor := uuid.MustParse(request.ActorID)
	documentID := uuid.MustParse(document.ID)
	workspaceID := uuid.MustParse(document.WorkspaceID)
	transaction, err := service.pool.Begin(ctx)
	if err != nil {
		return editors.Session{}, err
	}
	defer transaction.Rollback(ctx)
	if _, err = transaction.Exec(ctx, "INSERT INTO editor_sessions(id,document_id,actor_id,provider_id,can_edit,expires_at) VALUES($1,$2,$3,'collabora',$4,$5)", sessionID, documentID, actor, canEdit, expires); err != nil {
		return editors.Session{}, err
	}
	for _, event := range []string{"document.editor.opened", "document.collabora.opened"} {
		if _, err = transaction.Exec(ctx, "INSERT INTO audit_events(id,event_type,organization_id,workspace_id,actor_id,resource_id) VALUES($1,$2,(SELECT organization_id FROM documents WHERE id=$3),$4,$5,$3)", fmt.Sprintf("evt_%s", uuid.New()), event, documentID, workspaceID, actor); err != nil {
			return editors.Session{}, err
		}
	}
	if err = transaction.Commit(ctx); err != nil {
		return editors.Session{}, err
	}
	return editors.Session{ProviderID: editors.ProviderCollabora, Mode: "iframe", URL: actionURL.String(), AccessToken: token, ExpiresAt: expires.UTC().Format(time.RFC3339), ReadOnly: !canEdit, AccessTokenTtl: time.Until(expires).Milliseconds()}, nil
}
