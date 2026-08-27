package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func main() {
	organization := flag.String("organization", "", "organization name")
	workspace := flag.String("workspace", "", "workspace name")
	issuer := flag.String("issuer", "", "OIDC issuer")
	subject := flag.String("subject", "", "OIDC subject")
	email := flag.String("email", "", "email")
	name := flag.String("display-name", "", "display name")
	flag.Parse()
	if *organization == "" || *workspace == "" || *issuer == "" || *subject == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "organization, workspace, issuer, subject, and name are required")
		os.Exit(2)
	}
	if err := bootstrap(context.Background(), os.Getenv("DOCUMI_DATABASE_URL"), *organization, *workspace, *issuer, *subject, *email, *name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("bootstrap: organization owner is ready")
}

func bootstrap(ctx context.Context, databaseURL, organizationName, workspaceName, issuer, subject, email, displayName string) error {
	if databaseURL == "" {
		return fmt.Errorf("DOCUMI_DATABASE_URL is required")
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer connection.Close(ctx)
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var userID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT user_id FROM oidc_identities WHERE issuer=$1 AND subject=$2", issuer, subject).Scan(&userID)
	if err == pgx.ErrNoRows {
		userID = uuid.New()
		identityID := uuid.New()
		if _, err = tx.Exec(ctx, "INSERT INTO users(id,email,display_name) VALUES($1,$2,$3)", userID, email, displayName); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO oidc_identities(id,user_id,issuer,subject) VALUES($1,$2,$3,$4)", identityID, userID, issuer, subject); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var organizationID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT id FROM organizations WHERE name=$1 ORDER BY created_at LIMIT 1", organizationName).Scan(&organizationID)
	if err == pgx.ErrNoRows {
		organizationID = uuid.New()
		_, err = tx.Exec(ctx, "INSERT INTO organizations(id,name) VALUES($1,$2)", organizationID, organizationName)
	}
	if err != nil {
		return err
	}
	var workspaceID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT id FROM workspaces WHERE organization_id=$1 AND name=$2", organizationID, workspaceName).Scan(&workspaceID)
	if err == pgx.ErrNoRows {
		workspaceID = uuid.New()
		_, err = tx.Exec(ctx, "INSERT INTO workspaces(id,organization_id,name) VALUES($1,$2,$3)", workspaceID, organizationID, workspaceName)
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO authorization_grants(id,organization_id,principal_type,user_id,scope_type,role,created_by) SELECT $1,$2,'user',$3,'organization','owner',$3 WHERE NOT EXISTS(SELECT 1 FROM authorization_grants WHERE organization_id=$2 AND user_id=$3 AND scope_type='organization' AND role='owner')", uuid.New(), organizationID, userID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
