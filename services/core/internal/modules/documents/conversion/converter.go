package conversion

import (
	"context"
	"io"
)

type Format string

const (
	FormatPDF Format = "pdf"
)

type Request struct {
	SourceName    string
	SourceFormat  string
	TargetFormat  Format
	Content       io.Reader
	ContentLength int64
}

type Result struct {
	Content     io.ReadCloser
	Size        int64
	ContentType string
}

// Converter is implemented by controlled worker runtimes. The Go API must not
// execute office binaries in-process or accept arbitrary executable arguments.
type Converter interface {
	Available(context.Context) bool
	Convert(context.Context, Request) (Result, error)
}
