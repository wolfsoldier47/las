-- Revert the active-scope index to the old broken definition.
DROP INDEX IF EXISTS idx_master_baselines_active_scope;

CREATE UNIQUE INDEX IF NOT EXISTS idx_master_baselines_active_scope
    ON master_baselines (host_id, os_type, file_type)
    WHERE is_active = true;
