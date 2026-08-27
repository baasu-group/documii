package reporting

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"testing"
)

var sample = Dataset{Title: "Usage", Headers: []string{"Name", "Count"}, Rows: [][]string{{"Documents", "4"}, {"Storage", "598"}}}

func TestCSVRenderer(t *testing.T) {
	r, err := (CSVRenderer{}).Render(sample)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(r.Content)).ReadAll()
	if err != nil || len(rows) != 3 {
		t.Fatalf("invalid CSV: %v", err)
	}
}
func TestXLSXRenderer(t *testing.T) {
	r, err := (XLSXRenderer{}).Render(sample)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(r.Content), int64(len(r.Content)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range archive.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			body, _ := file.Open()
			value, _ := io.ReadAll(body)
			found = bytes.Contains(value, []byte("Documents"))
		}
	}
	if !found {
		t.Fatal("worksheet missing")
	}
}
func TestPDFRenderer(t *testing.T) {
	r, err := (PDFRenderer{}).Render(sample)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(r.Content, []byte("%PDF-1.4")) || !bytes.Contains(r.Content, []byte("Usage")) {
		t.Fatal("invalid PDF")
	}
}
