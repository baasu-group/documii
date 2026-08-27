package collabora

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid editor access token")

type Claims struct {
	FileID    string `json:"file_id"`
	ActorID   string `json:"actor_id"`
	VersionID string `json:"version_id,omitempty"`
	CanRead   bool   `json:"can_read"`
	CanEdit   bool   `json:"can_edit"`
	Expires   int64  `json:"expires"`
	Nonce     string `json:"nonce"`
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenService(secret []byte, ttl time.Duration) (*TokenService, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("editor token secret must contain at least 32 bytes")
	}
	if ttl <= 0 || ttl > 10*time.Minute {
		return nil, fmt.Errorf("editor token TTL must be between zero and ten minutes")
	}
	return &TokenService{secret: append([]byte(nil), secret...), ttl: ttl, now: time.Now}, nil
}

func (service *TokenService) Issue(fileID, actorID string, canRead, canEdit bool) (string, time.Time, error) {
	return service.IssueForVersion(fileID, "", actorID, canRead, canEdit)
}

func (service *TokenService) IssueForVersion(fileID, versionID, actorID string, canRead, canEdit bool) (string, time.Time, error) {
	if fileID == "" || actorID == "" || !canRead {
		return "", time.Time{}, fmt.Errorf("file ID, actor ID, and read access are required")
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("create token nonce: %w", err)
	}
	expiresAt := service.now().Add(service.ttl)
	claims := Claims{
		FileID:    fileID,
		ActorID:   actorID,
		VersionID: versionID,
		CanRead:   canRead,
		CanEdit:   canEdit,
		Expires:   expiresAt.Unix(),
		Nonce:     base64.RawURLEncoding.EncodeToString(nonceBytes),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encode editor claims: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + service.signature(encoded), expiresAt, nil
}

func (service *TokenService) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(service.signature(parts[0])), []byte(parts[1])) {
		return Claims{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.FileID == "" || claims.ActorID == "" || !claims.CanRead || service.now().Unix() >= claims.Expires {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (service *TokenService) signature(payload string) string {
	mac := hmac.New(sha256.New, service.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
