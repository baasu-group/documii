package secrets

import (
	"bytes"
	"github.com/google/uuid"
	"testing"
)

func TestSealUsesAADAndRandomNonce(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	connection := uuid.New()
	plain := []byte("refresh-token")
	first, n1, err := seal(key, plain, aad(connection, "oauth"))
	if err != nil {
		t.Fatal(err)
	}
	second, n2, err := seal(key, plain, aad(connection, "oauth"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(n1, n2) || bytes.Equal(first, second) {
		t.Fatal("encryption must use a fresh nonce")
	}
	got, err := open(key, first, n1, aad(connection, "oauth"))
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("round trip failed: %v", err)
	}
	if _, err := open(key, first, n1, aad(connection, "other")); err == nil {
		t.Fatal("ciphertext must be bound to connection and secret name")
	}
}
func TestNewStoreRejectsInvalidKey(t *testing.T) {
	if _, err := NewStore(nil, 1, map[int][]byte{1: make([]byte, 32)}); err == nil {
		t.Fatal("expected database error")
	}
}
func TestCipherVersionAndLabelBinding(t *testing.T) {
	cipher, err := NewCipher(2, map[int][]byte{1: bytes.Repeat([]byte{1}, 32), 2: bytes.Repeat([]byte{2}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, nonce, version, err := cipher.Encrypt("google-oauth-state", []byte("verifier"))
	if err != nil || version != 2 {
		t.Fatalf("encrypt: version=%d err=%v", version, err)
	}
	plain, err := cipher.Decrypt("google-oauth-state", encrypted, nonce, version)
	if err != nil || string(plain) != "verifier" {
		t.Fatalf("decrypt: %q err=%v", plain, err)
	}
	if _, err := cipher.Decrypt("wrong-label", encrypted, nonce, version); err == nil {
		t.Fatal("ciphertext was not bound to its purpose")
	}
}
