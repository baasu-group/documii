use rusqlite::{params, Connection, OptionalExtension};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Path, PathBuf};
use tauri::{AppHandle, Manager};
use uuid::Uuid;

const NATIVE_MIME: &str = "application/vnd.documi.document+json";
const MAX_NATIVE_BYTES: usize = 5 * 1024 * 1024;

const MIGRATION: &str = r#"
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS local_workspaces (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS local_documents (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL REFERENCES local_workspaces(id), title TEXT NOT NULL,
  family TEXT NOT NULL CHECK(family = 'native'), mime_type TEXT NOT NULL,
  editor_provider TEXT NOT NULL CHECK(editor_provider = 'native'), storage_mode TEXT NOT NULL CHECK(storage_mode = 'managed'),
  current_version_id TEXT, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS local_storage_objects (
  id TEXT PRIMARY KEY, storage_key TEXT NOT NULL UNIQUE, sha256 TEXT NOT NULL, size INTEGER NOT NULL,
  content_type TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS local_document_versions (
  id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES local_documents(id), parent_version_id TEXT,
  storage_object_id TEXT NOT NULL REFERENCES local_storage_objects(id), version_number INTEGER NOT NULL,
  actor_id TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(document_id, version_number)
);
CREATE TABLE IF NOT EXISTS local_idempotency (
  key TEXT PRIMARY KEY, operation TEXT NOT NULL, resource_id TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS local_audit_events (
  id TEXT PRIMARY KEY, event_type TEXT NOT NULL, workspace_id TEXT NOT NULL, actor_id TEXT NOT NULL,
  resource_id TEXT NOT NULL, data TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TRIGGER IF NOT EXISTS local_audit_no_update BEFORE UPDATE ON local_audit_events BEGIN SELECT RAISE(ABORT, 'audit events are immutable'); END;
CREATE TRIGGER IF NOT EXISTS local_audit_no_delete BEFORE DELETE ON local_audit_events BEGIN SELECT RAISE(ABORT, 'audit events are immutable'); END;
"#;

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalWorkspace { pub id: String, pub name: String, pub mode: &'static str }

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalVersion { pub id: String, pub document_id: String, pub parent_version_id: Option<String>, pub version_number: i64, pub sha256: String, pub size: i64, pub created_at: String }

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalDocument { pub id: String, pub workspace_id: String, pub title: String, pub family: String, pub mime_type: String, pub editor_provider: String, pub storage_mode: String, pub current_version_id: String, pub created_at: String, pub updated_at: String, pub content: Option<serde_json::Value> }

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateRequest { pub workspace_id: String, pub title: String, pub content: serde_json::Value, pub idempotency_key: String }

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SaveRequest { pub document_id: String, pub base_version_id: String, pub content: serde_json::Value, pub idempotency_key: String }

struct Paths { root: PathBuf, database: PathBuf }

fn paths(app: &AppHandle) -> Result<Paths, String> {
    let root = app.path().app_data_dir().map_err(|e| e.to_string())?;
    fs::create_dir_all(&root).map_err(|e| e.to_string())?;
    Ok(Paths { database: root.join("documi.sqlite3"), root })
}

fn connect(app: &AppHandle) -> Result<(Connection, Paths), String> {
    let paths = paths(app)?;
    let connection = Connection::open(&paths.database).map_err(|e| e.to_string())?;
    connection.execute_batch(MIGRATION).map_err(|e| e.to_string())?;
    Ok((connection, paths))
}

fn validate_content(content: &serde_json::Value) -> Result<Vec<u8>, String> {
    if content.get("format").and_then(|v| v.as_str()) != Some("documi-json-v1") { return Err("invalid_native_document".into()); }
    let encoded = serde_json::to_vec(content).map_err(|e| e.to_string())?;
    if encoded.len() > MAX_NATIVE_BYTES { return Err("native_document_too_large".into()); }
    Ok(encoded)
}

fn write_object(root: &Path, workspace: &str, document: &str, version: &str, content: &[u8]) -> Result<(String, String), String> {
    for id in [workspace, document, version] { Uuid::parse_str(id).map_err(|_| "invalid_identifier".to_string())?; }
    let relative = PathBuf::from("documents").join(workspace).join(document).join(version).join("document.documi.json");
    let target = root.join(&relative);
    if target.exists() { return Err("storage_object_exists".into()); }
    let parent = target.parent().ok_or("invalid_storage_path")?;
    fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    let temporary = parent.join(format!(".{}.tmp", Uuid::new_v4()));
    fs::write(&temporary, content).map_err(|e| e.to_string())?;
    fs::rename(&temporary, &target).map_err(|e| { let _ = fs::remove_file(&temporary); e.to_string() })?;
    Ok((relative.to_string_lossy().into_owned(), format!("{:x}", Sha256::digest(content))))
}

fn read_document(connection: &Connection, root: &Path, document_id: &str, include_content: bool) -> Result<LocalDocument, String> {
    let mut document = connection.query_row(
        "SELECT id, workspace_id, title, family, mime_type, editor_provider, storage_mode, current_version_id, created_at, updated_at FROM local_documents WHERE id=?1",
        [document_id], |row| Ok(LocalDocument { id: row.get(0)?, workspace_id: row.get(1)?, title: row.get(2)?, family: row.get(3)?, mime_type: row.get(4)?, editor_provider: row.get(5)?, storage_mode: row.get(6)?, current_version_id: row.get(7)?, created_at: row.get(8)?, updated_at: row.get(9)?, content: None })
    ).map_err(|e| e.to_string())?;
    if include_content {
        let key: String = connection.query_row("SELECT storage_key FROM local_storage_objects o JOIN local_document_versions v ON v.storage_object_id=o.id WHERE v.id=?1", [&document.current_version_id], |row| row.get(0)).map_err(|e| e.to_string())?;
        let bytes = fs::read(root.join(key)).map_err(|_| "document_content_missing".to_string())?;
        document.content = Some(serde_json::from_slice(&bytes).map_err(|_| "document_content_corrupt".to_string())?);
    }
    Ok(document)
}

#[tauri::command]
pub fn local_workspace_initialize(app: AppHandle) -> Result<LocalWorkspace, String> {
    let (connection, _) = connect(&app)?;
    let existing: Option<(String, String)> = connection.query_row("SELECT id,name FROM local_workspaces ORDER BY created_at LIMIT 1", [], |r| Ok((r.get(0)?,r.get(1)?))).optional().map_err(|e| e.to_string())?;
    let (id, name) = existing.unwrap_or_else(|| (Uuid::new_v4().to_string(), "Local workspace".into()));
    connection.execute("INSERT OR IGNORE INTO local_workspaces(id,name) VALUES(?1,?2)", params![id,name]).map_err(|e| e.to_string())?;
    Ok(LocalWorkspace { id, name, mode: "local" })
}

#[tauri::command]
pub fn local_documents_list(app: AppHandle, workspace_id: String) -> Result<Vec<LocalDocument>, String> {
    Uuid::parse_str(&workspace_id).map_err(|_| "invalid_identifier".to_string())?;
    let (connection, _) = connect(&app)?;
    let mut statement = connection.prepare("SELECT id, workspace_id, title, family, mime_type, editor_provider, storage_mode, current_version_id, created_at, updated_at FROM local_documents WHERE workspace_id=?1 ORDER BY updated_at DESC").map_err(|e| e.to_string())?;
    let rows = statement.query_map([workspace_id], |row| Ok(LocalDocument { id: row.get(0)?, workspace_id: row.get(1)?, title: row.get(2)?, family: row.get(3)?, mime_type: row.get(4)?, editor_provider: row.get(5)?, storage_mode: row.get(6)?, current_version_id: row.get(7)?, created_at: row.get(8)?, updated_at: row.get(9)?, content: None })).map_err(|e| e.to_string())?;
    rows.collect::<Result<Vec<_>, _>>().map_err(|e| e.to_string())
}

#[tauri::command]
pub fn local_document_create(app: AppHandle, request: CreateRequest) -> Result<LocalDocument, String> {
    let encoded = validate_content(&request.content)?;
    let (mut connection, paths) = connect(&app)?;
    if request.idempotency_key.trim().is_empty() { return Err("idempotency_key_required".into()); }
    if let Some(resource) = connection.query_row("SELECT resource_id FROM local_idempotency WHERE key=?1 AND operation='create'", [&request.idempotency_key], |r| r.get::<_,String>(0)).optional().map_err(|e| e.to_string())? { return read_document(&connection, &paths.root, &resource, true); }
    Uuid::parse_str(&request.workspace_id).map_err(|_| "invalid_identifier".to_string())?;
    if request.title.trim().is_empty() { return Err("title_required".into()); }
    let document_id = Uuid::new_v4().to_string(); let version_id = Uuid::new_v4().to_string(); let object_id = Uuid::new_v4().to_string();
    let (key, hash) = write_object(&paths.root, &request.workspace_id, &document_id, &version_id, &encoded)?;
    let tx = connection.transaction().map_err(|e| e.to_string())?;
    let result = (|| -> Result<(), rusqlite::Error> {
        tx.execute("INSERT INTO local_documents(id,workspace_id,title,family,mime_type,editor_provider,storage_mode,current_version_id) VALUES(?1,?2,?3,'native',?4,'native','managed',?5)", params![document_id,request.workspace_id,request.title.trim(),NATIVE_MIME,version_id])?;
        tx.execute("INSERT INTO local_storage_objects(id,storage_key,sha256,size,content_type) VALUES(?1,?2,?3,?4,?5)", params![object_id,key,hash,encoded.len() as i64,NATIVE_MIME])?;
        tx.execute("INSERT INTO local_document_versions(id,document_id,storage_object_id,version_number,actor_id) VALUES(?1,?2,?3,1,'local-owner')", params![version_id,document_id,object_id])?;
        tx.execute("INSERT INTO local_idempotency(key,operation,resource_id) VALUES(?1,'create',?2)", params![request.idempotency_key,document_id])?;
        tx.execute("INSERT INTO local_audit_events(id,event_type,workspace_id,actor_id,resource_id) VALUES(?1,'document.created',?2,'local-owner',?3)", params![format!("evt_{}",Uuid::new_v4()),request.workspace_id,document_id])?; Ok(()) })();
    if let Err(error) = result { let _=fs::remove_file(paths.root.join(&key)); return Err(error.to_string()); }
    if let Err(error) = tx.commit() { let _=fs::remove_file(paths.root.join(&key)); return Err(error.to_string()); }
    read_document(&connection, &paths.root, &document_id, true)
}

#[tauri::command]
pub fn local_document_get(app: AppHandle, document_id: String) -> Result<LocalDocument, String> { let (connection, paths)=connect(&app)?; let doc=read_document(&connection,&paths.root,&document_id,true)?; connection.execute("INSERT INTO local_audit_events(id,event_type,workspace_id,actor_id,resource_id) VALUES(?1,'document.editor.opened',?2,'local-owner',?3)",params![format!("evt_{}",Uuid::new_v4()),doc.workspace_id,doc.id]).map_err(|e|e.to_string())?; Ok(doc) }

#[tauri::command]
pub fn local_document_save(app: AppHandle, request: SaveRequest) -> Result<LocalDocument, String> {
    let encoded=validate_content(&request.content)?; let (mut connection,paths)=connect(&app)?;
    if request.idempotency_key.trim().is_empty() { return Err("idempotency_key_required".into()); }
    if let Some(resource)=connection.query_row("SELECT resource_id FROM local_idempotency WHERE key=?1 AND operation='save'",[&request.idempotency_key],|r|r.get::<_,String>(0)).optional().map_err(|e|e.to_string())? { return read_document(&connection,&paths.root,&resource,true); }
    let current=read_document(&connection,&paths.root,&request.document_id,false)?; if current.current_version_id != request.base_version_id { return Err("document_version_conflict".into()); }
    let version_id=Uuid::new_v4().to_string(); let object_id=Uuid::new_v4().to_string(); let (key,hash)=write_object(&paths.root,&current.workspace_id,&current.id,&version_id,&encoded)?;
    let database_result = (|| -> Result<(), String> {
        let tx=connection.transaction().map_err(|e|e.to_string())?;
        let number:i64=tx.query_row("SELECT COALESCE(MAX(version_number),0)+1 FROM local_document_versions WHERE document_id=?1",[&current.id],|r|r.get(0)).map_err(|e|e.to_string())?;
        let changed=tx.execute("UPDATE local_documents SET current_version_id=?1,updated_at=CURRENT_TIMESTAMP WHERE id=?2 AND current_version_id=?3",params![version_id,current.id,request.base_version_id]).map_err(|e|e.to_string())?;
        if changed!=1 { return Err("document_version_conflict".into()); }
        tx.execute("INSERT INTO local_storage_objects(id,storage_key,sha256,size,content_type) VALUES(?1,?2,?3,?4,?5)",params![object_id,key,hash,encoded.len() as i64,NATIVE_MIME]).map_err(|e|e.to_string())?;
        tx.execute("INSERT INTO local_document_versions(id,document_id,parent_version_id,storage_object_id,version_number,actor_id) VALUES(?1,?2,?3,?4,?5,'local-owner')",params![version_id,current.id,request.base_version_id,object_id,number]).map_err(|e|e.to_string())?;
        tx.execute("INSERT INTO local_idempotency(key,operation,resource_id) VALUES(?1,'save',?2)",params![request.idempotency_key,current.id]).map_err(|e|e.to_string())?;
        tx.execute("INSERT INTO local_audit_events(id,event_type,workspace_id,actor_id,resource_id) VALUES(?1,'document.editor.saved',?2,'local-owner',?3)",params![format!("evt_{}",Uuid::new_v4()),current.workspace_id,current.id]).map_err(|e|e.to_string())?;
        tx.commit().map_err(|e|e.to_string())
    })();
    if let Err(error)=database_result { let _=fs::remove_file(paths.root.join(&key)); return Err(error); }
    read_document(&connection,&paths.root,&request.document_id,true)
}

#[tauri::command]
pub fn local_document_versions(app: AppHandle, document_id: String) -> Result<Vec<LocalVersion>, String> { let (connection,_)=connect(&app)?; let mut s=connection.prepare("SELECT v.id,v.document_id,v.parent_version_id,v.version_number,o.sha256,o.size,v.created_at FROM local_document_versions v JOIN local_storage_objects o ON o.id=v.storage_object_id WHERE v.document_id=?1 ORDER BY v.version_number DESC").map_err(|e|e.to_string())?; let rows=s.query_map([document_id],|r|Ok(LocalVersion{id:r.get(0)?,document_id:r.get(1)?,parent_version_id:r.get(2)?,version_number:r.get(3)?,sha256:r.get(4)?,size:r.get(5)?,created_at:r.get(6)?})).map_err(|e|e.to_string())?; rows.collect::<Result<Vec<_>,_>>().map_err(|e|e.to_string()) }

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn migration_makes_audit_events_append_only() {
        let connection = Connection::open_in_memory().unwrap();
        connection.execute_batch(MIGRATION).unwrap();
        connection.execute("INSERT INTO local_audit_events(id,event_type,workspace_id,actor_id,resource_id) VALUES('evt_1','document.created','workspace','owner','document')", []).unwrap();
        assert!(connection.execute("UPDATE local_audit_events SET event_type='changed' WHERE id='evt_1'", []).is_err());
        assert!(connection.execute("DELETE FROM local_audit_events WHERE id='evt_1'", []).is_err());
    }

    #[test]
    fn object_paths_are_generated_from_uuid_identifiers() {
        let root = std::env::temp_dir().join(format!("documi-local-test-{}", Uuid::new_v4()));
        let workspace = Uuid::new_v4().to_string();
        let document = Uuid::new_v4().to_string();
        let version = Uuid::new_v4().to_string();
        let (key, hash) = write_object(&root, &workspace, &document, &version, b"content").unwrap();
        assert!(root.join(key).starts_with(&root));
        assert_eq!(hash, format!("{:x}", Sha256::digest(b"content")));
        assert!(write_object(&root, "../workspace", &document, &version, b"content").is_err());
        fs::remove_dir_all(root).unwrap();
    }
}
