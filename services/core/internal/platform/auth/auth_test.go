package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevelopmentMiddlewareInjectsSubject(t *testing.T) {
	handler := Middleware(DevelopmentAuthenticator{Subject: Subject{ID: "dev-user"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, ok := FromContext(r.Context())
		if !ok || subject.ID != "dev-user" {
			t.Fatal("subject absent")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d", response.Code)
	}
}
func TestDevelopmentAuthenticatorFailsClosedWithoutSubject(t *testing.T) {
	_, err := DevelopmentAuthenticator{}.Authenticate(context.Background(), "")
	if err == nil {
		t.Fatal("empty subject accepted")
	}
}
