-- ServiceNow tickets are now opened per host per scan job instead of per incident.
-- A single ticket can cover multiple incidents for the same host in the same scan.

ALTER TABLE service_now_tickets
    DROP CONSTRAINT IF EXISTS service_now_tickets_incident_id_key,
    ALTER COLUMN incident_id DROP NOT NULL,
    ADD COLUMN scan_job_id UUID REFERENCES scan_jobs(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS service_now_ticket_incidents (
    service_now_ticket_id UUID NOT NULL REFERENCES service_now_tickets(id) ON DELETE CASCADE,
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    PRIMARY KEY (service_now_ticket_id, incident_id)
);

CREATE INDEX IF NOT EXISTS idx_service_now_tickets_host_scan ON service_now_tickets(host_id, scan_job_id);
CREATE INDEX IF NOT EXISTS idx_service_now_ticket_incidents_incident_id ON service_now_ticket_incidents(incident_id);

-- Migrate existing 1-to-1 relationships into the join table.
INSERT INTO service_now_ticket_incidents (service_now_ticket_id, incident_id)
SELECT id, incident_id FROM service_now_tickets WHERE incident_id IS NOT NULL
ON CONFLICT DO NOTHING;
