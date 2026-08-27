package documents

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"
)

func officeTestArchive(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, content := range parts {
		part, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestValidateOfficeArchiveRejectsDisguisedMacrosAndTraversal(t *testing.T) {
	format := officeFormats[".docx"]
	for _, parts := range []map[string]string{
		{"[Content_Types].xml": `<Types><Override ContentType="application/vnd.ms-word.document.macroEnabled.main+xml"/></Types>`},
		{"[Content_Types].xml": `<Types/>`, "word/vbaProject.bin": "macro"},
		{"[Content_Types].xml": `<Types/>`, "../escaped": "content"},
	} {
		if err := validateOfficeArchive(officeTestArchive(t, parts), format); !errors.Is(err, ErrInvalidDocument) {
			t.Fatalf("validateOfficeArchive() error = %v", err)
		}
	}
}

func TestValidateOfficeArchiveAcceptsMinimalSafePackage(t *testing.T) {
	content := officeTestArchive(t, map[string]string{"[Content_Types].xml": `<Types/>`, "word/document.xml": `<document/>`})
	if err := validateOfficeArchive(content, officeFormats[".docx"]); err != nil {
		t.Fatalf("validateOfficeArchive() error = %v", err)
	}
}

func TestValidateOfficeArchiveRequiresExpectedPackageStructure(t *testing.T) {
	docx := officeFormats[".docx"]
	if err := validateOfficeArchive(officeTestArchive(t, map[string]string{"word/document.xml": "<w/>"}), docx); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("missing content types error = %v", err)
	}
	odt := officeFormats[".odt"]
	if err := validateOfficeArchive(officeTestArchive(t, map[string]string{"mimetype": odt.ODFMIME}), odt); err != nil {
		t.Fatalf("valid ODT package error = %v", err)
	}
	if err := validateOfficeArchive([]byte("not-a-zip"), docx); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("invalid ZIP error = %v", err)
	}
}
