package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrInvalidKey = errors.New("invalid storage key")

type Filesystem struct {
	root     string
	maxBytes int64
}

func NewFilesystem(root string, maxBytes int64) (*Filesystem, error) {
	absolute, err := filepath.Abs(root)
	if err != nil || strings.TrimSpace(root) == "" || maxBytes <= 0 {
		return nil, fmt.Errorf("filesystem root and positive size limit are required")
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Filesystem{root: absolute, maxBytes: maxBytes}, nil
}

func (provider *Filesystem) path(key string) (string, error) {
	if key == "" || filepath.IsAbs(key) || strings.ContainsRune(key, '\x00') {
		return "", ErrInvalidKey
	}
	clean := filepath.Clean(key)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrInvalidKey
	}
	result := filepath.Join(provider.root, clean)
	relative, err := filepath.Rel(provider.root, result)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrInvalidKey
	}
	return result, nil
}

func (provider *Filesystem) Put(_ context.Context, key string, content io.Reader) (Object, error) {
	target, err := provider.path(key)
	if err != nil {
		return Object{}, err
	}
	if _, err := os.Stat(target); err == nil {
		return Object{}, fmt.Errorf("storage object already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Object{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return Object{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".documi-*")
	if err != nil {
		return Object{}, err
	}
	name := temporary.Name()
	defer os.Remove(name)
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(content, provider.maxBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return Object{}, copyErr
	}
	if closeErr != nil {
		return Object{}, closeErr
	}
	if size > provider.maxBytes {
		return Object{}, fmt.Errorf("storage object exceeds %d bytes", provider.maxBytes)
	}
	if err := os.Rename(name, target); err != nil {
		return Object{}, err
	}
	return Object{Key: key, Size: size, SHA256: fmt.Sprintf("%x", hash.Sum(nil))}, nil
}

func (provider *Filesystem) Get(_ context.Context, key string) (io.ReadCloser, Object, error) {
	path, err := provider.path(key)
	if err != nil {
		return nil, Object{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Object{}, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, Object{}, err
	}
	return file, Object{Key: key, Size: info.Size()}, nil
}
func (provider *Filesystem) Delete(_ context.Context, key string) error {
	path, err := provider.path(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
func (provider *Filesystem) Move(_ context.Context, from, to string) error {
	source, err := provider.path(from)
	if err != nil {
		return err
	}
	target, err := provider.path(to)
	if err != nil {
		return err
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("storage object already exists")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	return os.Rename(source, target)
}
func (provider *Filesystem) Exists(_ context.Context, key string) (bool, error) {
	path, err := provider.path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
