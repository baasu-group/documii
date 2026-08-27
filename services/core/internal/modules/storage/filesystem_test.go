package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestFilesystemRejectsTraversalAndNeverOverwrites(t *testing.T) {
	provider, err := NewFilesystem(t.TempDir(), 16)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Put(context.Background(), "../secret", strings.NewReader("x")); err == nil {
		t.Fatal("traversal accepted")
	}
	object, err := provider.Put(context.Background(), "doc/version/content", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if object.Size != 5 || len(object.SHA256) != 64 {
		t.Fatalf("object=%#v", object)
	}
	if _, err := provider.Put(context.Background(), object.Key, strings.NewReader("changed")); err == nil {
		t.Fatal("overwrite accepted")
	}
	content, _, err := provider.Get(context.Background(), object.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer content.Close()
	value, _ := io.ReadAll(content)
	if string(value) != "hello" {
		t.Fatalf("content=%q", value)
	}
}

func TestFilesystemEnforcesLimit(t *testing.T) {
	provider, _ := NewFilesystem(t.TempDir(), 4)
	if _, err := provider.Put(context.Background(), "safe", strings.NewReader("12345")); err == nil {
		t.Fatal("oversized object accepted")
	}
}
