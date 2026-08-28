CREATE TABLE integration_oauth_states (
  state_hash bytea PRIMARY KEY,
  integration_id text NOT NULL REFERENCES integration_installations(id),
  organization_id uuid REFERENCES organizations(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  verifier_ciphertext bytea NOT NULL,
  verifier_nonce bytea NOT NULL,
  key_version integer NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX integration_oauth_states_expiry_idx ON integration_oauth_states(expires_at);
