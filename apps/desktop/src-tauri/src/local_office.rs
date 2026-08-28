use base64::{engine::general_purpose::STANDARD, Engine as _};
use rusqlite::{params, OptionalExtension};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs::{self, File};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::Command;
use tauri::AppHandle;
use uuid::Uuid;

use crate::libreoffice;
use crate::local_documents::{audit, connect, validate_uuid, LocalDocument};

const MAX_OFFICE_BYTES: usize = 100 * 1024 * 1024;
const DOCX: &str = include_str!(
    "../../../../services/core/internal/modules/documents/templates/assets/blank.docx.b64"
);
const XLSX: &str = include_str!(
    "../../../../services/core/internal/modules/documents/templates/assets/blank.xlsx.b64"
);
const PPTX: &str = include_str!(
    "../../../../services/core/internal/modules/documents/templates/assets/blank.pptx.b64"
);

#[derive(Clone, Copy)]
struct OfficeFormat {
    family: &'static str,
    extension: &'static str,
    mime_type: &'static str,
    checksum: &'static str,
    template: &'static str,
}

fn format(family: &str) -> Result<OfficeFormat, String> {
    match family {
        "text" => Ok(OfficeFormat {
            family: "text",
            extension: ".docx",
            mime_type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
            checksum: "560b0e299a592f6b7d1ec9abab98648f1d92bf8fa684ce544a0424fdde798903",
            template: DOCX,
        }),
        "spreadsheet" => Ok(OfficeFormat {
            family: "spreadsheet",
            extension: ".xlsx",
            mime_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            checksum: "32ce7d6f125c2cdde13a61c94c6fd0f05e96dea7ab73c0f7ee6b67b6dd37cf04",
            template: XLSX,
        }),
        "presentation" => Ok(OfficeFormat {
            family: "presentation",
            extension: ".pptx",
            mime_type: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
            checksum: "dd4746eb40918140f90310247c04e3a3aa7232689de2e2f37670a10e562a4454",
            template: PPTX,
        }),
        _ => Err("unsupported_document_family".into()),
    }
}

fn template_bytes(spec: OfficeFormat) -> Result<Vec<u8>, String> {
    let encoded: String = spec
        .template
        .chars()
        .filter(|value| !value.is_whitespace())
        .collect();
    let bytes = STANDARD
        .decode(encoded)
        .map_err(|_| "office_template_invalid".to_string())?;
    if format!("{:x}", Sha256::digest(&bytes)) != spec.checksum {
        return Err("office_template_checksum_mismatch".into());
    }
    Ok(bytes)
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateOfficeRequest {
    workspace_id: String,
    family: String,
    title: String,
    idempotency_key: String,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportOfficeRequest {
    workspace_id: String,
    family: String,
    title: String,
    file_name: String,
    content: Vec<u8>,
    idempotency_key: String,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SaveWorkingCopyRequest {
    working_copy_id: String,
    idempotency_key: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct WorkingCopy {
    id: String,
    document_id: String,
    base_version_id: String,
    file_name: String,
}

fn safe_title(value: &str) -> Result<String, String> {
    let value = value.trim();
    if value.is_empty() || value.chars().count() > 255 || value.contains(['/', '\\', '\0']) {
        return Err("invalid_title".into());
    }
    Ok(value.to_owned())
}

fn object_path(
    root: &Path,
    workspace_id: &str,
    document_id: &str,
    version_id: &str,
    extension: &str,
) -> Result<(PathBuf, PathBuf), String> {
    for value in [workspace_id, document_id, version_id] {
        validate_uuid(value)?;
    }
    let relative = PathBuf::from("documents")
        .join(workspace_id)
        .join(document_id)
        .join(version_id)
        .join(format!("document{extension}"));
    Ok((root.join(&relative), relative))
}

fn write_object(
    root: &Path,
    workspace_id: &str,
    document_id: &str,
    version_id: &str,
    extension: &str,
    content: &[u8],
) -> Result<(String, String), String> {
    if content.is_empty() || content.len() > MAX_OFFICE_BYTES || !content.starts_with(b"PK") {
        return Err("invalid_office_document".into());
    }
    let (target, relative) = object_path(root, workspace_id, document_id, version_id, extension)?;
    let parent = target.parent().ok_or("invalid_storage_path")?;
    fs::create_dir_all(parent).map_err(|error| error.to_string())?;
    if target.exists() {
        return Err("storage_object_exists".into());
    }
    let temporary = parent.join(format!(".{}.tmp", Uuid::new_v4()));
    let result = (|| -> Result<(), String> {
        let mut file = File::create(&temporary).map_err(|error| error.to_string())?;
        file.write_all(content).map_err(|error| error.to_string())?;
        file.sync_all().map_err(|error| error.to_string())?;
        fs::rename(&temporary, &target).map_err(|error| error.to_string())
    })();
    if let Err(error) = result {
        let _ = fs::remove_file(temporary);
        return Err(error);
    }
    Ok((
        relative.to_string_lossy().into_owned(),
        format!("{:x}", Sha256::digest(content)),
    ))
}

fn read_office_document(
    connection: &rusqlite::Connection,
    document_id: &str,
) -> Result<LocalDocument, String> {
    validate_uuid(document_id)?;
    connection
        .query_row(
            "SELECT id,workspace_id,title,family,mime_type,editor_provider,storage_mode,
                    current_version_id,created_at,updated_at
             FROM local_office_documents WHERE id=?1",
            [document_id],
            |row| {
                Ok(LocalDocument {
                    id: row.get(0)?,
                    workspace_id: row.get(1)?,
                    title: row.get(2)?,
                    family: row.get(3)?,
                    mime_type: row.get(4)?,
                    editor_provider: row.get(5)?,
                    storage_mode: row.get(6)?,
                    current_version_id: row.get(7)?,
                    created_at: row.get(8)?,
                    updated_at: row.get(9)?,
                    content: None,
                })
            },
        )
        .map_err(|error| error.to_string())
}

fn create_document(
    app: AppHandle,
    workspace_id: String,
    title: String,
    spec: OfficeFormat,
    bytes: Vec<u8>,
    idempotency_key: String,
    operation: &str,
) -> Result<LocalDocument, String> {
    validate_uuid(&workspace_id)?;
    let title = safe_title(&title)?;
    if idempotency_key.trim().is_empty() || idempotency_key.len() > 255 {
        return Err("idempotency_key_required".into());
    }
    let (mut connection, paths) = connect(&app)?;
    if let Some(id) = connection
        .query_row(
            "SELECT resource_id FROM local_idempotency WHERE key=?1 AND operation=?2",
            params![idempotency_key, operation],
            |row| row.get::<_, String>(0),
        )
        .optional()
        .map_err(|error| error.to_string())?
    {
        return read_office_document(&connection, &id);
    }
    let document_id = Uuid::new_v4().to_string();
    let version_id = Uuid::new_v4().to_string();
    let object_id = Uuid::new_v4().to_string();
    let (key, hash) = write_object(
        &paths.root,
        &workspace_id,
        &document_id,
        &version_id,
        spec.extension,
        &bytes,
    )?;
    let result = (|| -> Result<(), String> {
        let transaction = connection
            .transaction()
            .map_err(|error| error.to_string())?;
        transaction.execute(
            "INSERT INTO local_office_documents(id,workspace_id,title,family,mime_type,editor_provider,storage_mode,current_version_id)
             VALUES(?1,?2,?3,?4,?5,'libreoffice-desktop','managed',?6)",
            params![document_id, workspace_id, title, spec.family, spec.mime_type, version_id],
        ).map_err(|error| error.to_string())?;
        transaction.execute(
            "INSERT INTO local_storage_objects(id,storage_key,sha256,size,content_type) VALUES(?1,?2,?3,?4,?5)",
            params![object_id, key, hash, bytes.len() as i64, spec.mime_type],
        ).map_err(|error| error.to_string())?;
        transaction.execute(
            "INSERT INTO local_office_versions(id,document_id,storage_object_id,version_number,actor_id)
             VALUES(?1,?2,?3,1,'local-owner')",
            params![version_id, document_id, object_id],
        ).map_err(|error| error.to_string())?;
        transaction
            .execute(
                "INSERT INTO local_idempotency(key,operation,resource_id) VALUES(?1,?2,?3)",
                params![idempotency_key, operation, document_id],
            )
            .map_err(|error| error.to_string())?;
        audit(
            &transaction,
            "document.created",
            &workspace_id,
            &document_id,
        )?;
        transaction.commit().map_err(|error| error.to_string())
    })();
    if let Err(error) = result {
        let _ = fs::remove_file(paths.root.join(key));
        return Err(error);
    }
    read_office_document(&connection, &document_id)
}

#[tauri::command]
pub fn local_office_document_create(
    app: AppHandle,
    request: CreateOfficeRequest,
) -> Result<LocalDocument, String> {
    let spec = format(&request.family)?;
    let bytes = template_bytes(spec)?;
    create_document(
        app,
        request.workspace_id,
        request.title,
        spec,
        bytes,
        request.idempotency_key,
        "office_create",
    )
}

#[tauri::command]
pub fn local_office_document_import(
    app: AppHandle,
    request: ImportOfficeRequest,
) -> Result<LocalDocument, String> {
    let spec = format(&request.family)?;
    let file_name = request.file_name.to_ascii_lowercase();
    if !file_name.ends_with(spec.extension)
        || file_name.ends_with(".docm")
        || file_name.ends_with(".xlsm")
        || file_name.ends_with(".pptm")
    {
        return Err("unsupported_office_file".into());
    }
    create_document(
        app,
        request.workspace_id,
        request.title,
        spec,
        request.content,
        request.idempotency_key,
        "office_import",
    )
}

#[tauri::command]
pub fn open_document_in_libreoffice(
    app: AppHandle,
    document_id: String,
) -> Result<WorkingCopy, String> {
    let status = libreoffice::detect();
    let executable = status.executable.ok_or("libreoffice_not_installed")?;
    let (mut connection, paths) = connect(&app)?;
    let document = read_office_document(&connection, &document_id)?;
    let spec = format(&document.family)?;
    let (storage_key, expected_hash, expected_size): (String, String, i64) = connection
        .query_row(
            "SELECT o.storage_key,o.sha256,o.size FROM local_office_versions v
             JOIN local_storage_objects o ON o.id=v.storage_object_id WHERE v.id=?1",
            [&document.current_version_id],
            |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
        )
        .map_err(|error| error.to_string())?;
    let source = paths.root.join(&storage_key);
    let bytes = fs::read(&source).map_err(|_| "document_content_missing".to_string())?;
    if bytes.len() as i64 != expected_size
        || format!("{:x}", Sha256::digest(&bytes)) != expected_hash
    {
        return Err("document_content_corrupt".into());
    }
    let working_copy_id = Uuid::new_v4().to_string();
    let relative = PathBuf::from("editor-workspace")
        .join(&document.id)
        .join(&document.current_version_id)
        .join(&working_copy_id)
        .join(format!("document{}", spec.extension));
    let target = paths.root.join(&relative);
    fs::create_dir_all(target.parent().ok_or("invalid_working_copy_path")?)
        .map_err(|error| error.to_string())?;
    fs::write(&target, bytes).map_err(|error| error.to_string())?;
    let transaction = connection
        .transaction()
        .map_err(|error| error.to_string())?;
    transaction
        .execute(
            "INSERT INTO local_office_working_copies(id,document_id,base_version_id,relative_path)
         VALUES(?1,?2,?3,?4)",
            params![
                working_copy_id,
                document.id,
                document.current_version_id,
                relative.to_string_lossy()
            ],
        )
        .map_err(|error| error.to_string())?;
    audit(
        &transaction,
        "document.libreoffice.opened",
        &document.workspace_id,
        &document.id,
    )?;
    transaction.commit().map_err(|error| error.to_string())?;
    Command::new(executable)
        .args(["--nologo", "--nodefault", "--nofirststartwizard"])
        .arg(&target)
        .spawn()
        .map_err(|error| error.to_string())?;
    Ok(WorkingCopy {
        id: working_copy_id,
        document_id: document.id,
        base_version_id: document.current_version_id,
        file_name: format!("document{}", spec.extension),
    })
}

#[tauri::command]
pub fn save_libreoffice_working_copy(
    app: AppHandle,
    request: SaveWorkingCopyRequest,
) -> Result<LocalDocument, String> {
    validate_uuid(&request.working_copy_id)?;
    if request.idempotency_key.trim().is_empty() || request.idempotency_key.len() > 255 {
        return Err("idempotency_key_required".into());
    }
    let (mut connection, paths) = connect(&app)?;
    if let Some(document_id) = connection.query_row(
        "SELECT resource_id FROM local_idempotency WHERE key=?1 AND operation='office_save_back'",
        [&request.idempotency_key], |row| row.get::<_, String>(0)
    ).optional().map_err(|error| error.to_string())? {
        return read_office_document(&connection, &document_id);
    }
    let (document_id, base_version_id, relative_path): (String, String, String) = connection
        .query_row(
            "SELECT document_id,base_version_id,relative_path FROM local_office_working_copies
         WHERE id=?1 AND status='active'",
            [&request.working_copy_id],
            |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
        )
        .map_err(|_| "working_copy_not_found".to_string())?;
    let document = read_office_document(&connection, &document_id)?;
    if document.current_version_id != base_version_id {
        return Err("document_version_conflict".into());
    }
    let spec = format(&document.family)?;
    let working_root = paths.root.join("editor-workspace");
    let target = paths.root.join(&relative_path);
    let canonical_root = working_root
        .canonicalize()
        .map_err(|_| "invalid_working_copy_path".to_string())?;
    let canonical_target = target
        .canonicalize()
        .map_err(|_| "working_copy_missing".to_string())?;
    if !canonical_target.starts_with(&canonical_root) {
        return Err("invalid_working_copy_path".into());
    }
    let bytes = fs::read(&canonical_target).map_err(|_| "working_copy_missing".to_string())?;
    let version_id = Uuid::new_v4().to_string();
    let object_id = Uuid::new_v4().to_string();
    let (key, hash) = write_object(
        &paths.root,
        &document.workspace_id,
        &document.id,
        &version_id,
        spec.extension,
        &bytes,
    )?;
    let result = (|| -> Result<(), String> {
        let transaction = connection
            .transaction()
            .map_err(|error| error.to_string())?;
        let version_number: i64 = transaction.query_row(
            "SELECT COALESCE(MAX(version_number),0)+1 FROM local_office_versions WHERE document_id=?1",
            [&document.id], |row| row.get(0)
        ).map_err(|error| error.to_string())?;
        transaction.execute(
            "INSERT INTO local_storage_objects(id,storage_key,sha256,size,content_type) VALUES(?1,?2,?3,?4,?5)",
            params![object_id,key,hash,bytes.len() as i64,spec.mime_type],
        ).map_err(|error| error.to_string())?;
        transaction.execute(
            "INSERT INTO local_office_versions(id,document_id,parent_version_id,storage_object_id,version_number,actor_id)
             VALUES(?1,?2,?3,?4,?5,'local-owner')",
            params![version_id,document.id,base_version_id,object_id,version_number],
        ).map_err(|error| error.to_string())?;
        let changed = transaction.execute(
            "UPDATE local_office_documents SET current_version_id=?1,updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
             WHERE id=?2 AND current_version_id=?3",
            params![version_id,document.id,base_version_id],
        ).map_err(|error| error.to_string())?;
        if changed != 1 {
            return Err("document_version_conflict".into());
        }
        transaction.execute(
            "UPDATE local_office_working_copies SET base_version_id=?1,updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?2",
            params![version_id,request.working_copy_id],
        ).map_err(|error| error.to_string())?;
        transaction.execute(
            "INSERT INTO local_idempotency(key,operation,resource_id) VALUES(?1,'office_save_back',?2)",
            params![request.idempotency_key,document.id],
        ).map_err(|error| error.to_string())?;
        audit(
            &transaction,
            "document.libreoffice.saved",
            &document.workspace_id,
            &document.id,
        )?;
        audit(
            &transaction,
            "document.editor.saved",
            &document.workspace_id,
            &document.id,
        )?;
        transaction.commit().map_err(|error| error.to_string())
    })();
    if let Err(error) = result {
        let _ = fs::remove_file(paths.root.join(key));
        return Err(error);
    }
    read_office_document(&connection, &document.id)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn templates_match_the_cross_language_checksums() {
        for family in ["text", "spreadsheet", "presentation"] {
            let spec = format(family).unwrap();
            let bytes = template_bytes(spec).unwrap();
            assert!(bytes.starts_with(b"PK"));
        }
    }

    #[test]
    fn unsafe_titles_and_paths_are_rejected() {
        assert!(safe_title("../report").is_err());
        assert!(safe_title("valid report").is_ok());
    }
}
