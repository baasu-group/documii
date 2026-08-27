package collabora

import "testing"

func TestSafeOfficeName(t *testing.T) {
	name := safeOfficeName("../Quarterly: report.doc", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if name != "..-Quarterly- report.docx" {
		t.Fatalf("safeOfficeName() = %q", name)
	}
	if !safeFileName(name) {
		t.Fatalf("safeOfficeName() returned unsafe name %q", name)
	}
}
