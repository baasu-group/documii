package editors

import "strings"

var mimeFamilies = map[string]DocumentFamily{
	"application/vnd.documi.document+json": FamilyNative,
	"application/msword":                   FamilyText,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   FamilyText,
	"application/vnd.oasis.opendocument.text":                                   FamilyText,
	"application/vnd.ms-excel":                                                  FamilySpreadsheet,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         FamilySpreadsheet,
	"application/vnd.oasis.opendocument.spreadsheet":                            FamilySpreadsheet,
	"application/vnd.ms-powerpoint":                                             FamilyPresentation,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": FamilyPresentation,
	"application/vnd.oasis.opendocument.presentation":                           FamilyPresentation,
	"application/pdf": FamilyPDF,
}

func FamilyForMIME(mimeType string) (DocumentFamily, bool) {
	family, ok := mimeFamilies[strings.ToLower(strings.TrimSpace(strings.SplitN(mimeType, ";", 2)[0]))]
	return family, ok
}
