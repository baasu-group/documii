DROP INDEX IF EXISTS organization_memberships_org_status_idx;
ALTER TABLE organization_invitations
  DROP COLUMN IF EXISTS last_sent_at,
  DROP COLUMN IF EXISTS notification_error;
