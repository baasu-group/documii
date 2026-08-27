package onlyoffice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidJWT = errors.New("invalid ONLYOFFICE JWT")

// Signer signs server-generated ONLYOFFICE editor configurations and callback
// payloads with HS256. It never returns or serializes the signing secret.
type Signer struct {
	secret []byte
}

func NewSigner(secret []byte) (*Signer, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("ONLYOFFICE JWT secret must contain at least 32 bytes")
	}
	return &Signer{secret: append([]byte(nil), secret...)}, nil
}

func (signer *Signer) Sign(payload any) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode ONLYOFFICE payload: %w", err)
	}
	unsigned := encode(header) + "." + encode(body)
	return unsigned + "." + signer.signature(unsigned), nil
}

func (signer *Signer) Verify(token string, destination any) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ErrInvalidJWT
	}
	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(signer.signature(unsigned)), []byte(parts[2])) {
		return ErrInvalidJWT
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, destination) != nil {
		return ErrInvalidJWT
	}
	return nil
}

func (signer *Signer) signature(payload string) string {
	mac := hmac.New(sha256.New, signer.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}
