package reporting

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"html"
	"strconv"
	"strings"
)

type Dataset struct {
	Title   string
	Headers []string
	Rows    [][]string
}
type Rendered struct {
	Content                     []byte
	MIMEType, Extension, Family string
}
type Renderer interface {
	Format() string
	Render(Dataset) (Rendered, error)
}

type CSVRenderer struct{}

func (CSVRenderer) Format() string { return "csv" }
func (CSVRenderer) Render(data Dataset) (Rendered, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write(data.Headers); err != nil {
		return Rendered{}, err
	}
	for _, row := range data.Rows {
		if err := writer.Write(row); err != nil {
			return Rendered{}, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return Rendered{}, err
	}
	return Rendered{Content: output.Bytes(), MIMEType: "text/csv", Extension: "csv", Family: "text"}, nil
}

type XLSXRenderer struct{}

func (XLSXRenderer) Format() string { return "xlsx" }
func columnName(index int) string {
	value := ""
	for index >= 0 {
		value = string(rune('A'+index%26)) + value
		index = index/26 - 1
	}
	return value
}
func cell(ref, value string) string {
	return `<c r="` + ref + `" t="inlineStr"><is><t>` + html.EscapeString(value) + `</t></is></c>`
}
func (XLSXRenderer) Render(data Dataset) (Rendered, error) {
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	files := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Report" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
	}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	rows := append([][]string{data.Headers}, data.Rows...)
	for i, row := range rows {
		sheet.WriteString(`<row r="` + strconv.Itoa(i+1) + `">`)
		for j, value := range row {
			sheet.WriteString(cell(columnName(j)+strconv.Itoa(i+1), value))
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	files["xl/worksheets/sheet1.xml"] = sheet.String()
	for name, value := range files {
		entry, err := archive.Create(name)
		if err != nil {
			return Rendered{}, err
		}
		if _, err = entry.Write([]byte(value)); err != nil {
			return Rendered{}, err
		}
	}
	if err := archive.Close(); err != nil {
		return Rendered{}, err
	}
	return Rendered{Content: output.Bytes(), MIMEType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Extension: "xlsx", Family: "spreadsheet"}, nil
}

type PDFRenderer struct{}

func (PDFRenderer) Format() string { return "pdf" }
func pdfText(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return '?'
		}
		return r
	}, value)
	return strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(value)
}
func (PDFRenderer) Render(data Dataset) (Rendered, error) {
	lines := []string{data.Title, strings.Join(data.Headers, " | ")}
	for _, row := range data.Rows {
		lines = append(lines, strings.Join(row, " | "))
	}
	if len(lines) > 48 {
		lines = lines[:48]
		lines = append(lines, "Report truncated in PDF view; export CSV/XLSX for all rows.")
	}
	var stream strings.Builder
	stream.WriteString("BT /F1 10 Tf 48 760 Td 14 TL ")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("T* ")
		}
		stream.WriteString("(" + pdfText(line) + ") Tj ")
	}
	stream.WriteString("ET")
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>", fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", stream.Len(), stream.String()), "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"}
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, output.Len())
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&output, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&output, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return Rendered{Content: output.Bytes(), MIMEType: "application/pdf", Extension: "pdf", Family: "pdf"}, nil
}
