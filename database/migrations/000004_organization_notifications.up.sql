ALTER TABLE organization_invitations
  ADD COLUMN notification_error text,
  ADD COLUMN last_sent_at timestamptz;

CREATE INDEX organization_memberships_org_status_idx
  ON organization_memberships(organization_id,status,joined_at DESC,user_id);
