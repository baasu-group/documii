package templates

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestPinnedTemplates(t *testing.T) {
	for _, family := range []string{"text", "spreadsheet", "presentation"} {
		template, err := Get(family)
		if err != nil {
			t.Fatalf("Get(%q): %v", family, err)
		}
		archive, err := zip.NewReader(bytes.NewReader(template.Content), int64(len(template.Content)))
		if err != nil {
			t.Fatalf("%s is not a valid Office package: %v", family, err)
		}
		if len(archive.File) == 0 {
			t.Fatalf("%s template contains no package parts", family)
		}
		for _, part := range archive.File {
			if part.Name == "word/vbaProject.bin" || part.Name == "xl/vbaProject.bin" || part.Name == "ppt/vbaProject.bin" {
				t.Fatalf("%s template unexpectedly contains macros", family)
			}
		}
	}
}
