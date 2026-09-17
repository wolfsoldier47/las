package token

import "time"

// Maker is an interface for managing tokens.
type Maker interface {
	// CreateToken creates a new token for a specific username, permission, and duration.
	CreateToken(username, permission string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not.
	VerifyToken(token string) (*Payload, error)
}
