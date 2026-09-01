# Ulas Service — Build Skill

Use this skill when asked to build or extend **ulas-service**: a Go backend + Vite/React frontend that orchestrates Ansible Automation Platform (AAP) scans of `/etc/passwd` and `/etc/group`, compares captured files against master baselines and allowed deviations, and opens ServiceNow incidents for unauthorized deviations.

## Project context

- **Backend**: Go 1.25+, Gin web framework, layered architecture (handler → service → repository).
- **Frontend**: Vite + React + TypeScript.
- **Database**: PostgreSQL (schema managed with migrations, not ORM auto-migrate).
- **Message/queue**: none required for MVP; scans are triggered and completed via AAP callback.
- **External APIs**: AAP (Tower/Controller), ServiceNow.
- **Deployment**: containerized with Podman / Podman Compose (`Containerfile`, `compose.yml`).

The repository already contains domain models in `models/`. Reuse and extend them instead of replacing them.

## Domain summary

### Core entities

| Entity | Purpose |
|--------|---------|
| `Host` | Managed machine (Linux, Solaris, AIX). |
| `MasterBaseline` | Admin-defined "should be" `/etc/passwd` or `/etc/group` entry. `host_id = NULL` means global. |
| `MasterBaselineVersion` | Audit history of every baseline change. |
| `AllowedDeviation` | Pre-approved exception for a host/file/key. |
| `ScanJob` | AAP job launched from the portal. |
| `ScanResult` | Per-host result of one scan job. |
| `HostFileSnapshot` | Raw captured file content for a scan. Immutable. |
| `HostFileChange` | Detected change between two consecutive snapshots for a host/file. |
| `Incident` | Unauthorized deviation that must be tracked / ticketed. |
| `ServiceNowTicket` | One-to-one link between an incident and a ServiceNow ticket. |

### Shared enums (already present)

- `models.FileType` — `passwd`, `group`
- `models.OSType` — `linux`, `solaris`, `aix` (case-insensitive JSON unmarshaling)
- `models.IncidentSeverity` — `critical`, `high`, `medium`, `low`
- `models.IncidentStatus` — `open`, `acknowledged`, `in_progress`, `resolved`, `closed`
- `models.ScanJobStatus` — `initiating`, `running`, `completed`, `failed`, `timeout`, `cancelled`
- `models.ScanResultStatus` — `pending`, `success`, `failed`, `deviation_found`, `allowed_deviation`
- `models.ScanProcessingStatus` — `pending`, `processing`, `processed`, `failed`

## Layered backend architecture

```
cmd/ulas-service/
    main.go                 # wiring only
internal/
    config/                 # env/config loading
    handler/                # HTTP handlers (Gin)
        scan_handler.go
        host_handler.go
        baseline_handler.go
        deviation_handler.go
        incident_handler.go
        snapshot_handler.go
    service/                # business logic
        scan_service.go
        comparison_service.go
        baseline_service.go
        deviation_service.go
        incident_service.go
        servicenow_service.go
        aap_client.go       # AAP API client
    repository/             # database access
        pg/                 # PostgreSQL implementations
            scan_repository.go
            host_repository.go
            baseline_repository.go
            snapshot_repository.go
            incident_repository.go
            deviation_repository.go
    middleware/
        logging.go
        error.go
    models/                 # domain models (existing, do not duplicate)
```

Rules for layers:

1. **Handler** — parses HTTP requests, calls services, writes JSON responses. No DB or external API calls.
2. **Service** — contains business rules: scan orchestration, baseline comparison, deviation classification, incident creation, ServiceNow integration. Services hold repository interfaces and the AAP client.
3. **Repository** — SQL via `database/sql` or `jmoiron/sqlx`. Returns domain models. No business logic.
4. **Models** — plain structs + enums. Existing files in `models/` are the source of truth.

## Database schema (PostgreSQL)

Use a migration tool such as `golang-migrate` or `pressly/goose`. Create migrations in `migrations/`.

Key tables:

```sql
CREATE TABLE hosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname TEXT NOT NULL UNIQUE,
    os_type TEXT NOT NULL CHECK (os_type IN ('linux','solaris','aix')),
    environment TEXT,
    datacenter TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE master_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID REFERENCES hosts(id),
    os_type TEXT NOT NULL CHECK (os_type IN ('linux','solaris','aix')),
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    entry_key TEXT NOT NULL,
    entry_value TEXT NOT NULL,
    is_global BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (host_id, os_type, file_type, entry_key)
);

CREATE TABLE master_baseline_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_id UUID NOT NULL REFERENCES master_baselines(id) ON DELETE CASCADE,
    version INT NOT NULL,
    entry_value TEXT NOT NULL,
    change_reason TEXT,
    changed_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (baseline_id, version)
);

CREATE TABLE allowed_deviations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    entry_key TEXT NOT NULL,
    entry_value TEXT,
    justification TEXT NOT NULL,
    approved_by TEXT NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scan_jobs (
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
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scan_results (
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

CREATE TABLE host_file_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_result_id UUID NOT NULL REFERENCES scan_results(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
    raw_content TEXT NOT NULL,
    line_count INT NOT NULL DEFAULT 0,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE host_file_changes (
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

CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_number TEXT NOT NULL UNIQUE,
    scan_result_id UUID NOT NULL REFERENCES scan_results(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id),
    file_type TEXT NOT NULL CHECK (file_type IN ('passwd','group')),
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

CREATE TABLE service_now_tickets (
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
```

Add indexes on foreign keys and common query columns (`scan_job_id`, `host_id`, `file_type`, `status`).

## Backend API surface

### Admin / portal endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/hosts` | Register a host. |
| GET | `/api/hosts` | List hosts. |
| GET | `/api/hosts/:id` | Get host details. |
| POST | `/api/baselines` | Create master baseline. |
| PUT | `/api/baselines/:id` | Update baseline value (creates version row). |
| GET | `/api/baselines` | List baselines. |
| POST | `/api/deviations` | Register allowed deviation. |
| GET | `/api/deviations` | List allowed deviations. |
| POST | `/api/scans` | Trigger a scan. |
| GET | `/api/scans` | List scan jobs / history. |
| GET | `/api/scans/:id` | Get scan job summary. |
| GET | `/api/scans/:id/results` | Per-host results. |
| GET | `/api/scans/:id/deviations` | Deviations for a scan. |
| POST | `/api/incidents/:id/servicenow` | Open a ServiceNow ticket for one incident. |
| POST | `/api/incidents/bulk-servicenow` | Open tickets for selected incidents (idempotent). |
| GET | `/api/incidents` | List incidents. |
| GET | `/api/snapshots/:hostId/:fileType/history` | Snapshot history with change detection. |

### AAP callback endpoint

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/callbacks/scan` | Receives AAP scan result JSON. |

Callback payload shape (AAP posts this):

```json
{
  "job_id": "12345",
  "machine_name": "test123.zit.commerzbank.com",
  "machine_type": "solaris",
  "os_version": "9.7",
  "os_name": "RHEL",
  "passwd_file": [
    {"gluster": "999:999:/var/lib:/sbin/nologin"}
  ],
  "group_file": [
    {"gluster": "x:896:sam,sam2,sam3"}
  ],
  "timestamp": "2026-07-25T12:00:00Z",
  "ansible_facts": {}
}
```

Use `models.CallbackPayload` for binding. Note that `machine_type` is unmarshaled into `models.OSType`, which normalizes casing.

## Key workflows

### 1. Trigger scan

1. Admin selects inventory/limit in the portal and clicks **Scan**.
2. Handler calls `ScanService.InitiateScan(ctx, templateName, limit, initiatedBy)`.
3. Service asks `AAPClient`:
   - `GET /api/v2/job_templates/?name=<templateName>` to resolve template ID.
   - `POST /api/v2/job_templates/<id>/launch/` with `limit` and `extra_vars`.
4. Create `ScanJob` row with status `initiating`, store returned `ansible_job_id`.
5. Return `ScanResultResponse` with job ID and initial status.
6. Poll AAP job status or wait for callbacks; update `ScanJob` status (`running`, `completed`, `failed`).

### 2. Receive callback

1. AAP posts to `/api/callbacks/scan`.
2. Handler accepts either a single `models.CallbackPayload` or an array of `models.CallbackPayload` objects.
3. For each payload, service resolves `host_id` by `machine_name` (create host if missing, with normalized `os_type`).
4. Create `ScanResult` row for `(scan_job_id, host_id)`.
5. Store raw file contents as `HostFileSnapshot` rows (one per file type).
6. Update `ScanJob` counters (`callbacks_received`, `successful_hosts`).
7. Queue comparison: set `ScanResult.ProcessingStatus = pending`.

### 3. Compare and classify

Run in the same request or a background goroutine/worker:

1. Load applicable baselines for host (`host_id = <host> OR is_global = true`) and `os_type`.
2. For each `passwd` and `group` entry:
   - Find expected value from baselines.
   - Find actual value from snapshot.
   - If missing in baseline → deviation.
   - If mismatched → check `AllowedDeviation` for `(host_id, file_type, entry_key, entry_value)` that is active and not expired.
   - If allowed → mark as allowed deviation.
   - If not allowed → create `Incident` with severity based on file/entry rules.
3. Create `HostFileChange` rows by comparing with the previous snapshot for the same host/file.
4. Update `ScanResult` status and `deviations_found` count.

### 4. Open ServiceNow tickets

1. Admin selects incidents in the portal.
2. Frontend posts to `/api/incidents/bulk-servicenow` with incident IDs.
3. Service inserts/updates `ServiceNowTicket` rows with `ticket_opened = true`.
4. Background worker or synchronous call creates ServiceNow incident via REST API.
5. Store `ticket_number`, `ticket_url`, `opened_at`; flip `Incident.ServiceNowTicketOpened = true`.
6. Ensure idempotency: skip incidents that already have an opened ticket.

## Frontend (Vite + React + TypeScript)

Structure:

```
frontend/
  src/
    api/              # axios clients per domain
    components/       # reusable UI
    pages/
      Dashboard.tsx
      HostsPage.tsx
      BaselinesPage.tsx
      DeviationsPage.tsx
      ScanHistoryPage.tsx
      ScanDetailPage.tsx
      IncidentsPage.tsx
    hooks/
    types/            # generated/typed API shapes
    App.tsx
    main.tsx
  index.html
  package.json
  vite.config.ts
  tsconfig.json
```

Key pages:

- **Dashboard**: stats, latest scans, open incidents.
- **Hosts**: CRUD for hosts.
- **Baselines**: manage master baselines and versions.
- **Deviations**: manage allowed deviations.
- **Scan History**: list scans, drill into per-host results and deviations.
- **Scan Detail**: select hosts/deviations and open ServiceNow tickets.

Keep API base URL configurable via env var (e.g. `VITE_API_URL`).

## Containerization

Provide `Containerfile` for the backend and a multi-stage build for the frontend, plus `compose.yml` for Podman Compose.

```dockerfile
# backend/Containerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ulas-service ./cmd/ulas-service

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/ulas-service .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
CMD ["./ulas-service"]
```

Run with Podman:

```bash
podman compose up --build
```

`docker-compose.yml` should include:

- `postgres`
- `ulas-backend`
- `ulas-frontend` (served via nginx or Vite preview for dev)
- optional `migrations` service using `golang-migrate`.

## Implementation phases

When building from scratch, do it in this order:

1. **Bootstrap**: config, DB connection, migrations, router, basic health endpoint.
2. **Hosts & baselines**: full CRUD + repository + handlers.
3. **AAP client**: resolve template by name and launch job.
4. **Scan trigger**: create scan job, launch AAP job, return job ID.
5. **Callback endpoint**: bind payload, create scan result and snapshots.
6. **Comparison engine**: compare snapshot vs baselines + deviations, create incidents.
7. **Scan history & deviations UI**: list scans, results, deviations.
8. **ServiceNow integration**: open tickets one-by-one and in bulk.
9. **Snapshot history & change detection**: compare consecutive snapshots.
10. **Frontend pages**: tie everything together.
11. **Docker & compose**: containerize and add README.

## Testing

- Unit tests for `comparison_service` with table-driven cases.
- Repository tests using `ory/dockertest` or `testcontainers-go` against PostgreSQL.
- Handler tests with mocked services.
- AAP client tests with `httptest` server.

## Conventions

- Follow existing model names and tags in `models/`.
- Use `uuid.UUID` for IDs, `time.Time` or `*time.Time` for timestamps.
- Keep DB enum values in sync with `...Values` slices in `models/`.
- Return structured errors from services; handlers map them to HTTP status codes.
- Log at service boundaries, never log secrets.
