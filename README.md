
# Ulas Service

Ulas is a Go backend service (Gin + GORM + PostgreSQL) with a Vite/React frontend that orchestrates Ansible Automation Platform (AAP) scans of `/etc/passwd` and `/etc/group`, compares results against registered master baselines and allowed deviations, and opens ServiceNow incidents for unauthorized deviations.


```
{"time":"2026-09-03T15:16:47.407721198Z","level":"INFO","msg":"scan callback received","bytes":70}
[ENVELOPE] job=451853 hosts=1 failed=0 queue_depth=0
{
      "ansible_job_id": "438578",
      "hosts": [
          {
              "datacentre": "ffm",
              "group_file": [

                  {
                      "utmp": "x:22:"
                  }
              ],
              "machine_name": "test.zit.commerzbank.com",
              "os_name": "RedHat",
              "os_type": "Linux",
              "os_version": "8.10",
              "passwd_file": [
                  {
                      "root": "x:0:0:System Administrator of uvdcp06:/root:/bin/bash"
                  }
              ],
              "stage": "prod"
          }
      ],

  }

{
 "ansible_job_id": "438578",
    "failed_hosts":[
        123, 345 #if no failed hosts then send me an empty array
    ]
}

  

```
## Quick start (Podman)

1. Copy environment variables:
   ```bash
   cp .env.example .env
   # edit .env with your values
   ```

2. Start PostgreSQL. The backend uses GORM with `gorm.io/driver/postgres` and auto-migrates the schema on startup.

3. Start dependencies and services with Podman Compose:
   ```bash
   podman compose up --build
   ```

   Or with Podman pods manually:
   ```bash
   podman build -t ulas-backend -f Containerfile .
   podman build -t ulas-frontend -f frontend/Containerfile ./frontend
   podman compose up
   ```

4. Open the frontend at http://localhost and the backend API at http://localhost:8080/api/health.

## Development

### Backend

```bash
go run ./cmd/ulas-service
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Project structure

```
.
├── cmd/ulas-service          # Application entry point
├── internal/
│   ├── aap                   # Ansible Automation Platform client
│   ├── config                # Environment configuration
│   ├── database              # PostgreSQL connection
│   ├── handler               # HTTP handlers
│   ├── middleware            # HTTP middleware
│   ├── repository            # Database repositories
│   ├── router                # Route definitions
│   ├── service               # Business logic
│   └── servicenow            # ServiceNow client
├── models                    # Domain models and enums
├── migrations                # PostgreSQL schema migrations
├── frontend                  # Vite + React + TypeScript UI
└── .agents/skills/ulas.md    # Full build specification skill
```

## API endpoints (implemented)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Service health check |
| POST | `/api/hosts` | Create host |
| GET | `/api/hosts` | List hosts |
| GET | `/api/hosts/:id` | Get host |
| PUT | `/api/hosts/:id` | Update host |
| DELETE | `/api/hosts/:id` | Delete host |
| POST | `/api/baselines` | Create master baseline |
| GET | `/api/baselines` | List baselines |
| GET | `/api/baselines/:id` | Get baseline |
| PUT | `/api/baselines/:id` | Update baseline |
| DELETE | `/api/baselines/:id` | Delete baseline |
| POST | `/api/deviations` | Create allowed deviation |
| GET | `/api/deviations` | List deviations |
| GET | `/api/deviations/:id` | Get deviation |
| PUT | `/api/deviations/:id` | Update deviation |
| DELETE | `/api/deviations/:id` | Delete deviation |
| GET | `/api/scans` | List scan jobs |
| POST | `/api/scans` | Trigger a scan |
| POST | `/api/callbacks/scan` | AAP scan callback |
| GET | `/api/incidents` | List incidents |
| GET | `/api/incidents/:id` | Get incident |
| PUT | `/api/incidents/:id/status` | Update incident status |
| POST | `/api/incidents/:id/servicenow` | Open ServiceNow ticket |
| POST | `/api/incidents/bulk-servicenow` | Bulk open ServiceNow tickets |
| GET | `/api/scans/:id` | Scan job detail with results and incidents |
| GET | `/api/snapshots/:hostId/:fileType/history` | Snapshot history for host/file |
| GET | `/api/snapshots/:hostId/:fileType/changes` | Detected changes for host/file |

## Ansible callback format

The backend accepts callbacks at `POST /api/callbacks/scan`. The preferred format is an envelope with `ansible_job_id`, `hosts`, and optional `failed_hosts`.

### Correct Ansible task

```yaml
- name: Send scan result back to ULAS
  ansible.builtin.uri:
    url: "{{ back_end_base_url }}/api/callbacks/scan"
    method: POST
    body_format: json
    body: "{{ ulas_payload }}"
    status_code: [202]
  delegate_to: localhost
  run_once: true
```

### Correct payload construction

```yaml
ulas_host_result:
  machine_name: "{{ inventory_hostname }}"
  os_type: "{{ ansible_os_family | default('Linux') }}"
  os_version: "{{ ansible_distribution_version | default('') }}"
  os_name: "{{ ansible_distribution | default('') }}"
  stage: "{{ env | default('') }}"
  datacentre: "{{ datacenter | default('') }}"
  passwd_file: "{{ ulas_passwd_entries }}"
  group_file: "{{ ulas_group_entries }}"

ulas_payload:
  ansible_job_id: "{{ ansible_job_id }}"
  hosts: "{{ [ulas_host_result] }}"
  failed_hosts: "{{ ansible_play_hosts_all | difference(ansible_play_batch) }}"
```

### Common mistakes that cause `urlopen error timed out`

- `hosts: [ "{{ ulas_host_result }}" ]` stringifies the dict into a single string element. Use `hosts: "{{ [ulas_host_result] }}"` so Jinja2 renders a real list.
- `body: "{{ ulas_payload | to_json }}"` with `body_format: json` double-encodes the body. Pass the dict directly: `body: "{{ ulas_payload }}"`.
- Running an old backend binary. The handler returns `202 Accepted` immediately only in builds that include the async worker pool.

The backend normalizes `os_type` case-insensitively and maps common Linux variants (`RHEL`, `Ubuntu`, `CentOS`, …) to `linux`.

## Frontend pages

- `/` — Dashboard with health status
- `/hosts` — Host management + snapshot links
- `/baselines` — Master baseline management
- `/deviations` — Allowed deviation management
- `/scans` — Scan history and trigger new scans
- `/scans/:id` — Scan detail with host results and incidents
- `/snapshots/:hostId/:fileType` — Snapshot history and diff view
- `/incidents` — Incident list

## Running tests

```bash
go test ./...
```

Unit tests cover:
- Comparison engine (deviation detection + allowed deviations)
- AAP client
- ServiceNow client

## Architecture

The backend follows a layered architecture:

- **Handler** — HTTP request/response handling
- **Service** — business logic (scan orchestration, comparison, ticketing)
- **Repository** — data access layer
- **Models** — domain types shared across layers

## License

Proprietary
