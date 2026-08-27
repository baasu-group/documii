package templates

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
)

type Template struct {
	Family    string
	Extension string
	MIMEType  string
	SHA256    string
	Content   []byte
}

// These assets are base64 encoded only so they can be reviewed and patched as
// text. The decoded Office package bytes are verified before every use.
//
//go:embed assets/blank.docx.b64
var blankDOCX string

//go:embed assets/blank.xlsx.b64
var blankXLSX string

//go:embed assets/blank.pptx.b64
var blankPPTX string

var specifications = map[string]struct {
	extension string
	mimeType  string
	sha256    string
	encoded   *string
}{
	"text": {
		extension: ".docx",
		mimeType:  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		sha256:    "560b0e299a592f6b7d1ec9abab98648f1d92bf8fa684ce544a0424fdde798903",
		encoded:   &blankDOCX,
	},
	"spreadsheet": {
		extension: ".xlsx",
		mimeType:  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		sha256:    "32ce7d6f125c2cdde13a61c94c6fd0f05e96dea7ab73c0f7ee6b67b6dd37cf04",
		encoded:   &blankXLSX,
	},
	"presentation": {
		extension: ".pptx",
		mimeType:  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		sha256:    "dd4746eb40918140f90310247c04e3a3aa7232689de2e2f37670a10e562a4454",
		encoded:   &blankPPTX,
	},
}

func Get(family string) (Template, error) {
	specification, ok := specifications[family]
	if !ok {
		return Template{}, fmt.Errorf("unsupported Office template family %q", family)
	}
	encoded := strings.Map(func(value rune) rune {
		if value == ' ' || value == '\n' || value == '\r' || value == '\t' {
			return -1
		}
		return value
	}, *specification.encoded)
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Template{}, fmt.Errorf("decode %s template: %w", family, err)
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(content))
	if actual != specification.sha256 {
		return Template{}, fmt.Errorf("%s template checksum mismatch: expected %s, got %s", family, specification.sha256, actual)
	}
	return Template{Family: family, Extension: specification.extension, MIMEType: specification.mimeType, SHA256: specification.sha256, Content: content}, nil
}

func Ready() error {
	for _, family := range []string{"text", "spreadsheet", "presentation"} {
		if _, err := Get(family); err != nil {
			return err
		}
	}
	return nil
}
