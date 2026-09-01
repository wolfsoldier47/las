-- Add baseline snapshot to scan jobs so the portal can show which master files
-- were active when the scan was initiated.
ALTER TABLE scan_jobs
    ADD COLUMN IF NOT EXISTS baseline_snapshot JSONB;
