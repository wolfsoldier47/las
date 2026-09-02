ALTER TABLE scan_jobs
ADD COLUMN IF NOT EXISTS os_type TEXT NOT NULL DEFAULT 'linux'
CHECK (os_type IN ('linux','solaris','aix'));
