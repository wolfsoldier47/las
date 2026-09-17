package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Different types of error returned by the VerifyToken function.
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// Payload contains the payload data of the token.
type Payload struct {
	ID         uuid.UUID `json:"id"`
	Username   string    `json:"username"`
	Permission string    `json:"permission"` // read | admin
	jwt.RegisteredClaims
}

// NewPayload creates a new token payload with a specific username, permission, and duration.
func NewPayload(username, permission string, duration time.Duration) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	payload := &Payload{
		ID:         tokenID,
		Username:   username,
		Permission: permission,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID.String(),
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}
	return payload, nil
}

// Valid checks if the token payload is valid or not.
func (payload *Payload) Valid() error {
	if payload.Username == "" || payload.Permission == "" {
		return ErrInvalidToken
	}
	return nil
}
