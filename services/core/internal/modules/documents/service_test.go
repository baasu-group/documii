package documents

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateNative(t *testing.T) {
	valid := []byte(`{"format":"documi-json-v1","content":{"type":"doc","content":[]}}`)
	result, err := validateNative(valid)
	if err != nil || !strings.Contains(string(result), NativeFormat) {
		t.Fatalf("validateNative() = %q, %v", result, err)
	}
	for _, value := range [][]byte{nil, []byte(`{"format":"wrong"}`), []byte(`not-json`)} {
		if _, err := validateNative(value); !errors.Is(err, ErrInvalidDocument) {
			t.Fatalf("validateNative(%q) error = %v", value, err)
		}
	}
}

func TestValidateNativeSizeLimit(t *testing.T) {
	value := append([]byte(`{"format":"documi-json-v1","padding":"`), make([]byte, MaxNativeBytes)...)
	value = append(value, []byte(`"}`)...)
	if _, err := validateNative(value); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("validateNative(oversized) error = %v", err)
	}
}
