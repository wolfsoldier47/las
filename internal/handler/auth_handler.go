package handler

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ulas-service/internal/config"
	"ulas-service/internal/ldap"
	"ulas-service/internal/repository"
	"ulas-service/internal/token"
	"ulas-service/models"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	tokenMaker token.Maker
	ldapClient *ldap.Client
	cfg        *config.AppConfig
	accessRepo repository.AccessRepository
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(tokenMaker token.Maker, ldapClient *ldap.Client, cfg *config.AppConfig, accessRepo repository.AccessRepository) *AuthHandler {
	return &AuthHandler{
		tokenMaker: tokenMaker,
		ldapClient: ldapClient,
		cfg:        cfg,
		accessRepo: accessRepo,
	}
}

// LoginRequest is the body for POST /api/login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned after a successful login.
type LoginResponse struct {
	AccessToken string            `json:"access_token"`
	TokenType   string            `json:"token_type"`
	ExpiresIn   int64             `json:"expires_in"`
	Username    string            `json:"username"`
	Permission  string            `json:"permission"`
	UserInfo    map[string]string `json:"user_info"`
}

// Login handles POST /api/login.
// If LDAP is configured, credentials are verified against LDAP.
// Otherwise, a development fallback allows any non-empty username/password.
// After authentication the user must have an entry in the user_access table
// with level "read" or "admin"; users without a row get no access at all.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authenticated, userInfo, err := h.authenticate(c, req.Username, req.Password)
	if err != nil || !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// The development fallback (no LDAP) implies full local access; production
	// logins always go through the user_access table below.
	permission := models.AccessLevelAdmin
	if h.ldapClient != nil && h.cfg.LDAPServer != "" {
		permission, err = h.accessRepo.GetAccessLevel(c.Request.Context(), req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if permission != models.AccessLevelRead && permission != models.AccessLevelAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "user has no access to this system"})
			return
		}
	}

	duration := h.cfg.JWTAccessTokenDurationDuration()
	accessToken, _, err := h.tokenMaker.CreateToken(req.Username, permission, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(duration.Seconds()),
		Username:    req.Username,
		Permission:  permission,
		UserInfo:    userInfo,
	})
}

func (h *AuthHandler) authenticate(c *gin.Context, username, password string) (bool, map[string]string, error) {
	if h.ldapClient != nil && h.cfg.LDAPServer != "" {
		return h.ldapClient.Authenticate(username, password)
	}

	// Development fallback when LDAP is not configured.
	// In production, LDAP must be configured.
	if h.cfg.AppStage == "DEV" && username == "admin" && password == "admin" {
		return true, randomUserInfo(), nil
	}
	return false, nil, nil
}

// returns sample user attributes for the (OPENSHIT_STAGE)DEV fallback login.
func randomUserInfo() map[string]string {
	profiles := []struct {
		name string
		role string
	}{
		{"Noemi Lemme", "Security Engineer"},
		{"Benedikt Eckert", "Compliance Officer"},
		{"Simon Jennsovsky", "System Engineer"},
		{"Nick Khalid", "Software Engineer"},
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	profile := profiles[r.Intn(len(profiles))]
	parts := strings.Split(profile.name, " ")
	email := strings.ToLower(parts[0]+"."+parts[1]) + "@ulas.local"

	return map[string]string{
		"cn":    "admin",
		"name":  profile.name,
		"role":  profile.role,
		"email": email,
	}
}

// GetCurrentUser handles GET /api/me.
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	payload, exists := c.Get("authorization_payload")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":   authPayload.Username,
		"permission": authPayload.Permission,
	})
}

// AdminMiddleware rejects authenticated users without admin permission.
// It must be applied after AuthMiddleware.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("permission") != models.AccessLevelAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

// AuthMiddleware creates a gin middleware for JWT authorization.
func AuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is not provided"})
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		if fields[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unsupported authorization type"})
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		// Tokens issued before the permission claim existed carry no access
		// level — treat them as expired credentials.
		if payload.Permission == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token carries no permission"})
			return
		}

		c.Set("authorization_payload", payload)
		c.Set("username", payload.Username)
		c.Set("permission", payload.Permission)
		c.Next()
	}
}

// AllowOriginMiddleware wraps the existing CORS middleware to ensure login
// requests work from the frontend.
func AllowOriginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
