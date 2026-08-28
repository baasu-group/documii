use serde::Serialize;
use std::path::{Path, PathBuf};
use std::process::Command;
use tauri::{AppHandle, Manager};

use crate::local_documents::connect;

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LibreOfficeStatus {
    pub(crate) installed: bool,
    pub(crate) version: Option<String>,
    pub(crate) executable: Option<String>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LibreOfficeExtensionStatus {
    installed: bool,
    version: Option<String>,
    package_available: bool,
    last_error: Option<String>,
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
    detect()
}

pub(crate) fn detect() -> LibreOfficeStatus {
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

fn unopkg(status: &LibreOfficeStatus) -> Option<PathBuf> {
    let executable = status.executable.as_ref()?;
    let path = PathBuf::from(executable);
    if path.is_absolute() {
        let candidate = path.parent()?.join(if cfg!(target_os = "windows") {
            "unopkg.exe"
        } else {
            "unopkg"
        });
        if candidate.is_file() {
            return Some(candidate);
        }
    }
    Some(PathBuf::from(if cfg!(target_os = "windows") {
        "unopkg.exe"
    } else {
        "unopkg"
    }))
}

fn packaged_extension(app: &AppHandle) -> Option<PathBuf> {
    let bundled = app
        .path()
        .resource_dir()
        .ok()?
        .join("libreoffice")
        .join("documi-libreoffice.oxt");
    if bundled.is_file() {
        return Some(bundled);
    }
    let development = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("resources")
        .join("libreoffice")
        .join("documi-libreoffice.oxt");
    development.is_file().then_some(development)
}

fn extension_status(app: &AppHandle) -> LibreOfficeExtensionStatus {
    let package_available = packaged_extension(app).is_some();
    let office = detect();
    let Some(unopkg) = unopkg(&office) else {
        return LibreOfficeExtensionStatus {
            installed: false,
            version: None,
            package_available,
            last_error: Some("libreoffice_not_installed".into()),
        };
    };
    let output = Command::new(unopkg).args(["list", "--shared"]).output();
    match output {
        Ok(output) if output.status.success() => {
            let listing = String::from_utf8_lossy(&output.stdout);
            let installed = listing.contains("org.documi.libreoffice");
            let version = installed
                .then(|| {
                    listing
                        .lines()
                        .find_map(|line| line.trim().strip_prefix("Version:"))
                        .map(str::trim)
                        .map(str::to_owned)
                })
                .flatten();
            LibreOfficeExtensionStatus {
                installed,
                version,
                package_available,
                last_error: None,
            }
        }
        Ok(output) => LibreOfficeExtensionStatus {
            installed: false,
            version: None,
            package_available,
            last_error: Some(format!(
                "unopkg_failed_{}",
                output.status.code().unwrap_or(-1)
            )),
        },
        Err(_) => LibreOfficeExtensionStatus {
            installed: false,
            version: None,
            package_available,
            last_error: Some("unopkg_unavailable".into()),
        },
    }
}

#[tauri::command]
pub fn check_documi_libreoffice_extension(app: AppHandle) -> LibreOfficeExtensionStatus {
    extension_status(&app)
}

#[tauri::command]
pub fn install_documi_libreoffice_extension(
    app: AppHandle,
    confirmed: bool,
) -> Result<LibreOfficeExtensionStatus, String> {
    if !confirmed {
        return Err("installation_confirmation_required".into());
    }
    let office = detect();
    if !office.installed {
        return Err("libreoffice_not_installed".into());
    }
    let executable = unopkg(&office).ok_or("unopkg_unavailable")?;
    let package = packaged_extension(&app).ok_or("extension_package_unavailable")?;
    let output = Command::new(executable)
        .args(["add", "--shared", "--force"])
        .arg(package)
        .output()
        .map_err(|error| error.to_string())?;
    let error = (!output.status.success()).then(|| {
        String::from_utf8_lossy(&output.stderr)
            .chars()
            .take(500)
            .collect::<String>()
    });
    let (connection, _) = connect(&app)?;
    connection
        .execute(
            "INSERT INTO local_editor_extensions(id,installed_version,last_checked_at,last_error)
             VALUES('org.documi.libreoffice',NULL,strftime('%Y-%m-%dT%H:%M:%fZ','now'),?1)
             ON CONFLICT(id) DO UPDATE SET last_checked_at=excluded.last_checked_at,last_error=excluded.last_error",
            [error.as_deref()],
        )
        .map_err(|error| error.to_string())?;
    if let Some(error) = error {
        return Err(if error.trim().is_empty() {
            "extension_install_failed".into()
        } else {
            error
        });
    }
    Ok(extension_status(&app))
}

#[cfg(test)]
mod tests {
    use super::{check_libreoffice, unopkg};

    #[test]
    fn absence_is_a_normal_status() {
        let _ = check_libreoffice();
    }

    #[test]
    fn unopkg_is_derived_without_frontend_arguments() {
        let status = check_libreoffice();
        if status.installed {
            assert!(unopkg(&status).is_some());
        }
    }
}
