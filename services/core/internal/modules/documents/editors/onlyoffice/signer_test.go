package onlyoffice

import "testing"

func TestSignerSignsAndValidatesServerConfiguration(t *testing.T) {
	signer, err := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.Sign(map[string]any{"document": map[string]string{"key": "ver_1"}})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := signer.Verify(token, &payload); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if _, ok := payload["secret"]; ok {
		t.Fatal("signed payload exposed the secret")
	}
}

func TestSignerRejectsTampering(t *testing.T) {
	signer, _ := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	token, _ := signer.Sign(map[string]string{"key": "ver_1"})
	if err := signer.Verify(token+"tampered", &map[string]any{}); err == nil {
		t.Fatal("Verify() accepted a tampered token")
	}
}
