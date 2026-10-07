package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ulas-service/internal/service"
	"ulas-service/internal/token"
)

// fakeEntraService stubs service.EntraService.
type fakeEntraService struct {
	configured  bool
	authURL     string
	identity    *service.EntraIdentity
	exchangeErr error
}

func (f *fakeEntraService) Configured() bool { return f.configured }
func (f *fakeEntraService) AuthURL(_ context.Context, state string) (string, error) {
	return f.authURL + "?state=" + state, nil
}
func (f *fakeEntraService) Exchange(_ context.Context, code string) (*service.EntraIdentity, error) {
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return f.identity, nil
}

func newTestEntraHandler(svc service.EntraService) *EntraHandler {
	maker, _ := token.NewJWTMaker("01234567890123456789012345678901")
	return NewEntraHandler(maker, devTestConfig(), &fakeAccessRepo{level: "admin"}, nil, svc)
}

func TestEntraLoginRedirect_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestEntraHandler(&fakeEntraService{configured: false})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/entra/login", nil)
	h.EntraLoginRedirect(c)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when not configured, got %d", recorder.Code)
	}
}

func TestEntraLoginRedirect_Configured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestEntraHandler(&fakeEntraService{configured: true, authURL: "https://login.microsoftonline.com/authorize"})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/entra/login", nil)
	h.EntraLoginRedirect(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", recorder.Code)
	}
	loc, _ := url.Parse(recorder.Header().Get("Location"))
	if !strings.HasPrefix(loc.String(), "https://login.microsoftonline.com/authorize?state=") {
		t.Fatalf("unexpected redirect target: %s", loc)
	}
	if loc.Query().Get("state") == "" {
		t.Fatal("redirect did not carry an oauth state")
	}
}

func entraCallback(h *EntraHandler, rawQuery string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	target := "/api/auth/entra/callback"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	h.EntraCallback(c)
	return recorder
}

func TestEntraCallback_InvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestEntraHandler(&fakeEntraService{configured: true, identity: &service.EntraIdentity{Username: "jdoe1"}})

	rec := entraCallback(h, "code=abc&state=forged")
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	if !strings.Contains(loc.Query().Get("error"), "state") {
		t.Fatalf("expected state error, got: %s", loc)
	}
}

func TestEntraCallback_SuccessIssuesSingleUseCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestEntraHandler(&fakeEntraService{
		configured: true,
		identity:   &service.EntraIdentity{Username: "jdoe1", Name: "Jane Doe", Email: "jane@bank.com"},
	})

	// Obtain a valid state from the login redirect.
	loginRec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(loginRec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/entra/login", nil)
	h.EntraLoginRedirect(c)
	loc, _ := url.Parse(loginRec.Header().Get("Location"))
	state := loc.Query().Get("state")

	rec := entraCallback(h, "code=good&state="+state)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d: %s", rec.Code, rec.Body.String())
	}
	cbLoc, _ := url.Parse(rec.Header().Get("Location"))
	if cbLoc.Path != "/login" {
		t.Fatalf("expected redirect to /login, got %s", cbLoc.Path)
	}
	code := cbLoc.Query().Get("code")
	if code == "" {
		t.Fatal("callback did not issue a one-time code")
	}

	// First exchange succeeds.
	first := entraExchange(t, h, code)
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200 from exchange, got %d: %s", first.Code, first.Body.String())
	}
	var resp LoginResponse
	if err := json.Unmarshal(first.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal exchange response: %v", err)
	}
	if resp.Username != "jdoe1" || resp.Permission != "admin" {
		t.Fatalf("unexpected login response: %+v", resp)
	}
	if resp.AccessToken == "" {
		t.Fatal("exchange returned no access token")
	}

	// Second exchange with the same code must fail (single use).
	second := entraExchange(t, h, code)
	if second.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 reusing one-time code, got %d", second.Code)
	}
}

func TestEntraCallback_ExchangeErrorRedirectsWithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestEntraHandler(&fakeEntraService{configured: true, exchangeErr: errors.New("bad code")})

	loginRec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(loginRec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/entra/login", nil)
	h.EntraLoginRedirect(c)
	loc, _ := url.Parse(loginRec.Header().Get("Location"))
	state := loc.Query().Get("state")

	rec := entraCallback(h, "code=bad&state="+state)
	cbLoc, _ := url.Parse(rec.Header().Get("Location"))
	if cbLoc.Query().Get("error") == "" {
		t.Fatalf("expected error redirect, got %s", cbLoc)
	}
}

func entraExchange(t *testing.T, h *EntraHandler, code string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"code": code})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/entra/exchange", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.EntraExchange(c)
	return recorder
}
