package collabora

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSafeOfficeName(t *testing.T) {
	name := safeOfficeName("../Quarterly: report.doc", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if name != "..-Quarterly- report.docx" {
		t.Fatalf("safeOfficeName() = %q", name)
	}
	if !safeFileName(name) {
		t.Fatalf("safeOfficeName() returned unsafe name %q", name)
	}
}

func TestSafeOfficeNameHandlesWindowsReservedAndUnicodeNames(t *testing.T) {
	if name := safeOfficeName("CON.doc", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"); name != "_CON.docx" {
		t.Fatalf("reserved safeOfficeName() = %q", name)
	}
	name := safeOfficeName(strings.Repeat("文", 200), "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if !utf8.ValidString(name) || utf8.RuneCountInString(strings.TrimSuffix(name, ".docx")) != 180 {
		t.Fatalf("unicode safeOfficeName() returned invalid result")
	}
}
