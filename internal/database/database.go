package database

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ulas-service/internal/config"
	"ulas-service/models"
)

var (
	db      *gorm.DB
	sqlDB   *sql.DB
	initErr error
)

// Initialize opens the PostgreSQL connection with GORM, configures the pool, and runs auto-migrations.
func Initialize(cfg *config.AppConfig) error {
	connection := cfg.DatabaseDSN()
	var err error
	db, err = gorm.Open(postgres.Open(connection), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		initErr = fmt.Errorf("failed to connect to database: %w", err)
		return initErr
	}

	sqlDB, err = db.DB()
	if err != nil {
		initErr = fmt.Errorf("failed to get underlying sql.DB: %w", err)
		return initErr
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnsInt())
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnsInt())
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := migrate(); err != nil {
		initErr = fmt.Errorf("failed to run migrations: %w", err)
		return initErr
	}

	initErr = nil
	return nil
}

// DB returns the initialized GORM database instance.
func DB() *gorm.DB {
	return db
}

// SQLDB returns the underlying *sql.DB connection pool.
func SQLDB() *sql.DB {
	return sqlDB
}

// InitError returns any error from initialization.
func InitError() error {
	return initErr
}

// migrate runs GORM auto-migration for all domain models, then applies
// idempotent schema changes that GORM auto-migration may not handle.
func migrate() error {
	if err := db.AutoMigrate(
		&models.Host{},
		&models.MasterBaseline{},
		&models.MasterBaselineVersion{},
		&models.AllowedDeviation{},
		&models.ScanJob{},
		&models.ScanResult{},
		&models.HostFileSnapshot{},
		&models.HostFileChange{},
		&models.Incident{},
		&models.ServiceNowTicket{},
		&models.ScanSchedule{},
		&models.ScanScheduleRun{},
	); err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}
	return runManualMigrations()
}

// runManualMigrations applies idempotent schema changes that GORM auto-migration may not handle.
func runManualMigrations() error {
	baselineMigration := `
		ALTER TABLE master_baselines
			ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1,
			ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'master_baselines' AND column_name = 'host_id'
			) THEN
				DELETE FROM master_baselines WHERE host_id IS NOT NULL;
				ALTER TABLE master_baselines DROP COLUMN IF EXISTS host_id;
			END IF;
		END $$;

		ALTER TABLE master_baselines
			DROP COLUMN IF EXISTS is_global;

		ALTER TABLE master_baselines
			DROP CONSTRAINT IF EXISTS master_baselines_host_id_os_type_file_type_entry_key_key;

		ALTER TABLE master_baselines
			DROP CONSTRAINT IF EXISTS master_baselines_host_id_os_type_file_type_entry_key_version_key;

		ALTER TABLE master_baselines
			DROP CONSTRAINT IF EXISTS master_baselines_os_type_file_type_entry_key_version_key;

		ALTER TABLE master_baselines
			ADD CONSTRAINT master_baselines_os_type_file_type_entry_key_version_key
			UNIQUE (os_type, file_type, entry_key, version);

		DROP INDEX IF EXISTS idx_master_baselines_active_scope;
	`
	if err := db.Exec(baselineMigration).Error; err != nil {
		return fmt.Errorf("baseline versioning migration failed: %w", err)
	}

	scanJobMigration := `
		ALTER TABLE scan_jobs
			ADD COLUMN IF NOT EXISTS baseline_snapshot JSONB;
	`
	if err := db.Exec(scanJobMigration).Error; err != nil {
		return fmt.Errorf("scan job baseline snapshot migration failed: %w", err)
	}

	deviationHostnameMigration := `
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'allowed_deviations' AND column_name = 'host_id'
			) THEN
				ALTER TABLE allowed_deviations ADD COLUMN IF NOT EXISTS hostname TEXT;

				UPDATE allowed_deviations d
				SET hostname = h.hostname
				FROM hosts h
				WHERE d.host_id = h.id AND d.hostname IS NULL;

				UPDATE allowed_deviations SET hostname = '' WHERE hostname IS NULL;

				ALTER TABLE allowed_deviations ALTER COLUMN hostname SET NOT NULL;
				ALTER TABLE allowed_deviations DROP COLUMN IF EXISTS host_id;
			END IF;
		END $$;
	`
	if err := db.Exec(deviationHostnameMigration).Error; err != nil {
		return fmt.Errorf("deviation hostname migration failed: %w", err)
	}

	deviationUniqueMigration := `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'allowed_deviations_hostname_file_type_entry_key_key'
			) THEN
				ALTER TABLE allowed_deviations
					ADD CONSTRAINT allowed_deviations_hostname_file_type_entry_key_key
					UNIQUE (hostname, file_type, entry_key);
			END IF;
		END $$;
	`
	if err := db.Exec(deviationUniqueMigration).Error; err != nil {
		return fmt.Errorf("deviation unique constraint migration failed: %w", err)
	}

	hostOSVersionMigration := `
		ALTER TABLE hosts
			ADD COLUMN IF NOT EXISTS os_version TEXT;
	`
	if err := db.Exec(hostOSVersionMigration).Error; err != nil {
		return fmt.Errorf("host os_version migration failed: %w", err)
	}

	scanJobFailedHostNamesMigration := `
		ALTER TABLE scan_jobs
			ADD COLUMN IF NOT EXISTS failed_host_names JSONB;
	`
	if err := db.Exec(scanJobFailedHostNamesMigration).Error; err != nil {
		return fmt.Errorf("scan job failed_host_names migration failed: %w", err)
	}

	snapshotIndexMigration := `
		CREATE INDEX IF NOT EXISTS idx_host_file_snapshots_host_type_time
			ON host_file_snapshots(host_id, file_type, snapshot_at DESC);
	`
	if err := db.Exec(snapshotIndexMigration).Error; err != nil {
		return fmt.Errorf("snapshot host/type/time index migration failed: %w", err)
	}

	scanResultBaselineMigration := `
		ALTER TABLE scan_results
			ADD COLUMN IF NOT EXISTS baseline_version_at_scan INT,
			ADD COLUMN IF NOT EXISTS no_baseline BOOLEAN NOT NULL DEFAULT false;
	`
	if err := db.Exec(scanResultBaselineMigration).Error; err != nil {
		return fmt.Errorf("scan result baseline migration failed: %w", err)
	}

	return nil
}
