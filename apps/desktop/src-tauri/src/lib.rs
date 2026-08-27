mod libreoffice;
mod local_documents;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            libreoffice::check_libreoffice,
            local_documents::local_workspace_initialize,
            local_documents::local_documents_list,
            local_documents::local_document_create,
            local_documents::local_document_get,
            local_documents::local_document_save,
            local_documents::local_document_versions
        ])
        .run(tauri::generate_context!())
        .expect("error while running Documi Desktop");
}
