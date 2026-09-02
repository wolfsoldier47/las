-- Existing rows created before the os_type column existed may be NULL.
-- Backfill them to linux so the NOT NULL constraint can be applied safely.
UPDATE scan_jobs SET os_type = 'linux' WHERE os_type IS NULL;

ALTER TABLE scan_jobs
ALTER COLUMN os_type SET NOT NULL;

ALTER TABLE scan_jobs
ALTER COLUMN os_type SET DEFAULT 'linux';
