CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE documi_installation (
    id smallint PRIMARY KEY CHECK (id = 1),
    installation_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE documi_installation IS 'Minimal connectivity marker; product domain tables arrive with their owning vertical slice.';
