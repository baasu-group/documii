package integrations

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/documi/documi/services/core/internal/modules/documents"
	"github.com/documi/documi/services/core/internal/modules/secrets"
	platformauth "github.com/documi/documi/services/core/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"net/url"
	"time"
)

type GoogleConnector struct {
	pool         *pgxpool.Pool
	store        *secrets.Store
	cipher       *secrets.Cipher
	oauth        oauth2.Config
	appURL       string
	httpClient   *http.Client
	driveBaseURL string
}
type OAuthStart struct {
	AuthorizationURL string    `json:"authorizationUrl"`
	ExpiresAt        time.Time `json:"expiresAt"`
}
type PickerSession struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Scope       string    `json:"scope"`
}
type GoogleDocumentRequest struct {
	ConnectionID, WorkspaceID, IdempotencyKey string
	FolderID                                  *string
	Family, Title, ExternalFileID             string
}

func NewGoogleConnector(pool *pgxpool.Pool, store *secrets.Store, cipher *secrets.Cipher, clientID, clientSecret, redirectURL, appURL string) *GoogleConnector {
	return &GoogleConnector{pool: pool, store: store, cipher: cipher, appURL: appURL, httpClient: &http.Client{Timeout: 15 * time.Second}, driveBaseURL: "https://www.googleapis.com/drive/v3", oauth: oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}, Scopes: []string{"openid", "email", "https://www.googleapis.com/auth/drive.file"}}}
}
func randomURLToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
func (g *GoogleConnector) Start(ctx context.Context, userID string, organizationID *string) (OAuthStart, error) {
	user, err := uuid.Parse(userID)
	if err != nil {
		return OAuthStart{}, ErrForbidden
	}
	var org *uuid.UUID
	if organizationID != nil {
		value, err := uuid.Parse(*organizationID)
		if err != nil {
			return OAuthStart{}, ErrNotFound
		}
		org = &value
	}
	state, err := randomURLToken(32)
	if err != nil {
		return OAuthStart{}, err
	}
	verifier, err := randomURLToken(48)
	if err != nil {
		return OAuthStart{}, err
	}
	stateHash := sha256.Sum256([]byte(state))
	challengeHash := sha256.Sum256([]byte(verifier))
	label := "google-oauth-state\n" + hex.EncodeToString(stateHash[:])
	ciphertext, nonce, version, err := g.cipher.Encrypt(label, []byte(verifier))
	if err != nil {
		return OAuthStart{}, err
	}
	expires := time.Now().UTC().Add(10 * time.Minute)
	_, err = g.pool.Exec(ctx, `INSERT INTO integration_oauth_states(state_hash,integration_id,organization_id,user_id,verifier_ciphertext,verifier_nonce,key_version,expires_at) VALUES($1,'google-workspace',$2,$3,$4,$5,$6,$7)`, stateHash[:], org, user, ciphertext, nonce, version, expires)
	if err != nil {
		return OAuthStart{}, err
	}
	authorization := g.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"), oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(challengeHash[:])), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	return OAuthStart{AuthorizationURL: authorization, ExpiresAt: expires}, nil
}
func (g *GoogleConnector) Callback(ctx context.Context, state, code string) (Connection, string, error) {
	if state == "" || code == "" {
		return Connection{}, "", ErrInvalid
	}
	hash := sha256.Sum256([]byte(state))
	var orgID *string
	var userID string
	var ciphertext, nonce []byte
	var version int
	var expires time.Time
	err := g.pool.QueryRow(ctx, `DELETE FROM integration_oauth_states WHERE state_hash=$1 RETURNING organization_id,user_id,verifier_ciphertext,verifier_nonce,key_version,expires_at`, hash[:]).Scan(&orgID, &userID, &ciphertext, &nonce, &version, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return Connection{}, "", ErrInvalid
	}
	if err != nil {
		return Connection{}, "", err
	}
	if time.Now().After(expires) {
		return Connection{}, "", ErrInvalid
	}
	label := "google-oauth-state\n" + hex.EncodeToString(hash[:])
	verifier, err := g.cipher.Decrypt(label, ciphertext, nonce, version)
	if err != nil {
		return Connection{}, "", ErrInvalid
	}
	token, err := g.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", string(verifier)))
	if err != nil {
		return Connection{}, "", ErrInvalid
	}
	if token.RefreshToken == "" {
		return Connection{}, "", ErrInvalid
	}
	encoded, err := json.Marshal(token)
	if err != nil {
		return Connection{}, "", err
	}
	connectionID := uuid.New()
	actor := uuid.MustParse(userID)
	safe := map[string]any{"scope": "drive.file", "oauth": "authorization-code-pkce"}
	safeJSON, _ := json.Marshal(safe)
	connectionType := "personal-oauth"
	var org *uuid.UUID
	if orgID != nil {
		parsed := uuid.MustParse(*orgID)
		org = &parsed
		connectionType = "organization-oauth"
	}
	tx, err := g.pool.Begin(ctx)
	if err != nil {
		return Connection{}, "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO integration_connections(id,integration_id,organization_id,user_id,connection_type,status,safe_metadata,created_by) VALUES($1,'google-workspace',$2,$3,$4,'connected',$5,$3)`, connectionID, org, actor, connectionType, safeJSON); err != nil {
		return Connection{}, "", err
	}
	if err = g.store.PutTx(ctx, tx, connectionID.String(), "oauth-token", encoded); err != nil {
		return Connection{}, "", err
	}
	if org != nil {
		if _, err = tx.Exec(ctx, `INSERT INTO audit_events(id,event_type,organization_id,actor_id,resource_id) VALUES($1,'integration.google.connected',$2,$3,$4)`, "evt_"+uuid.NewString(), org, actor, connectionID); err != nil {
			return Connection{}, "", err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Connection{}, "", err
	}
	now := time.Now().UTC()
	connection := Connection{ID: connectionID.String(), IntegrationID: "google-workspace", OrganizationID: orgID, UserID: &userID, ConnectionType: connectionType, Status: "connected", SafeMetadata: safe, CreatedAt: now, UpdatedAt: now}
	redirect := g.appURL + "/dashboard?google=connected"
	if orgID != nil {
		redirect = g.appURL + "/organizations/" + url.PathEscape(*orgID) + "/integrations?google=connected"
	}
	return connection, redirect, nil
}
func (g *GoogleConnector) Picker(ctx context.Context, subject platformauth.Subject, connectionID string) (PickerSession, error) {
	connection, err := uuid.Parse(connectionID)
	if err != nil {
		return PickerSession{}, ErrNotFound
	}
	var owner uuid.UUID
	var org *uuid.UUID
	if err = g.pool.QueryRow(ctx, `SELECT user_id,organization_id FROM integration_connections WHERE id=$1 AND integration_id='google-workspace' AND status='connected'`, connection).Scan(&owner, &org); errors.Is(err, pgx.ErrNoRows) {
		return PickerSession{}, ErrNotFound
	} else if err != nil {
		return PickerSession{}, err
	}
	actor, err := uuid.Parse(subject.ID)
	if err != nil {
		return PickerSession{}, ErrForbidden
	}
	allowed := owner == actor
	if !allowed && org != nil {
		if err = g.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_memberships WHERE organization_id=$1 AND user_id=$2 AND status='active' AND role IN('owner','admin'))`, org, actor).Scan(&allowed); err != nil {
			return PickerSession{}, err
		}
	}
	if !allowed {
		return PickerSession{}, ErrForbidden
	}
	encoded, err := g.store.Get(ctx, connectionID, "oauth-token")
	if err != nil {
		return PickerSession{}, ErrForbidden
	}
	var token oauth2.Token
	if json.Unmarshal(encoded, &token) != nil {
		return PickerSession{}, ErrInvalid
	}
	fresh, err := g.oauth.TokenSource(ctx, &token).Token()
	if err != nil {
		return PickerSession{}, ErrInvalid
	}
	if fresh.RefreshToken == "" {
		fresh.RefreshToken = token.RefreshToken
	}
	updated, _ := json.Marshal(fresh)
	_ = g.store.Put(ctx, connectionID, "oauth-token", updated)
	return PickerSession{AccessToken: fresh.AccessToken, ExpiresAt: fresh.Expiry, Scope: "https://www.googleapis.com/auth/drive.file"}, nil
}

type driveFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MIMEType    string `json:"mimeType"`
	WebViewLink string `json:"webViewLink"`
}

func googleFormat(family string) (provider, mime string, err error) {
	switch family {
	case "text":
		return "google-docs", "application/vnd.google-apps.document", nil
	case "spreadsheet":
		return "google-sheets", "application/vnd.google-apps.spreadsheet", nil
	case "presentation":
		return "google-slides", "application/vnd.google-apps.presentation", nil
	default:
		return "", "", ErrInvalid
	}
}

func (g *GoogleConnector) driveRequest(ctx context.Context, accessToken, method, endpoint string, body any) (driveFile, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return driveFile{}, err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(g.driveBaseURL, "/")+endpoint, reader)
	if err != nil {
		return driveFile{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := g.httpClient.Do(request)
	if err != nil {
		return driveFile{}, ErrInvalid
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 1<<20)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, limited)
		return driveFile{}, ErrInvalid
	}
	var value driveFile
	if json.NewDecoder(limited).Decode(&value) != nil || value.ID == "" {
		return driveFile{}, ErrInvalid
	}
	return value, nil
}

func (g *GoogleConnector) CreateFile(ctx context.Context, subject platformauth.Subject, request GoogleDocumentRequest) (driveFile, string, string, error) {
	provider, mime, err := googleFormat(request.Family)
	if err != nil || strings.TrimSpace(request.Title) == "" || len(request.Title) > 255 {
		return driveFile{}, "", "", ErrInvalid
	}
	session, err := g.Picker(ctx, subject, request.ConnectionID)
	if err != nil {
		return driveFile{}, "", "", err
	}
	fields := url.QueryEscape("id,name,mimeType,webViewLink")
	value, err := g.driveRequest(ctx, session.AccessToken, http.MethodPost, "/files?fields="+fields, map[string]string{"name": strings.TrimSpace(request.Title), "mimeType": mime})
	if err != nil {
		return driveFile{}, "", "", err
	}
	if value.WebViewLink == "" {
		value.WebViewLink = "https://docs.google.com/open?id=" + url.QueryEscape(value.ID)
	}
	return value, provider, mime, nil
}

func (g *GoogleConnector) InspectFile(ctx context.Context, subject platformauth.Subject, connectionID, fileID string) (driveFile, string, string, error) {
	if fileID == "" || strings.ContainsAny(fileID, "/\\") {
		return driveFile{}, "", "", ErrInvalid
	}
	session, err := g.Picker(ctx, subject, connectionID)
	if err != nil {
		return driveFile{}, "", "", err
	}
	fields := url.QueryEscape("id,name,mimeType,webViewLink")
	value, err := g.driveRequest(ctx, session.AccessToken, http.MethodGet, path.Join("/files", url.PathEscape(fileID))+"?fields="+fields, nil)
	if err != nil {
		return driveFile{}, "", "", err
	}
	var family string
	switch value.MIMEType {
	case "application/vnd.google-apps.document":
		family = "text"
	case "application/vnd.google-apps.spreadsheet":
		family = "spreadsheet"
	case "application/vnd.google-apps.presentation":
		family = "presentation"
	default:
		return driveFile{}, "", "", ErrInvalid
	}
	provider, _, _ := googleFormat(family)
	if value.WebViewLink == "" {
		value.WebViewLink = "https://docs.google.com/open?id=" + url.QueryEscape(value.ID)
	}
	return value, provider, family, nil
}

func (g *GoogleConnector) CreateReference(ctx context.Context, documentService *documents.Service, subject platformauth.Subject, request GoogleDocumentRequest, create bool) (documents.Document, error) {
	var file driveFile
	var provider, family, mime string
	var err error
	if create {
		file, provider, mime, err = g.CreateFile(ctx, subject, request)
		family = request.Family
	} else {
		file, provider, family, err = g.InspectFile(ctx, subject, request.ConnectionID, request.ExternalFileID)
		mime = file.MIMEType
	}
	if err != nil {
		return documents.Document{}, err
	}
	return documentService.CreateExternal(ctx, documents.CreateExternalRequest{ActorID: subject.ID, WorkspaceID: request.WorkspaceID, FolderID: request.FolderID, Title: file.Name, Family: family, MIMEType: mime, Provider: provider, ExternalFileID: file.ID, ExternalURL: file.WebViewLink, ExternalConnectionID: request.ConnectionID, IdempotencyKey: request.IdempotencyKey})
}
