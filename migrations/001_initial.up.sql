CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS hosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname TEXT NOT NULL UNIQUE,
    os_type TEXT NOT NULL CHECK (os_type IN ('linux','solaris','aix')),
    os_name TEXT,
    os_version TEXT,
    environment TEXT,
    datacenter TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS master_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID REFERENCES hosts(id),
    os_type TEXT NOT NULL CHECK (os_type IN ('linux','solaris','aix')),
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    entry_key TEXT NOT NULL,
    entry_value TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_global BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, os_type, file_type, entry_key, version)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_master_baselines_active_scope
    ON master_baselines (host_id, os_type, file_type, version)
    WHERE is_active = true;

CREATE TABLE IF NOT EXISTS master_baseline_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_id UUID NOT NULL REFERENCES master_baselines(id) ON DELETE CASCADE,
    version INT NOT NULL,
    entry_value TEXT NOT NULL,
    change_reason TEXT,
    changed_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (baseline_id, version)
);

CREATE TABLE IF NOT EXISTS allowed_deviations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname TEXT NOT NULL,
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    entry_key TEXT NOT NULL,
    entry_value TEXT,
    justification TEXT NOT NULL,
    approved_by TEXT NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hostname, file_type, entry_key)
);

CREATE TABLE IF NOT EXISTS scan_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ansible_job_id TEXT,
    job_template_id INT NOT NULL,
    limit TEXT,
    status TEXT NOT NULL CHECK (status IN ('initiating','running','completed','failed','timeout','cancelled')),
    initiated_by TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    total_hosts INT NOT NULL DEFAULT 0,
    callbacks_received INT NOT NULL DEFAULT 0,
    successful_hosts INT NOT NULL DEFAULT 0,
    failed_hosts INT NOT NULL DEFAULT 0,
    failed_host_names JSONB,
    error_message TEXT,
    baseline_snapshot JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scan_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id),
    status TEXT NOT NULL CHECK (status IN ('pending','success','failed','deviation_found','allowed_deviation')),
    error_message TEXT,
    processing_status TEXT NOT NULL CHECK (processing_status IN ('pending','processing','processed','failed')),
    deviations_found INT NOT NULL DEFAULT 0,
    received_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scan_job_id, host_id)
);

CREATE TABLE IF NOT EXISTS host_file_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_result_id UUID NOT NULL REFERENCES scan_results(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    raw_content TEXT NOT NULL,
    line_count INT NOT NULL DEFAULT 0,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS host_file_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES hosts(id),
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    change_type TEXT NOT NULL CHECK (change_type IN ('added','removed','modified')),
    previous_content TEXT,
    current_content TEXT,
    previous_scan_job_id UUID REFERENCES scan_jobs(id),
    current_scan_job_id UUID NOT NULL REFERENCES scan_jobs(id),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_number TEXT NOT NULL UNIQUE,
    scan_result_id UUID NOT NULL REFERENCES scan_results(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id),
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    entry_key TEXT NOT NULL,
    expected_value TEXT,
    actual_value TEXT NOT NULL,
    baseline_version_at_scan INT,
    severity TEXT NOT NULL CHECK (severity IN ('critical','high','medium','low')),
    status TEXT NOT NULL CHECK (status IN ('open','acknowledged','in_progress','resolved','closed')),
    notes TEXT,
    service_now_ticket_opened BOOLEAN NOT NULL DEFAULT false,
    resolution TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS service_now_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL UNIQUE REFERENCES incidents(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id),
    ticket_number TEXT,
    ticket_url TEXT,
    ticket_opened BOOLEAN NOT NULL DEFAULT false,
    opened_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scan_results_scan_job_id ON scan_results(scan_job_id);
CREATE INDEX IF NOT EXISTS idx_scan_results_host_id ON scan_results(host_id);
CREATE INDEX IF NOT EXISTS idx_host_file_snapshots_scan_result_id ON host_file_snapshots(scan_result_id);
CREATE INDEX IF NOT EXISTS idx_host_file_snapshots_host_id ON host_file_snapshots(host_id);
CREATE INDEX IF NOT EXISTS idx_host_file_snapshots_host_type_time ON host_file_snapshots(host_id, file_type, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_host_file_changes_host_id ON host_file_changes(host_id);
CREATE INDEX IF NOT EXISTS idx_incidents_scan_result_id ON incidents(scan_result_id);
CREATE INDEX IF NOT EXISTS idx_incidents_host_id ON incidents(host_id);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
