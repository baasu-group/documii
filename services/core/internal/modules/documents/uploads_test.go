package documents

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"
)

func officeArchive(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, value := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestValidateOfficeArchive(t *testing.T) {
	docx := officeFormats[".docx"]
	if err := validateOfficeArchive(officeArchive(t, map[string]string{"[Content_Types].xml": "<Types/>"}), docx); err != nil {
		t.Fatal(err)
	}
	if err := validateOfficeArchive(officeArchive(t, map[string]string{"word/document.xml": "<w/>"}), docx); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("missing content types error=%v", err)
	}
	odt := officeFormats[".odt"]
	if err := validateOfficeArchive(officeArchive(t, map[string]string{"mimetype": odt.ODFMIME}), odt); err != nil {
		t.Fatal(err)
	}
	if err := validateOfficeArchive([]byte("not-a-zip"), docx); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("invalid ZIP error=%v", err)
	}
}
