package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"ulas-service/internal/config"
	"ulas-service/internal/ldap"
	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/internal/token"
	"ulas-service/models"
)

// errNoAccess mirrors the password login's 403 for users missing from user_access.
var errNoAccess = errors.New("user has no access to this system")

// entraCallbackPath is where the browser is sent after the Entra round trip.
// The frontend login page picks up ?code= (success) or ?error= (failure).
const entraCallbackPath = "/login"

// EntraHandler handles the Entra ID (OIDC) login flow:
// GET  /api/auth/entra/login     → redirect to Microsoft Entra authorize URL
// GET  /api/auth/entra/callback  → verify state, exchange code, issue app JWT,
//
//	then redirect to the SPA with a one-time code
//
// POST /api/auth/entra/exchange  → trade the one-time code for the LoginResponse
//
// The one-time code indirection keeps the JWT out of the browser URL/history.
type EntraHandler struct {
	tokenMaker     token.Maker
	cfg            *config.AppConfig
	accessRepo     repository.AccessRepository
	entra          service.EntraService
	ldapConfigured bool

	mu     sync.Mutex
	states map[string]time.Time // oauth state → expiry
	codes  map[string]pendingEntraLogin
}

type pendingEntraLogin struct {
	login   LoginResponse
	expires time.Time
}

// NewEntraHandler creates an EntraHandler. ldapClient is used only to decide
// whether the production user_access permission check applies (same rule as
// the password login). svc may be nil when Entra ID is not configured.
func NewEntraHandler(tokenMaker token.Maker, cfg *config.AppConfig, accessRepo repository.AccessRepository, ldapClient *ldap.Client, svc service.EntraService) *EntraHandler {
	return &EntraHandler{
		tokenMaker:     tokenMaker,
		cfg:            cfg,
		accessRepo:     accessRepo,
		entra:          svc,
		ldapConfigured: ldapClient != nil && cfg.LDAPServer != "",
		states:         make(map[string]time.Time),
		codes:          make(map[string]pendingEntraLogin),
	}
}

// EntraLoginRedirect handles GET /api/auth/entra/login.
func (h *EntraHandler) EntraLoginRedirect(c *gin.Context) {
	if h.entra == nil || !h.entra.Configured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "entra id login is not configured"})
		return
	}
	state, err := randomHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.mu.Lock()
	h.pruneLocked()
	h.states[state] = time.Now().Add(5 * time.Minute)
	h.mu.Unlock()

	authURL, err := h.entra.AuthURL(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

// EntraCallback handles GET /api/auth/entra/callback.
func (h *EntraHandler) EntraCallback(c *gin.Context) {
	fail := func(status int, msg string) {
		// The SPA login page renders ?error= for the user.
		c.Redirect(status, entraCallbackPath+"?error="+url.QueryEscape(msg))
	}

	if h.entra == nil || !h.entra.Configured() {
		fail(http.StatusFound, "entra id login is not configured")
		return
	}
	query := c.Request.URL.Query()
	state, code := query.Get("state"), query.Get("code")
	if errParam := query.Get("error"); errParam != "" {
		fail(http.StatusFound, "entra sign-in failed: "+query.Get("error_description"))
		return
	}
	if code == "" || !h.consumeState(state) {
		fail(http.StatusFound, "invalid or expired entra sign-in state")
		return
	}

	identity, err := h.entra.Exchange(c.Request.Context(), code)
	if err != nil {
		fail(http.StatusFound, "entra token exchange failed")
		return
	}
	login, err := h.buildLoginResponse(c, identity)
	if err != nil {
		fail(http.StatusFound, err.Error())
		return
	}

	oneTime, err := randomHex(24)
	if err != nil {
		fail(http.StatusFound, err.Error())
		return
	}
	h.mu.Lock()
	h.pruneLocked()
	h.codes[oneTime] = pendingEntraLogin{login: *login, expires: time.Now().Add(2 * time.Minute)}
	h.mu.Unlock()

	c.Redirect(http.StatusFound, entraCallbackPath+"?code="+oneTime)
}

// EntraExchangeRequest is the body for POST /api/auth/entra/exchange.
type EntraExchangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// EntraExchange handles POST /api/auth/entra/exchange.
func (h *EntraHandler) EntraExchange(c *gin.Context) {
	var req EntraExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.mu.Lock()
	pending, ok := h.codes[req.Code]
	if ok {
		delete(h.codes, req.Code) // single use
	}
	h.mu.Unlock()
	if !ok || time.Now().After(pending.expires) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired login code"})
		return
	}
	c.JSON(http.StatusOK, pending.login)
}

// buildLoginResponse resolves permissions and mints the application JWT,
// mirroring the rules of the password login (AuthHandler.Login).
func (h *EntraHandler) buildLoginResponse(c *gin.Context, identity *service.EntraIdentity) (*LoginResponse, error) {
	// The development fallback (no LDAP) implies full local access; production
	// logins always go through the user_access table.
	permission := models.AccessLevelAdmin
	if h.ldapConfigured {
		level, err := h.accessRepo.GetAccessLevel(c.Request.Context(), identity.Username)
		if err != nil {
			return nil, err
		}
		if level != models.AccessLevelRead && level != models.AccessLevelAdmin {
			return nil, errNoAccess
		}
		permission = level
	}

	duration := h.cfg.JWTAccessTokenDurationDuration()
	accessToken, _, err := h.tokenMaker.CreateToken(identity.Username, permission, duration)
	if err != nil {
		return nil, err
	}

	userInfo := map[string]string{
		"cn":    identity.Username,
		"name":  identity.Name,
		"email": identity.Email,
	}
	return &LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(duration.Seconds()),
		Username:    identity.Username,
		Permission:  permission,
		UserInfo:    userInfo,
	}, nil
}

// consumeState reports whether the state exists and has not expired, removing
// it so it cannot be replayed.
func (h *EntraHandler) consumeState(state string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	expiry, ok := h.states[state]
	if !ok || time.Now().After(expiry) {
		return false
	}
	delete(h.states, state)
	return true
}

func (h *EntraHandler) pruneLocked() {
	now := time.Now()
	for s, expiry := range h.states {
		if now.After(expiry) {
			delete(h.states, s)
		}
	}
	for code, pending := range h.codes {
		if now.After(pending.expires) {
			delete(h.codes, code)
		}
	}
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
