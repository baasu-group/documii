DROP INDEX IF EXISTS documents_external_connection_idx;
ALTER TABLE documents DROP COLUMN IF EXISTS external_connection_id;
