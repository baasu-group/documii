# Documi LibreOffice extension

This first-party UNO extension adds fixed **Save to Documi** and **Show in Documi** commands. It never receives a Documi access token, database path, or storage path. The desktop application gives it only an opaque working-copy identifier; the extension opens a `documi://` deep link and Tauri resolves and validates the identifier locally.

Build `src/` reproducibly into `documi-libreoffice.oxt` in CI and place the release artifact under the desktop bundle's `libreoffice/` resources. Generated `.oxt` archives are deliberately ignored and are not committed.
