package collabora

import (
	"errors"
	"testing"
	"time"
)

func TestTokenExpires(t *testing.T) {
	service, err := NewTokenService([]byte("0123456789abcdef0123456789abcdef"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0)
	service.now = func() time.Time { return now }
	token, _, err := service.Issue("doc_1", "user_1", true, true)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now.Add(time.Minute) }
	if _, err := service.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenCarriesPersistedSessionBinding(t *testing.T) {
	service, err := NewTokenService([]byte("0123456789abcdef0123456789abcdef"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := service.IssueForSession("doc_1", "ver_1", "user_1", "session_1", true, true)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SessionID != "session_1" || claims.FileID != "doc_1" || claims.VersionID != "ver_1" {
		t.Fatalf("Verify() claims = %#v", claims)
	}
}
