package collabora

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeGateway struct {
	access    Access
	file      File
	content   []byte
	saved     []byte
	saveCalls int
	saveErr   error
}

func (gateway *fakeGateway) SessionAccess(context.Context, string, string, string) (Access, error) {
	return gateway.access, nil
}

func (gateway *fakeGateway) Access(context.Context, string, string) (Access, error) {
	return gateway.access, nil
}

func (gateway *fakeGateway) Info(context.Context, string, string) (File, error) {
	return gateway.file, nil
}

func (gateway *fakeGateway) OpenContent(context.Context, string, string) (io.ReadCloser, File, error) {
	return io.NopCloser(bytes.NewReader(gateway.content)), gateway.file, nil
}

func (gateway *fakeGateway) SaveNewVersion(_ context.Context, request SaveRequest) (File, error) {
	if gateway.saveErr != nil {
		return File{}, gateway.saveErr
	}
	content, err := io.ReadAll(request.Content)
	if err != nil {
		return File{}, err
	}
	gateway.saved = content
	gateway.saveCalls++
	gateway.file.VersionID = "ver_2"
	gateway.file.Size = int64(len(content))
	return gateway.file, nil
}

func TestWOPILockConflictReturnsCurrentLock(t *testing.T) {
	handler, tokens, gateway := testHandler(t, Access{CanRead: true, CanEdit: true})
	gateway.saveErr = ErrConflict
	token, _, _ := tokens.IssueForVersion("doc_1", "ver_1", "user_1", true, true)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/wopi/files/doc_1/contents?access_token="+token, strings.NewReader("changed"))
	request.Header.Set("X-WOPI-Lock", "wrong")
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.PutFile(response, request)
	if response.Code != http.StatusConflict || response.Header().Get("X-WOPI-Lock") != "lock-1" {
		t.Fatalf("status=%d lock=%q", response.Code, response.Header().Get("X-WOPI-Lock"))
	}
}

func (gateway *fakeGateway) ManageLock(_ context.Context, request LockRequest) (LockResult, error) {
	if request.Operation == LockGet {
		return LockResult{CurrentLock: "lock-1"}, nil
	}
	return LockResult{}, nil
}

func TestWOPILockOperationRequiresEditAndReturnsCurrentLock(t *testing.T) {
	handler, tokens, _ := testHandler(t, Access{CanRead: true, CanEdit: true})
	token, _, _ := tokens.Issue("doc_1", "user_1", true, true)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/wopi/files/doc_1?access_token="+token, nil)
	request.SetPathValue("id", "doc_1")
	request.Header.Set("X-WOPI-Override", "GET_LOCK")
	response := httptest.NewRecorder()
	handler.FileOperation(response, request)
	if response.Code != http.StatusOK || response.Header().Get("X-WOPI-Lock") != "lock-1" {
		t.Fatalf("status=%d lock=%q", response.Code, response.Header().Get("X-WOPI-Lock"))
	}
}

func testHandler(t *testing.T, access Access) (*Handler, *TokenService, *fakeGateway) {
	t.Helper()
	tokens, err := NewTokenService([]byte("0123456789abcdef0123456789abcdef"), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	gateway := &fakeGateway{
		access:  access,
		file:    File{ID: "doc_1", Name: "report.docx", VersionID: "ver_1", Size: 5, ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		content: []byte("hello"),
	}
	handler, err := NewHandler(tokens, gateway, 1024)
	if err != nil {
		t.Fatal(err)
	}
	return handler, tokens, gateway
}

func TestWOPIRejectsMissingToken(t *testing.T) {
	handler, _, _ := testHandler(t, Access{CanRead: true, CanEdit: true})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/wopi/files/doc_1", nil)
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.CheckFileInfo(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("CheckFileInfo() = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestWOPIReadOnlyTokenCanReadButCannotSave(t *testing.T) {
	handler, tokens, gateway := testHandler(t, Access{CanRead: true, CanEdit: true})
	token, _, err := tokens.Issue("doc_1", "user_1", true, false)
	if err != nil {
		t.Fatal(err)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/wopi/files/doc_1/contents?access_token="+token, nil)
	getRequest.SetPathValue("id", "doc_1")
	getResponse := httptest.NewRecorder()
	handler.GetFile(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || getResponse.Body.String() != "hello" {
		t.Fatalf("GetFile() = %d %q", getResponse.Code, getResponse.Body.String())
	}

	putRequest := httptest.NewRequest(http.MethodPost, "/api/v1/wopi/files/doc_1/contents?access_token="+token, strings.NewReader("changed"))
	putRequest.SetPathValue("id", "doc_1")
	putResponse := httptest.NewRecorder()
	handler.PutFile(putResponse, putRequest)
	if putResponse.Code != http.StatusNotFound || gateway.saveCalls != 0 {
		t.Fatalf("PutFile() = %d, save calls = %d", putResponse.Code, gateway.saveCalls)
	}
}

func TestWOPISaveCreatesNewVersionThroughDocumentsGateway(t *testing.T) {
	handler, tokens, gateway := testHandler(t, Access{CanRead: true, CanEdit: true})
	token, _, err := tokens.Issue("doc_1", "user_1", true, true)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/wopi/files/doc_1/contents?access_token="+token, strings.NewReader("changed"))
	request.Header.Set("X-Documi-Base-Version", "ver_1")
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.PutFile(response, request)

	if response.Code != http.StatusOK || gateway.saveCalls != 1 || string(gateway.saved) != "changed" {
		t.Fatalf("PutFile() = %d, calls = %d, saved = %q", response.Code, gateway.saveCalls, gateway.saved)
	}
}

func TestWOPIRechecksCurrentPermission(t *testing.T) {
	handler, tokens, gateway := testHandler(t, Access{CanRead: true, CanEdit: true})
	token, _, _ := tokens.Issue("doc_1", "user_1", true, true)
	gateway.access = Access{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/wopi/files/doc_1?access_token="+token, nil)
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.CheckFileInfo(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("CheckFileInfo() = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestWOPIRejectsOversizedSaveBeforeGateway(t *testing.T) {
	handler, tokens, gateway := testHandler(t, Access{CanRead: true, CanEdit: true})
	token, _, _ := tokens.IssueForVersion("doc_1", "ver_1", "user_1", true, true)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/wopi/files/doc_1/contents?access_token="+token, strings.NewReader(strings.Repeat("x", 1025)))
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.PutFile(response, request)

	if response.Code != http.StatusRequestEntityTooLarge || gateway.saveCalls != 0 {
		t.Fatalf("PutFile() = %d, save calls = %d", response.Code, gateway.saveCalls)
	}
}

func TestWOPIRejectsUnsafeFilename(t *testing.T) {
	handler, tokens, gateway := testHandler(t, Access{CanRead: true})
	gateway.file.Name = "../../secret.docx"
	token, _, _ := tokens.Issue("doc_1", "user_1", true, false)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/wopi/files/doc_1?access_token="+token, nil)
	request.SetPathValue("id", "doc_1")
	response := httptest.NewRecorder()
	handler.CheckFileInfo(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("CheckFileInfo() = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
