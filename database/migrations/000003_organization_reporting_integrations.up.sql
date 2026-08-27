ALTER TABLE organizations
  ADD COLUMN status text NOT NULL DEFAULT 'active' CHECK (status IN ('pending','active','suspended')),
  ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

CREATE TABLE platform_authorization_grants (
  id uuid PRIMARY KEY,
  issuer text NOT NULL,
  subject text NOT NULL,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (issuer, subject)
);

CREATE TABLE organization_settings (
  organization_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
  share_document_titles_with_platform_admins boolean NOT NULL DEFAULT false,
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE organization_memberships (
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role authorization_role NOT NULL,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
  joined_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id, user_id)
);

INSERT INTO organization_memberships(organization_id,user_id,role)
SELECT DISTINCT ON (organization_id,user_id) organization_id,user_id,role
FROM authorization_grants WHERE user_id IS NOT NULL
ORDER BY organization_id,user_id,CASE role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'manager' THEN 3 WHEN 'editor' THEN 4 WHEN 'contributor' THEN 5 WHEN 'viewer' THEN 6 ELSE 7 END
ON CONFLICT DO NOTHING;

CREATE TABLE organization_invitations (
  id uuid PRIMARY KEY,
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email text NOT NULL,
  role authorization_role NOT NULL,
  token_hash bytea NOT NULL UNIQUE,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','revoked','expired')),
  expires_at timestamptz NOT NULL,
  created_by uuid NOT NULL REFERENCES users(id),
  accepted_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organization_invitations_org_status_idx ON organization_invitations(organization_id,status,created_at DESC);

CREATE TABLE report_runs (
  id uuid PRIMARY KEY,
  organization_id uuid NOT NULL REFERENCES organizations(id),
  workspace_id uuid REFERENCES workspaces(id),
  definition_id text NOT NULL,
  format text NOT NULL CHECK (format IN ('csv','xlsx','pdf')),
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL CHECK (status IN ('queued','running','completed','failed')),
  document_id uuid REFERENCES documents(id),
  error_code text,
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);
CREATE INDEX report_runs_org_created_idx ON report_runs(organization_id,created_at DESC,id DESC);

CREATE TABLE integration_installations (
  id text PRIMARY KEY,
  version text NOT NULL,
  enabled boolean NOT NULL DEFAULT false,
  status text NOT NULL DEFAULT 'setup-required',
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE integration_connections (
  id uuid PRIMARY KEY,
  integration_id text NOT NULL REFERENCES integration_installations(id),
  organization_id uuid REFERENCES organizations(id) ON DELETE CASCADE,
  user_id uuid REFERENCES users(id) ON DELETE CASCADE,
  connection_type text NOT NULL,
  status text NOT NULL,
  safe_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (organization_id IS NOT NULL OR user_id IS NOT NULL)
);

CREATE TABLE encrypted_secrets (
  id uuid PRIMARY KEY,
  connection_id uuid NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
  name text NOT NULL,
  ciphertext bytea NOT NULL,
  nonce bytea NOT NULL,
  key_version integer NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(connection_id,name)
);

INSERT INTO integration_installations(id,version,enabled,status) VALUES
 ('native','1',true,'installed'),
 ('collabora','1',false,'setup-required'),
 ('libreoffice-desktop','1',true,'installed'),
 ('google-workspace','1',false,'setup-required')
ON CONFLICT DO NOTHING;
