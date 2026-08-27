DROP TABLE IF EXISTS encrypted_secrets;
DROP TABLE IF EXISTS integration_connections;
DROP TABLE IF EXISTS integration_installations;
DROP TABLE IF EXISTS report_runs;
DROP TABLE IF EXISTS organization_invitations;
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organization_settings;
DROP TABLE IF EXISTS platform_authorization_grants;
ALTER TABLE organizations DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS status;
