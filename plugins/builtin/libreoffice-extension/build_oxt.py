#!/usr/bin/env python3
"""Build the first-party OXT with deterministic timestamps and ordering."""

from pathlib import Path
from zipfile import ZIP_DEFLATED, ZipFile, ZipInfo

ROOT = Path(__file__).resolve().parent
SOURCE = ROOT / "src"
OUTPUT = ROOT.parent.parent.parent / "apps" / "desktop" / "src-tauri" / "resources" / "libreoffice" / "documi-libreoffice.oxt"


def main():
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    with ZipFile(OUTPUT, "w", compression=ZIP_DEFLATED, compresslevel=9) as archive:
        for source in sorted(path for path in SOURCE.rglob("*") if path.is_file()):
            relative = source.relative_to(SOURCE).as_posix()
            info = ZipInfo(relative, date_time=(1980, 1, 1, 0, 0, 0))
            info.compress_type = ZIP_DEFLATED
            info.external_attr = 0o644 << 16
            archive.writestr(info, source.read_bytes())
    print(OUTPUT)


if __name__ == "__main__":
    main()
