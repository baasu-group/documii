mod libreoffice;
mod local_documents;
mod local_office;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            libreoffice::check_libreoffice,
            libreoffice::check_documi_libreoffice_extension,
            libreoffice::install_documi_libreoffice_extension,
            local_documents::local_workspace_initialize,
            local_documents::local_documents_list,
            local_documents::local_document_create,
            local_documents::local_document_get,
            local_documents::local_document_save,
            local_documents::local_document_versions,
            local_office::local_office_document_create,
            local_office::local_office_document_import,
            local_office::open_document_in_libreoffice,
            local_office::save_libreoffice_working_copy
        ])
        .run(tauri::generate_context!())
        .expect("error while running Documi Desktop");
}
