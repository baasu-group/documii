package administration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/documi/documi/services/core/internal/modules/permissions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuditCursorRoundTripAndValidation(t *testing.T) {
	want := auditCursor{Timestamp: time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC), ID: "evt_test"}
	got, err := decodeCursor(encodeCursor(want))
	if err != nil || got != want {
		t.Fatalf("decodeCursor() = %#v, %v", got, err)
	}
	if _, err := decodeCursor("not-base64"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("decodeCursor() error = %v", err)
	}
}

func TestPostgresAdministrationFlow(t *testing.T) {
	databaseURL := os.Getenv("DOCUMI_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DOCUMI_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	owner, member := uuid.New(), uuid.New()
	organization, workspace := uuid.New(), uuid.New()
	if _, err = pool.Exec(ctx, "INSERT INTO users(id,email,display_name) VALUES($1,$2,'Owner'),($3,$4,'Member')", owner, owner.String()+"@example.test", member, member.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO organizations(id,name) VALUES($1,$2)", organization, "Test "+organization.String()); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO workspaces(id,organization_id,name) VALUES($1,$2,'Main')", workspace, organization); err != nil {
		t.Fatal(err)
	}
	ownerGrant := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO authorization_grants(id,organization_id,principal_type,user_id,scope_type,role,created_by)
		VALUES($1,$2,'user',$3,'organization','owner',$3)`, ownerGrant, organization, owner); err != nil {
		t.Fatal(err)
	}

	service := NewService(pool, permissions.NewAuthorizer(pool), func(context.Context, string, string) error { return nil })
	folder, err := service.CreateFolder(ctx, owner.String(), workspace.String(), "Finance", nil)
	if err != nil {
		t.Fatal(err)
	}
	group, err := service.CreateGroup(ctx, owner.String(), organization.String(), "Reviewers")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.AddMember(ctx, owner.String(), group.ID, member.String()); err != nil {
		t.Fatal(err)
	}
	grant, err := service.CreateGrant(ctx, owner.String(), organization.String(), CreateGrantRequest{
		PrincipalType: "group", PrincipalID: group.ID, ScopeType: "folder", ScopeID: folder.ID, Role: "viewer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if grant.ScopeID != folder.ID {
		t.Fatalf("grant scope = %q", grant.ScopeID)
	}
	capabilities, err := service.EffectiveAccess(ctx, owner.String(), member.String(), "folder", folder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(capabilities) != 2 || capabilities[0] != permissions.CanRead || capabilities[1] != permissions.CanDownload {
		t.Fatalf("capabilities = %v", capabilities)
	}
	policy, err := service.SetEditorPolicy(ctx, owner.String(), organization.String(), "text", "collabora")
	if err != nil || policy.ProviderID != "collabora" {
		t.Fatalf("SetEditorPolicy() = %#v, %v", policy, err)
	}
	page, err := service.Audit(ctx, owner.String(), organization.String(), "", "", "", 2)
	if err != nil || len(page.Events) != 2 || page.NextCursor == "" {
		t.Fatalf("Audit() = %#v, %v", page, err)
	}
	if err := service.DeleteGrant(ctx, owner.String(), ownerGrant.String()); !errors.Is(err, ErrConflict) {
		t.Fatalf("last owner deletion error = %v", err)
	}
}
