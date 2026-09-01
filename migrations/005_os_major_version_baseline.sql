-- Track which OS major version a scan result was compared against and whether no baseline existed.
ALTER TABLE scan_results
    ADD COLUMN IF NOT EXISTS baseline_version_at_scan INT,
    ADD COLUMN IF NOT EXISTS no_baseline BOOLEAN NOT NULL DEFAULT false;
