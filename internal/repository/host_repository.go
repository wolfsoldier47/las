package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrHostNotFound is returned when a host cannot be located.
var ErrHostNotFound = errors.New("host not found")

// HostFilters contains optional filters for listing hosts.
type HostFilters struct {
	Search *string
}

// HostRepository defines storage operations for hosts.
type HostRepository interface {
	Create(ctx context.Context, host *models.Host) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Host, error)
	GetByHostname(ctx context.Context, hostname string) (*models.Host, error)
	List(ctx context.Context) ([]models.Host, error)
	ListPaginated(ctx context.Context, filters HostFilters, page, limit int) ([]models.Host, int, error)
	Update(ctx context.Context, host *models.Host) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PgHostRepository is a PostgreSQL implementation of HostRepository.
type PgHostRepository struct {
	db *sql.DB
}

// NewPgHostRepository creates a new PostgreSQL host repository.
func NewPgHostRepository(db *sql.DB) *PgHostRepository {
	return &PgHostRepository{db: db}
}

// Create inserts a new host.
func (r *PgHostRepository) Create(ctx context.Context, host *models.Host) error {
	query := `
		INSERT INTO hosts (id, hostname, os_type, os_name, os_version, environment, datacenter, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		host.ID,
		host.Hostname,
		host.OSType,
		host.OSName,
		host.OSVersion,
		host.Environment,
		host.Datacenter,
		host.Description,
		host.CreatedAt,
		host.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create host: %w", err)
	}
	return nil
}

// GetByID returns a host by its UUID.
func (r *PgHostRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Host, error) {
	query := `
		SELECT id, hostname, os_type, os_name, os_version, environment, datacenter, description, created_at, updated_at
		FROM hosts
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var host models.Host
	if err := row.Scan(
		&host.ID,
		&host.Hostname,
		&host.OSType,
		&host.OSName,
		&host.OSVersion,
		&host.Environment,
		&host.Datacenter,
		&host.Description,
		&host.CreatedAt,
		&host.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrHostNotFound
		}
		return nil, fmt.Errorf("get host by id: %w", err)
	}
	return &host, nil
}

// GetByHostname returns a host by its hostname.
func (r *PgHostRepository) GetByHostname(ctx context.Context, hostname string) (*models.Host, error) {
	query := `
		SELECT id, hostname, os_type, os_name, os_version, environment, datacenter, description, created_at, updated_at
		FROM hosts
		WHERE hostname = $1
	`
	row := r.db.QueryRowContext(ctx, query, hostname)

	var host models.Host
	if err := row.Scan(
		&host.ID,
		&host.Hostname,
		&host.OSType,
		&host.OSName,
		&host.OSVersion,
		&host.Environment,
		&host.Datacenter,
		&host.Description,
		&host.CreatedAt,
		&host.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrHostNotFound
		}
		return nil, fmt.Errorf("get host by hostname: %w", err)
	}
	return &host, nil
}

// List returns all registered hosts.
func (r *PgHostRepository) List(ctx context.Context) ([]models.Host, error) {
	query := `
		SELECT id, hostname, os_type, os_name, os_version, environment, datacenter, description, created_at, updated_at
		FROM hosts
		ORDER BY hostname
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list hosts: %w", err)
	}
	defer rows.Close()

	var hosts []models.Host
	for rows.Next() {
		var host models.Host
		if err := rows.Scan(
			&host.ID,
			&host.Hostname,
			&host.OSType,
			&host.OSName,
			&host.OSVersion,
			&host.Environment,
			&host.Datacenter,
			&host.Description,
			&host.CreatedAt,
			&host.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan host: %w", err)
		}
		hosts = append(hosts, host)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hosts: %w", err)
	}
	return hosts, nil
}

// ListPaginated returns a page of hosts matching optional filters.
func (r *PgHostRepository) ListPaginated(ctx context.Context, filters HostFilters, page, limit int) ([]models.Host, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	where, args := r.buildHostWhere(filters)

	var total int
	countQuery := "SELECT COUNT(*) FROM hosts" + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count hosts: %w", err)
	}

	query := `
		SELECT id, hostname, os_type, os_name, os_version, environment, datacenter, description, created_at, updated_at
		FROM hosts
	` + where + `
		ORDER BY hostname
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list hosts paginated: %w", err)
	}
	defer rows.Close()

	var hosts []models.Host
	for rows.Next() {
		var host models.Host
		if err := rows.Scan(
			&host.ID,
			&host.Hostname,
			&host.OSType,
			&host.OSName,
			&host.OSVersion,
			&host.Environment,
			&host.Datacenter,
			&host.Description,
			&host.CreatedAt,
			&host.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan host: %w", err)
		}
		hosts = append(hosts, host)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate hosts: %w", err)
	}
	return hosts, total, nil
}

func (r *PgHostRepository) buildHostWhere(filters HostFilters) (string, []interface{}) {
	var args []interface{}
	var argCount int
	where := " WHERE 1=1"

	if filters.Search != nil && *filters.Search != "" {
		argCount++
		where += fmt.Sprintf(
			" AND (hostname ILIKE $%[1]d OR os_type ILIKE $%[1]d OR os_name ILIKE $%[1]d OR environment ILIKE $%[1]d OR datacenter ILIKE $%[1]d)",
			argCount,
		)
		args = append(args, "%"+*filters.Search+"%")
	}

	return where, args
}

// Update modifies an existing host.
func (r *PgHostRepository) Update(ctx context.Context, host *models.Host) error {
	query := `
		UPDATE hosts
		SET hostname = $2,
		    os_type = $3,
		    os_name = $4,
		    os_version = $5,
		    environment = $6,
		    datacenter = $7,
		    description = $8,
		    updated_at = $9
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		host.ID,
		host.Hostname,
		host.OSType,
		host.OSName,
		host.OSVersion,
		host.Environment,
		host.Datacenter,
		host.Description,
		host.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update host: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrHostNotFound
	}
	return nil
}

// Delete removes a host by its UUID.
func (r *PgHostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM hosts WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete host: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrHostNotFound
	}
	return nil
}
