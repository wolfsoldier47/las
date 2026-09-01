DROP INDEX IF EXISTS idx_service_now_ticket_incidents_incident_id;
DROP INDEX IF EXISTS idx_service_now_tickets_host_scan;
DROP TABLE IF EXISTS service_now_ticket_incidents;

-- Host-based tickets created by the new code cannot be rolled back to the old
-- 1-to-1 schema, so remove them before re-adding the NOT NULL/UNIQUE constraints.
DELETE FROM service_now_tickets WHERE incident_id IS NULL;

ALTER TABLE service_now_tickets
    DROP COLUMN IF EXISTS scan_job_id,
    ALTER COLUMN incident_id SET NOT NULL,
    ADD CONSTRAINT service_now_tickets_incident_id_key UNIQUE (incident_id);
