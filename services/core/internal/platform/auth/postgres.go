package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PersistentAuthenticator converts an externally validated identity into the
// stable internal user ID used by authorization and audit records.
type PersistentAuthenticator struct {
	upstream Authenticator
	pool     *pgxpool.Pool
}

func NewPersistentAuthenticator(upstream Authenticator, pool *pgxpool.Pool) *PersistentAuthenticator {
	return &PersistentAuthenticator{upstream: upstream, pool: pool}
}

func (authenticator *PersistentAuthenticator) Authenticate(ctx context.Context, raw string) (Subject, error) {
	subject, err := authenticator.upstream.Authenticate(ctx, raw)
	if err != nil {
		return Subject{}, err
	}
	transaction, err := authenticator.pool.Begin(ctx)
	if err != nil {
		return Subject{}, err
	}
	defer transaction.Rollback(ctx)
	identityKey := subject.Issuer + "\n" + subject.ExternalSubject
	if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", identityKey); err != nil {
		return Subject{}, err
	}
	var userID uuid.UUID
	err = transaction.QueryRow(ctx, "SELECT user_id FROM oidc_identities WHERE issuer=$1 AND subject=$2", subject.Issuer, subject.ExternalSubject).Scan(&userID)
	if err == pgx.ErrNoRows {
		userID = uuid.New()
		if _, err := transaction.Exec(ctx, "INSERT INTO users(id,email,display_name) VALUES($1,$2,$3)", userID, nullable(subject.Email), displayName(subject)); err != nil {
			return Subject{}, err
		}
		if _, err := transaction.Exec(ctx, "INSERT INTO oidc_identities(id,user_id,issuer,subject) VALUES($1,$2,$3,$4)", uuid.New(), userID, subject.Issuer, subject.ExternalSubject); err != nil {
			return Subject{}, err
		}
	} else if err != nil {
		return Subject{}, err
	} else {
		if _, err := transaction.Exec(ctx, "UPDATE users SET email=$2,display_name=$3,updated_at=now() WHERE id=$1", userID, nullable(subject.Email), displayName(subject)); err != nil {
			return Subject{}, err
		}
		if _, err := transaction.Exec(ctx, "UPDATE oidc_identities SET last_seen_at=now() WHERE issuer=$1 AND subject=$2", subject.Issuer, subject.ExternalSubject); err != nil {
			return Subject{}, err
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return Subject{}, err
	}
	subject.ID = userID.String()
	return subject, nil
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func displayName(subject Subject) string {
	if subject.DisplayName != "" {
		return subject.DisplayName
	}
	if subject.Email != "" {
		return subject.Email
	}
	return fmt.Sprintf("User %s", subject.ExternalSubject)
}
