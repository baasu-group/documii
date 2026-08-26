package storage

import (
	"context"
	"io"
)

type Object struct {
	Key         string
	Size        int64
	ContentType string
	SHA256      string
}

type Provider interface {
	Put(context.Context, string, io.Reader) (Object, error)
	Get(context.Context, string) (io.ReadCloser, Object, error)
	Delete(context.Context, string) error
	Move(context.Context, string, string) error
	Exists(context.Context, string) (bool, error)
}
