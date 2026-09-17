package token

import (
	"testing"
	"time"
)

func TestJWTMakerPermissionRoundTrip(t *testing.T) {
	maker, err := NewJWTMaker("01234567890123456789012345678901")
	if err != nil {
		t.Fatalf("NewJWTMaker: %v", err)
	}

	for _, permission := range []string{"read", "admin"} {
		tokenStr, payload, err := maker.CreateToken("comsiid123", permission, time.Minute)
		if err != nil {
			t.Fatalf("CreateToken: %v", err)
		}
		if tokenStr == "" {
			t.Fatal("expected non-empty token")
		}
		if payload.Permission != permission {
			t.Errorf("payload permission = %q, want %q", payload.Permission, permission)
		}

		verified, err := maker.VerifyToken(tokenStr)
		if err != nil {
			t.Fatalf("VerifyToken: %v", err)
		}
		if verified.Username != "comsiid123" {
			t.Errorf("username = %q", verified.Username)
		}
		if verified.Permission != permission {
			t.Errorf("verified permission = %q, want %q", verified.Permission, permission)
		}
	}
}

// Enforcement of the permission claim lives in AuthMiddleware, not the token
// parser: jwt/v5 does not invoke Payload.Valid, so a token minted without a
// permission still parses — but carries an empty claim the middleware rejects.
func TestJWTMakerTokenWithoutPermissionParsesEmpty(t *testing.T) {
	maker, err := NewJWTMaker("01234567890123456789012345678901")
	if err != nil {
		t.Fatalf("NewJWTMaker: %v", err)
	}

	tokenStr, _, err := maker.CreateToken("comsiid123", "", time.Minute)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	verified, err := maker.VerifyToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if verified.Permission != "" {
		t.Errorf("expected empty permission, got %q", verified.Permission)
	}
}
