CREATE TABLE users (
  id uuid PRIMARY KEY, email text, display_name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE oidc_identities (
  id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), issuer text NOT NULL, subject text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), last_seen_at timestamptz NOT NULL DEFAULT now(), UNIQUE(issuer, subject)
);
CREATE TABLE organizations (id uuid PRIMARY KEY, name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE workspaces (id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES organizations(id), name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(organization_id,name));
CREATE TABLE folders (
  id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES organizations(id), workspace_id uuid NOT NULL REFERENCES workspaces(id),
  parent_id uuid REFERENCES folders(id), name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE NULLS NOT DISTINCT (workspace_id,parent_id,name)
);
CREATE FUNCTION validate_folder_hierarchy() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_workspace uuid; parent_organization uuid; cycle_found boolean;
BEGIN
  IF NEW.parent_id IS NULL THEN RETURN NEW; END IF;
  SELECT workspace_id, organization_id INTO parent_workspace, parent_organization FROM folders WHERE id=NEW.parent_id;
  IF parent_workspace IS NULL OR parent_workspace<>NEW.workspace_id OR parent_organization<>NEW.organization_id THEN
    RAISE EXCEPTION 'folder parent must belong to the same organization and workspace';
  END IF;
  WITH RECURSIVE ancestors(id,parent_id) AS (
    SELECT id,parent_id FROM folders WHERE id=NEW.parent_id
    UNION ALL SELECT f.id,f.parent_id FROM folders f JOIN ancestors a ON f.id=a.parent_id
  ) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id=NEW.id) INTO cycle_found;
  IF cycle_found THEN RAISE EXCEPTION 'folder cycle is not allowed'; END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER folders_validate_hierarchy BEFORE INSERT OR UPDATE OF parent_id,workspace_id,organization_id ON folders FOR EACH ROW EXECUTE FUNCTION validate_folder_hierarchy();
CREATE TABLE groups (id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES organizations(id), name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(organization_id,name));
CREATE TABLE group_members (group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(group_id,user_id));
CREATE TYPE authorization_role AS ENUM ('owner','admin','manager','editor','contributor','viewer','guest');
CREATE TYPE authorization_scope AS ENUM ('organization','workspace','folder','document');
CREATE TYPE principal_type AS ENUM ('user','group');
CREATE TABLE authorization_grants (
  id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES organizations(id), principal_type principal_type NOT NULL,
  user_id uuid REFERENCES users(id), group_id uuid REFERENCES groups(id), scope_type authorization_scope NOT NULL,
  workspace_id uuid REFERENCES workspaces(id), folder_id uuid REFERENCES folders(id), document_id uuid,
  role authorization_role NOT NULL, created_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((principal_type='user' AND user_id IS NOT NULL AND group_id IS NULL) OR (principal_type='group' AND group_id IS NOT NULL AND user_id IS NULL)),
  CHECK ((scope_type='organization' AND workspace_id IS NULL AND folder_id IS NULL AND document_id IS NULL) OR
         (scope_type='workspace' AND workspace_id IS NOT NULL AND folder_id IS NULL AND document_id IS NULL) OR
         (scope_type='folder' AND workspace_id IS NOT NULL AND folder_id IS NOT NULL AND document_id IS NULL) OR
         (scope_type='document' AND workspace_id IS NOT NULL AND document_id IS NOT NULL))
);
CREATE FUNCTION validate_authorization_grant() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.group_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM groups WHERE id=NEW.group_id AND organization_id=NEW.organization_id) THEN
    RAISE EXCEPTION 'grant group must belong to the same organization';
  END IF;
  IF NEW.workspace_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM workspaces WHERE id=NEW.workspace_id AND organization_id=NEW.organization_id) THEN
    RAISE EXCEPTION 'grant workspace must belong to the same organization';
  END IF;
  IF NEW.folder_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM folders WHERE id=NEW.folder_id AND workspace_id=NEW.workspace_id AND organization_id=NEW.organization_id) THEN
    RAISE EXCEPTION 'grant folder must belong to the same workspace and organization';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER authorization_grants_validate BEFORE INSERT OR UPDATE ON authorization_grants FOR EACH ROW EXECUTE FUNCTION validate_authorization_grant();

CREATE FUNCTION protect_last_organization_owner() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE removing_owner boolean := false;
BEGIN
  IF OLD.scope_type='organization' AND OLD.role='owner' THEN
    PERFORM 1 FROM organizations WHERE id=OLD.organization_id FOR UPDATE;
    IF TG_OP='DELETE' THEN removing_owner := true;
    ELSE removing_owner := NEW.scope_type<>'organization' OR NEW.role<>'owner' OR NEW.organization_id<>OLD.organization_id;
    END IF;
  END IF;
  IF removing_owner AND NOT EXISTS (SELECT 1 FROM authorization_grants WHERE organization_id=OLD.organization_id AND scope_type='organization' AND role='owner' AND id<>OLD.id) THEN
    RAISE EXCEPTION 'cannot remove the last organization owner';
  END IF;
  IF TG_OP='DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER authorization_grants_protect_owner BEFORE DELETE OR UPDATE ON authorization_grants FOR EACH ROW EXECUTE FUNCTION protect_last_organization_owner();
CREATE TABLE storage_objects (
  id uuid PRIMARY KEY, provider text NOT NULL, storage_key text NOT NULL UNIQUE, sha256 text NOT NULL CHECK(length(sha256)=64),
  size bigint NOT NULL CHECK(size>=0), content_type text NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE documents (
  id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES organizations(id), workspace_id uuid NOT NULL REFERENCES workspaces(id), folder_id uuid REFERENCES folders(id),
  title text NOT NULL, family text NOT NULL CHECK(family IN ('native','text','spreadsheet','presentation','pdf')), mime_type text NOT NULL,
  editor_provider text NOT NULL, storage_mode text NOT NULL CHECK(storage_mode IN ('managed','external')), current_version_id uuid,
  external_provider text, external_file_id text, external_url text, native_format text,
  created_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((storage_mode='managed' AND external_file_id IS NULL) OR (storage_mode='external' AND external_provider IS NOT NULL AND external_file_id IS NOT NULL AND external_url IS NOT NULL)),
  CHECK (native_format IS NULL OR native_format='documi-json-v1')
);
CREATE FUNCTION validate_document_folder() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.folder_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM folders WHERE id=NEW.folder_id AND workspace_id=NEW.workspace_id AND organization_id=NEW.organization_id
  ) THEN RAISE EXCEPTION 'document folder must belong to the same organization and workspace'; END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER documents_validate_folder BEFORE INSERT OR UPDATE OF folder_id,workspace_id,organization_id ON documents FOR EACH ROW EXECUTE FUNCTION validate_document_folder();
ALTER TABLE authorization_grants ADD CONSTRAINT authorization_grants_document_fk FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE;
CREATE FUNCTION validate_document_grant() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.document_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM documents WHERE id=NEW.document_id AND workspace_id=NEW.workspace_id AND organization_id=NEW.organization_id) THEN
    RAISE EXCEPTION 'grant document must belong to the same workspace and organization';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER authorization_grants_validate_document BEFORE INSERT OR UPDATE OF document_id,workspace_id,organization_id ON authorization_grants FOR EACH ROW EXECUTE FUNCTION validate_document_grant();
CREATE TABLE document_versions (
  id uuid PRIMARY KEY, document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE, parent_version_id uuid REFERENCES document_versions(id),
  storage_object_id uuid NOT NULL REFERENCES storage_objects(id), version_number bigint NOT NULL CHECK(version_number>0), created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(document_id,version_number)
);
CREATE FUNCTION validate_document_version_parent() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.parent_version_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM document_versions WHERE id=NEW.parent_version_id AND document_id=NEW.document_id) THEN
    RAISE EXCEPTION 'parent version must belong to the same document';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER document_versions_validate_parent BEFORE INSERT OR UPDATE ON document_versions FOR EACH ROW EXECUTE FUNCTION validate_document_version_parent();
ALTER TABLE documents ADD CONSTRAINT documents_current_version_fk FOREIGN KEY(current_version_id) REFERENCES document_versions(id);
CREATE FUNCTION validate_document_current_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.current_version_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM document_versions WHERE id=NEW.current_version_id AND document_id=NEW.id) THEN
    RAISE EXCEPTION 'current version must belong to the document';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER documents_validate_current_version BEFORE UPDATE OF current_version_id ON documents FOR EACH ROW EXECUTE FUNCTION validate_document_current_version();
CREATE TABLE document_editor_policies (organization_id uuid NOT NULL REFERENCES organizations(id), family text NOT NULL, provider_id text NOT NULL, updated_by uuid NOT NULL REFERENCES users(id), updated_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(organization_id,family));
CREATE TABLE audit_events (
  id text PRIMARY KEY CHECK(id LIKE 'evt_%'), event_type text NOT NULL, organization_id uuid NOT NULL REFERENCES organizations(id), workspace_id uuid REFERENCES workspaces(id), actor_id uuid REFERENCES users(id), resource_id uuid, request_id text, data jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION reject_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit events are immutable'; END $$;
CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation();
CREATE TRIGGER audit_events_no_delete BEFORE DELETE ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation();
CREATE TABLE idempotency_keys (organization_id uuid NOT NULL REFERENCES organizations(id), actor_id uuid NOT NULL REFERENCES users(id), key text NOT NULL, operation text NOT NULL, resource_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(organization_id,actor_id,key));
CREATE TABLE editor_sessions (id uuid PRIMARY KEY, document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE, actor_id uuid NOT NULL REFERENCES users(id), provider_id text NOT NULL, can_edit boolean NOT NULL, expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE wopi_locks (document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE, lock_value text NOT NULL CHECK(length(lock_value)<=1024), base_version_id uuid NOT NULL REFERENCES document_versions(id), expires_at timestamptz NOT NULL, updated_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX documents_workspace_updated_idx ON documents(workspace_id,updated_at DESC);
CREATE INDEX documents_folder_idx ON documents(workspace_id,folder_id);
CREATE INDEX versions_document_idx ON document_versions(document_id,version_number DESC);
CREATE INDEX grants_user_idx ON authorization_grants(organization_id,user_id,scope_type);
CREATE INDEX grants_group_idx ON authorization_grants(organization_id,group_id,scope_type);
CREATE INDEX folders_parent_idx ON folders(workspace_id,parent_id);
CREATE UNIQUE INDEX grants_unique_user ON authorization_grants(organization_id,user_id,scope_type,COALESCE(workspace_id,'00000000-0000-0000-0000-000000000000'),COALESCE(folder_id,'00000000-0000-0000-0000-000000000000'),COALESCE(document_id,'00000000-0000-0000-0000-000000000000'),role) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX grants_unique_group ON authorization_grants(organization_id,group_id,scope_type,COALESCE(workspace_id,'00000000-0000-0000-0000-000000000000'),COALESCE(folder_id,'00000000-0000-0000-0000-000000000000'),COALESCE(document_id,'00000000-0000-0000-0000-000000000000'),role) WHERE group_id IS NOT NULL;
