package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Subject struct {
	ID              string   `json:"id"`
	Issuer          string   `json:"issuer"`
	ExternalSubject string   `json:"externalSubject"`
	Email           string   `json:"email,omitempty"`
	DisplayName     string   `json:"displayName"`
	EmailVerified   bool     `json:"emailVerified"`
	Groups          []string `json:"-"`
}
type subjectKey struct{}

func WithSubject(ctx context.Context, subject Subject) context.Context {
	return context.WithValue(ctx, subjectKey{}, subject)
}
func FromContext(ctx context.Context) (Subject, bool) {
	subject, ok := ctx.Value(subjectKey{}).(Subject)
	return subject, ok && subject.ID != ""
}

type Authenticator interface {
	Authenticate(context.Context, string) (Subject, error)
}

type DevelopmentAuthenticator struct{ Subject Subject }

func (authenticator DevelopmentAuthenticator) Authenticate(context.Context, string) (Subject, error) {
	if authenticator.Subject.ID == "" {
		return Subject{}, errors.New("development subject missing")
	}
	return authenticator.Subject, nil
}

type OIDCAuthenticator struct {
	verifier   *oidc.IDTokenVerifier
	groupClaim string
}

func NewOIDCAuthenticator(ctx context.Context, issuer, discoveryURL, audience string, groupClaim ...string) (*OIDCAuthenticator, error) {
	if discoveryURL != issuer {
		ctx = oidc.InsecureIssuerURLContext(ctx, issuer)
	}
	provider, err := oidc.NewProvider(ctx, discoveryURL)
	if err != nil {
		return nil, err
	}
	claim := "groups"
	if len(groupClaim) > 0 && strings.TrimSpace(groupClaim[0]) != "" {
		claim = groupClaim[0]
	}
	return &OIDCAuthenticator{verifier: provider.Verifier(&oidc.Config{ClientID: audience}), groupClaim: claim}, nil
}
func (authenticator *OIDCAuthenticator) Authenticate(ctx context.Context, raw string) (Subject, error) {
	token, err := authenticator.verifier.Verify(ctx, raw)
	if err != nil {
		return Subject{}, err
	}
	var claims struct {
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := token.Claims(&claims); err != nil {
		return Subject{}, err
	}
	if token.Subject == "" {
		return Subject{}, errors.New("OIDC subject missing")
	}
	var claimValues map[string]json.RawMessage
	_ = token.Claims(&claimValues)
	var groups []string
	if value := claimValues[authenticator.groupClaim]; value != nil {
		_ = json.Unmarshal(value, &groups)
	}
	return Subject{ID: token.Subject, Issuer: token.Issuer, ExternalSubject: token.Subject, Email: claims.Email, DisplayName: claims.Name, EmailVerified: claims.EmailVerified, Groups: groups}, nil
}

func Middleware(authenticator Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw := ""
		header := request.Header.Get("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			raw = strings.TrimPrefix(header, "Bearer ")
		}
		subject, err := authenticator.Authenticate(request.Context(), raw)
		if err != nil {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]string{"error": "authentication_required"})
			return
		}
		next.ServeHTTP(writer, request.WithContext(WithSubject(request.Context(), subject)))
	})
}
