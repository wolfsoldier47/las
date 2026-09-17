package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ulas-service/internal/config"
	"ulas-service/internal/token"
	"ulas-service/models"
)

// fakeAccessRepo stubs repository.AccessRepository.
type fakeAccessRepo struct {
	level string
	err   error
}

func (f *fakeAccessRepo) GetAccessLevel(_ context.Context, _ string) (string, error) {
	return f.level, f.err
}

func devTestConfig() *config.AppConfig {
	cfg := *config.Get()
	cfg.AppStage = "DEV"
	return &cfg
}

func loginRequest(t *testing.T, h *AuthHandler, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	c.Writer.WriteHeaderNow()
	return recorder
}

// The DEV fallback (no LDAP configured) grants admin so local development
// keeps working; production logins go through the user_access table.
func TestLogin_DevFallbackGrantsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := token.NewJWTMaker("01234567890123456789012345678901")
	if err != nil {
		t.Fatalf("NewJWTMaker: %v", err)
	}
	h := NewAuthHandler(maker, nil, devTestConfig(), &fakeAccessRepo{level: ""})

	recorder := loginRequest(t, h, "admin", "admin")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from dev fallback login, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp LoginResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Permission != models.AccessLevelAdmin {
		t.Errorf("expected admin permission from dev fallback, got %q", resp.Permission)
	}
}

func TestAdminMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	run := func(permission string) int {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		if permission != "" {
			c.Set("permission", permission)
		}
		c.Request = httptest.NewRequest(http.MethodPost, "/api/scans", nil)

		AdminMiddleware()(c)
		if !c.IsAborted() {
			c.Status(http.StatusNoContent)
			c.Writer.WriteHeaderNow()
		}
		return recorder.Code
	}

	if code := run("admin"); code != http.StatusNoContent {
		t.Errorf("admin: expected 204, got %d", code)
	}
	if code := run("read"); code != http.StatusForbidden {
		t.Errorf("read: expected 403, got %d", code)
	}
	if code := run(""); code != http.StatusForbidden {
		t.Errorf("no permission: expected 403, got %d", code)
	}
}
