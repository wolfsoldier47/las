package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AccessRepository defines storage operations for the user access table.
type AccessRepository interface {
	// GetAccessLevel returns the access level ("read" or "admin") for a
	// username, or an empty string if the user has no access row.
	GetAccessLevel(ctx context.Context, username string) (string, error)
}

// PgAccessRepository is a PostgreSQL implementation of AccessRepository.
type PgAccessRepository struct {
	db *sql.DB
}

// NewPgAccessRepository creates a new PostgreSQL access repository.
func NewPgAccessRepository(db *sql.DB) *PgAccessRepository {
	return &PgAccessRepository{db: db}
}

// GetAccessLevel returns the access level for a username, comparing
// case-insensitively (comsiid casing varies by source). Empty string means
// the user is not in the table and has no access.
func (r *PgAccessRepository) GetAccessLevel(ctx context.Context, username string) (string, error) {
	query := `
		SELECT access_level
		FROM user_access
		WHERE LOWER(username) = LOWER($1)
	`
	var level string
	if err := r.db.QueryRowContext(ctx, query, username).Scan(&level); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get access level: %w", err)
	}
	return level, nil
}
