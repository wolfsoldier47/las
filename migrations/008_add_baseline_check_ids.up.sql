-- Privilege list support: entries flagged here get their uid/gid compared
-- against the baseline; all other entries ignore uid/gid differences.
ALTER TABLE master_baselines ADD COLUMN check_ids BOOLEAN NOT NULL DEFAULT FALSE;
