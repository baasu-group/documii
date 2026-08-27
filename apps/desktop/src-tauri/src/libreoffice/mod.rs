use serde::Serialize;
use std::path::{Path, PathBuf};
use std::process::Command;

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LibreOfficeStatus {
    installed: bool,
    version: Option<String>,
    executable: Option<String>,
}

fn known_candidates() -> Vec<PathBuf> {
    #[cfg(target_os = "macos")]
    let candidates = vec![PathBuf::from(
        "/Applications/LibreOffice.app/Contents/MacOS/soffice",
    )];

    #[cfg(target_os = "windows")]
    let candidates = ["PROGRAMFILES", "PROGRAMFILES(X86)"]
        .into_iter()
        .filter_map(std::env::var_os)
        .map(|root| PathBuf::from(root).join("LibreOffice/program/soffice.exe"))
        .collect();

    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    let candidates = Vec::new();

    candidates
}

fn inspect(executable: &Path) -> Option<LibreOfficeStatus> {
    let output = Command::new(executable).arg("--version").output().ok()?;
    if !output.status.success() {
        return None;
    }
    let version = String::from_utf8_lossy(&output.stdout).trim().to_owned();
    Some(LibreOfficeStatus {
        installed: true,
        version: (!version.is_empty()).then_some(version),
        executable: Some(executable.to_string_lossy().into_owned()),
    })
}

fn inspect_path_command(command: &str) -> Option<LibreOfficeStatus> {
    let output = Command::new(command).arg("--version").output().ok()?;
    if !output.status.success() {
        return None;
    }
    let version = String::from_utf8_lossy(&output.stdout).trim().to_owned();
    Some(LibreOfficeStatus {
        installed: true,
        version: (!version.is_empty()).then_some(version),
        executable: Some(command.to_owned()),
    })
}

#[tauri::command]
pub fn check_libreoffice() -> LibreOfficeStatus {
    for candidate in known_candidates() {
        if candidate.is_file() {
            if let Some(status) = inspect(&candidate) {
                return status;
            }
        }
    }

    for command in ["soffice", "libreoffice"] {
        if let Some(status) = inspect_path_command(command) {
            return status;
        }
    }

    LibreOfficeStatus {
        installed: false,
        version: None,
        executable: None,
    }
}

#[cfg(test)]
mod tests {
    use super::check_libreoffice;

    #[test]
    fn absence_is_a_normal_status() {
        let _ = check_libreoffice();
    }
}
