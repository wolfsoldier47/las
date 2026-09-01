ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS allowed_deviations JSONB DEFAULT '[]';
