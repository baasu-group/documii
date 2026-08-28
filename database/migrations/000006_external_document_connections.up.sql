ALTER TABLE documents
  ADD COLUMN external_connection_id uuid REFERENCES integration_connections(id) ON DELETE SET NULL;

CREATE INDEX documents_external_connection_idx
  ON documents(external_connection_id)
  WHERE external_connection_id IS NOT NULL;
