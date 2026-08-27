package editors

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPolicyStore struct{ pool *pgxpool.Pool }

func NewPostgresPolicyStore(pool *pgxpool.Pool) *PostgresPolicyStore {
	return &PostgresPolicyStore{pool: pool}
}

func (store *PostgresPolicyStore) Preferred(ctx context.Context, organizationID string, family DocumentFamily) (ProviderID, bool, error) {
	id, err := uuid.Parse(organizationID)
	if err != nil {
		return "", false, ErrProviderUnavailable
	}
	var provider ProviderID
	err = store.pool.QueryRow(ctx, `SELECT provider_id FROM document_editor_policies
		WHERE organization_id=$1 AND family=$2`, id, family).Scan(&provider)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return provider, err == nil, err
}
