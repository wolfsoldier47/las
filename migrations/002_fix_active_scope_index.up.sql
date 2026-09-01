-- Fix the active-scope unique index so it covers version.
-- Without version, only one active entry row per scope could exist,
-- which breaks multi-entry master files.
DROP INDEX IF EXISTS idx_master_baselines_active_scope;

CREATE UNIQUE INDEX IF NOT EXISTS idx_master_baselines_active_scope
    ON master_baselines (host_id, os_type, file_type, version)
    WHERE is_active = true;
