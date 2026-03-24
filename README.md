# EnvNexus

[中文版本](README.zh-CN.md)

**EnvNexus** is an AI-native platform for environment governance, secure local diagnosis, and guided repair. It combines a multi-tenant control plane, an Electron desktop client, and a local Go execution core to deliver tenant-specific distribution, policy-driven diagnosis, approval-based repair, and end-to-end auditability.

---

## What Problem Does EnvNexus Solve?

Traditional remote support tools grant unrestricted shell access to endpoints. This creates security, compliance, and audit risks — especially in enterprise environments where hundreds of devices are managed by distributed teams.

EnvNexus takes a different approach:

- **Default read-only**: the agent only runs read-only diagnostic tools unless a write action is explicitly approved.
- **Approval gate**: every repair action (proxy toggle, config change, container reload) must pass through a multi-step approval state machine before execution.
- **Policy-driven**: each tenant defines which models, tools, and risk levels are allowed on their managed devices.
- **Full audit trail**: every session, tool invocation, approval decision, and rollback is recorded and queryable.
- **Local-first execution**: the AI diagnosis engine runs on the endpoint. The platform orchestrates, but never executes commands on the device directly.

### Typical Use Cases

- End-user self-service diagnosis: "my network is broken" → AI diagnoses → suggests repair → user approves → agent executes
- Remote support with audit: support engineer initiates a session → agent collects data → repair requires operator approval
- Enterprise endpoint governance: 500 devices across 10 tenants, each with distinct model/policy/tool configurations
- Private deployment: customer manages everything on-premise, including LLM keys and audit storage

### What EnvNexus Is NOT

- It is **not** a remote desktop or arbitrary shell tool
- It is **not** an RMM (Remote Monitoring and Management) agent that can run any command
- It does **not** bypass local policy — the local agent always has the final say

---

## Architecture Overview

### System Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              PLATFORM SIDE                                      │
│                         (Docker Compose / K8s)                                  │
│                                                                                 │
│  ┌──────────────┐     ┌───────────────────┐     ┌──────────────────────┐       │
│  │ console-web  │────>│   platform-api    │────>│    session-gateway   │       │
│  │ (Next.js 14) │     │   (Go / Gin)      │     │   (Go / WebSocket)  │       │
│  │  :3000       │     │   :8080           │     │   :8081              │       │
│  └──────────────┘     └─────┬──┬──┬───────┘     └──────────┬───────────┘       │
│                             │  │  │                         │                   │
│              ┌──────────────┘  │  └──────────────┐          │                   │
│              v                 v                  v          v                   │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐          │
│  │  MySQL 8      │  │   Redis      │  │    MinIO     │  │ Redis    │          │
│  │  (主数据库)    │  │  (缓存/队列)  │  │ (对象存储)   │  │ (pub/sub)│          │
│  │  :3306        │  │  :6379       │  │  :9000       │  │          │          │
│  └───────────────┘  └──────────────┘  └──────────────┘  └──────────┘          │
│                             │                                                   │
│                             v                                                   │
│                     ┌───────────────┐                                           │
│                     │  job-runner   │                                           │
│                     │  (Go Workers) │                                           │
│                     │  :8082        │                                           │
│                     └───────────────┘                                           │
└─────────────────────────────────────────────────────────────────────────────────┘
                              │ HTTPS/WSS
                              │
┌─────────────────────────────┼───────────────────────────────────────────────────┐
│                    ENDPOINT SIDE                                                │
│                                                                                 │
│  ┌──────────────────┐     ┌───────────────────────────────────────────┐         │
│  │  agent-desktop   │────>│              agent-core                   │         │
│  │  (Electron 30)   │ IPC │          (Go / localhost:17700)           │         │
│  │                  │     │                                           │         │
│  │  - System Tray   │     │  ┌─────────┐ ┌──────────┐ ┌──────────┐  │         │
│  │  - Dashboard     │     │  │ LLM     │ │ Tool     │ │Governance│  │         │
│  │  - Chat UI       │     │  │ Router  │ │ Registry │ │ Engine   │  │         │
│  │  - Approvals     │     │  │(7 provs)│ │(10 tools)│ │(baseline)│  │         │
│  │  - Settings      │     │  └─────────┘ └──────────┘ └──────────┘  │         │
│  └──────────────────┘     │  ┌─────────┐ ┌──────────┐ ┌──────────┐  │         │
│                           │  │ Policy  │ │ Audit    │ │ Diagnosis│  │         │
│                           │  │ Engine  │ │ Client   │ │ Engine   │  │         │
│                           │  └─────────┘ └──────────┘ └──────────┘  │         │
│                           │  ┌──────────────────────────────────┐    │         │
│                           │  │        SQLite + Local Files       │    │         │
│                           │  └──────────────────────────────────┘    │         │
│                           └───────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```text
                    ┌─────────────────── CONTROL PLANE ───────────────────┐
                    │                                                      │
  Admin Browser ──> │ console-web ──HTTP──> platform-api ──SQL──> MySQL   │
                    │                           │                          │
                    │                      ┌────┴────┐                    │
                    │                      │         │                    │
                    │                    Redis     MinIO                  │
                    │                   (cache)   (packages              │
                    │                             & archives)            │
                    │                                                      │
                    │ job-runner <──poll DB──> MySQL                      │
                    │   (7 workers: cleanup, build, scan, expire, flush)  │
                    └──────────────────────────┬──────────────────────────┘
                                               │
                           ┌───────────────────┼───────────────────┐
                           │  HTTP (REST)       │  WebSocket        │
                           │  - enroll          │  - session events │
                           │  - heartbeat       │  - tool results   │
                           │  - config pull     │  - approvals      │
                           │  - audit upload    │                   │
                           v                    v                   │
                    ┌──────────────────────────────────────────────┐│
                    │              agent-core                       ││
                    │                                               ││
                    │  1. Boot → enroll/load identity               ││
                    │  2. Pull remote config + policy               ││
                    │  3. Connect WebSocket to session-gateway      ││
                    │  4. Heartbeat loop (every 60s)                ││
                    │  5. Receive diagnosis request                 ││
                    │  6. LLM router → provider → structured output││
                    │  7. Tool execution (read-only or approved)    ││
                    │  8. Audit events → platform + local SQLite    ││
                    │  9. Governance: baseline capture + drift scan ││
                    └──────────────────────────────────────────────┘│
                           ^                                        │
                           │ IPC (Electron contextBridge)           │
                    ┌──────┴──────┐                                 │
                    │agent-desktop│    User approves/rejects ───────┘
                    │  (Electron) │    repair actions here
                    └─────────────┘
```

### Service Call Chain (End-to-End Diagnosis Flow)

```text
1. User types "my network is broken" in agent-desktop chat UI
       │
       ▼
2. agent-desktop ──IPC──> agent-core local API (POST /local/v1/diagnose)
       │
       ▼
3. agent-core Diagnosis Engine:
   a. Collects system context (network config, DNS, processes, disk)
   b. Sends context + user query to LLM Router
   c. LLM Router picks provider (OpenAI / DeepSeek / Anthropic / Ollama / ...)
   d. LLM returns structured diagnosis with recommended tools
       │
       ▼
4. Read-only tools execute immediately (read_network_config, read_system_info, ...)
   Write tools → create ApprovalRequest
       │
       ▼
5. agent-core ──HTTP POST──> platform-api (create approval request)
   platform-api stores in MySQL, emits audit event
       │
       ▼
6. agent-desktop polls pending approvals, shows to user:
   "Tool: proxy.toggle | Risk: L1 | Action: enable proxy http://proxy:8080"
   [Approve] [Reject]
       │
       ▼
7. User clicks [Approve]
   agent-desktop ──IPC──> agent-core ──HTTP POST──> platform-api (approve)
       │
       ▼
8. agent-core executes the tool, records result
   agent-core ──HTTP POST──> platform-api (audit event: tool.executed, succeeded)
       │
       ▼
9. Result displayed in agent-desktop chat UI
   Audit trail visible in console-web audit events page
```

---

## Module Reference

### Platform Services

| Module | Language | Port | Description |
|--------|----------|------|-------------|
| **platform-api** | Go (Gin) | 8080 | Central REST API. Handles authentication (JWT access/refresh/device/session tokens), tenant CRUD, profile management (model/policy/agent), device registration, session management, approval state machine, audit events, RBAC (5 roles, 17 permissions), webhook dispatch, usage metrics, license validation. 25+ API resource groups. |
| **session-gateway** | Go (Gorilla WS) | 8081 | WebSocket relay between platform and agents. Authenticates via session tokens. Routes session events (diagnosis requests, tool results, approvals) with event_id deduplication. Uses Redis pub/sub for horizontal scaling. |
| **job-runner** | Go | 8082 | Background worker service. Runs 7 workers: `token_cleanup`, `link_cleanup`, `audit_flush` (MinIO + local FS fallback), `session_cleanup`, `approval_expiry`, `package_build`, `governance_scan`. Polls MySQL for jobs. |
| **console-web** | Next.js 14 | 3000 | Admin web console. 12+ pages: login, tenant management, device list (online/offline status), session list + detail, audit events (filterable), model/policy/agent profiles, download packages. Centralized i18n (zh/en). Unified API client. |

### Endpoint Applications

| Module | Language | Description |
|--------|----------|-------------|
| **agent-core** | Go | Local execution core. Runs on the managed endpoint as `enx-agent`. Exposes a localhost-only API on port 17700. Contains: LLM Router (7 providers: OpenAI, Anthropic, DeepSeek, Gemini, OpenRouter, Ollama, local), Tool Registry (10 tools), Diagnosis Engine (5-step), Policy Engine, Governance Engine (baseline + drift), Audit Client (buffered upload), SQLite local store. Supports offline degraded mode. |
| **agent-desktop** | Electron 30 | Desktop UI shell. System tray with connection status. 5 pages: dashboard (status cards), diagnosis chat (multi-turn), pending approvals (approve/reject), session history, settings (language, platform URL, log level, agent binary path). Manages agent-core subprocess lifecycle. 10 IPC channels via contextBridge. |

### Tool Registry (agent-core)

| Tool | Risk | R/W | Description |
|------|------|-----|-------------|
| `read_network_config` | L0 | Read | Collect IP addresses, DNS servers, routes |
| `read_system_info` | L0 | Read | OS, hostname, CPU, memory |
| `read_disk_usage` | L0 | Read | Disk partitions and usage |
| `read_process_list` | L0 | Read | Running processes (PID, name) |
| `flush_dns` | L1 | Write | Flush system DNS cache |
| `service.restart` | L2 | Write | Restart a system service |
| `cache.rebuild` | L1 | Write | Rebuild application cache |
| `proxy.toggle` | L1 | Write | Enable/disable system proxy (Linux/macOS/Windows) |
| `config.modify` | L1 | Write | Modify whitelisted config keys in env file |
| `container.reload` | L2 | Write | Reload container or process (docker/systemd/SIGHUP) |

### Infrastructure

| Component | Purpose |
|-----------|---------|
| **MySQL 8** | Primary data store. 25 tables (13 core + 12 extension). Stores tenants, users, roles, devices, sessions, audit events, approval requests, profiles, webhooks, jobs, metrics, licenses. |
| **Redis** | Cache (token blacklist, rate limiting), session-gateway pub/sub for WebSocket fan-out. |
| **MinIO** | S3-compatible object storage. Stores agent distribution packages and audit archives. Falls back to local filesystem when unavailable. |
| **SQLite** | Agent-side local persistence. Stores sessions, audit events, config cache, governance baselines and drifts. Enables offline operation. |

### Security & Auth

| Mechanism | Description |
|-----------|-------------|
| **JWT Access Token** | 1-hour expiry. Used by console-web. Contains user_id, tenant_id. |
| **JWT Refresh Token** | 7-day expiry. `POST /api/v1/auth/refresh` to get new access token. |
| **JWT Device Token** | 1-year expiry. Issued at enrollment. Used by agent-core for API calls. Supports rotation via `POST /devices/:id/rotate-token`. |
| **JWT Session Token** | 30-minute expiry. Scoped to a specific WebSocket session. |
| **RBAC** | 5 preset roles: `platform_super_admin`, `tenant_admin`, `security_auditor`, `ops_operator`, `read_only_observer`. 17 permission constants. `RequirePermission` middleware. |
| **Rate Limiting** | Login: 10 req/min/IP. General API: 50 req/s/IP. |
| **CORS** | Configurable via `ENX_CORS_ALLOWED_ORIGINS`. |

---

## Strengths & Limitations

### Strengths

- **Security-first design**: no arbitrary shell, structured tool execution only, approval gate for all writes
- **Multi-tenant isolation**: database, cache, object storage, and audit are all tenant-scoped
- **AI-native diagnosis**: LLM router with 7 provider backends, structured output, tool-calling pattern
- **Offline-capable**: agent-core degrades gracefully when platform is unreachable (read-only tools, local SQLite)
- **Full audit trail**: every action is recorded, queryable, archivable (MinIO or local FS fallback)
- **Extensible tool system**: add new tools by implementing a single Go interface
- **Private deployment ready**: Helm chart, offline license key, local LLM (Ollama), no cloud dependency required

### Current Limitations

- **No automated tests beyond unit**: integration and E2E test coverage is low
- **No OpenAPI/Swagger docs**: API documentation is code-only
- **No LDAP/SAML/OIDC**: enterprise SSO integration is not yet implemented
- **No Stripe billing**: SaaS payment integration is planned but not built
- **No auto-update for desktop**: Electron auto-updater is not integrated
- **Shared library is minimal**: `libs/shared/` has only base models, not fully extracted
- **No PII redaction in audit exports**: data masking pipeline is not yet implemented
- **Redis job queue not implemented**: job-runner uses DB polling, not Redis-based queue

---

## Repository Layout

```text
envnexus/
├── apps/
│   ├── console-web/              # Next.js 14 admin console (TypeScript)
│   │   ├── src/app/              #   Pages: login, tenants, devices, sessions, audit, profiles
│   │   ├── src/components/       #   Sidebar, Header
│   │   └── src/lib/              #   API client, i18n dictionary
│   ├── agent-desktop/            # Electron 30 desktop shell (TypeScript)
│   │   └── src/
│   │       ├── main/main.ts      #   Main process: tray, window, IPC, subprocess mgmt
│   │       ├── preload/          #   Context bridge (10 IPC channels)
│   │       └── renderer/         #   Multi-page HTML UI
│   └── agent-core/               # Go local execution core
│       ├── cmd/enx-agent/        #   Entry point
│       └── internal/
│           ├── bootstrap/        #   10-step boot sequence
│           ├── llm/              #   Router + 7 providers
│           ├── tools/            #   10 structured tools (system/network/service/cache)
│           ├── governance/       #   Baseline capture + drift detection
│           ├── diagnosis/        #   5-step diagnosis engine
│           ├── store/            #   SQLite local persistence
│           ├── session/          #   WebSocket client
│           └── api/              #   Local HTTP server (:17700)
├── services/
│   ├── platform-api/             # Go central API (Gin + GORM)
│   │   ├── cmd/platform-api/     #   Entry point + DI wiring
│   │   ├── internal/
│   │   │   ├── domain/           #   Entities (DDD): Session, ApprovalRequest, Role, Webhook, ...
│   │   │   ├── repository/       #   MySQL repositories (GORM)
│   │   │   ├── service/          #   Business logic: auth, rbac, webhook, metrics, license, ...
│   │   │   ├── handler/          #   HTTP handlers (console API + agent API)
│   │   │   ├── middleware/       #   JWT auth, RBAC, rate limiting, CORS, response envelope
│   │   │   ├── infrastructure/   #   Redis, MinIO, Gateway clients
│   │   │   └── dto/              #   Request/response DTOs
│   │   └── migrations/           #   SQL migrations (auto-run on startup)
│   ├── session-gateway/          # Go WebSocket gateway
│   │   └── internal/handler/ws/  #   WS handler with event dedup
│   └── job-runner/               # Go background workers
│       └── internal/worker/      #   7 workers
├── libs/shared/                  # Shared Go library (errors, base models)
├── deploy/
│   ├── docker/                   # Docker Compose deployment
│   │   ├── docker-compose.yml
│   │   ├── Dockerfile.*          #   Per-service Dockerfiles
│   │   └── .env.example
│   └── k8s/helm/envnexus/       # Kubernetes Helm chart
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/            #   4 Deployments + Services, Ingress, Secrets
├── scripts/
│   ├── smoke-test.sh             # MVP 12-step smoke test
│   └── seed.sh                   # Seed default tenant + admin
├── docs/
│   ├── envnexus-proposal.md      # Full product proposal
│   ├── development-roadmap.md    # Phase 0-6 roadmap with completion status
│   └── commercialization-plan.md # Business plan
├── Makefile
└── README.md
```

---

## Deployment

### Option 1: Docker Compose (Recommended for Development & Single-Host)

**Prerequisites**: Docker 24+, Docker Compose v2, Git

```bash
# 1. Clone
git clone https://github.com/zy-eagle/envnexus.git
cd envnexus

# 2. Configure secrets
cp deploy/docker/.env.example deploy/docker/.env
# IMPORTANT: edit .env and set these to secure random values:
#   ENX_JWT_SECRET
#   ENX_DEVICE_TOKEN_SECRET
#   ENX_SESSION_TOKEN_SECRET

# 3. Start everything
cd deploy/docker
docker compose up -d

# 4. Verify health
curl http://localhost:8080/healthz    # platform-api
curl http://localhost:8081/healthz    # session-gateway
curl http://localhost:8082/healthz    # job-runner

# 5. Seed initial data
cd ../..
bash scripts/seed.sh

# 6. Open console
# http://localhost:3000
# Login: admin@envnexus.io / admin123
```

**Service startup order** (Docker Compose handles this via `depends_on`):

```text
mysql → redis → minio → migration → platform-api → session-gateway → job-runner → console-web
```

**Default ports**:

| Service | Port | Notes |
|---------|------|-------|
| console-web | 3000 | Admin UI |
| platform-api | 8080 | REST API |
| session-gateway | 8081 | WebSocket |
| job-runner | 8082 | Health only |
| MySQL | 3306 | |
| Redis | 6379 | |
| MinIO API | 9000 | |
| MinIO Console | 9001 | |
| agent-core | 17700 | localhost only |

### Option 2: Kubernetes (Helm Chart)

```bash
# Add bitnami repo for MySQL/Redis dependencies
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install
helm install envnexus deploy/k8s/helm/envnexus \
  --namespace envnexus --create-namespace \
  --set env.ENX_JWT_SECRET="$(openssl rand -hex 32)" \
  --set env.ENX_DEVICE_SECRET="$(openssl rand -hex 32)" \
  --set env.ENX_SESSION_SECRET="$(openssl rand -hex 32)"
```

The Helm chart deploys: platform-api (2 replicas), session-gateway (2 replicas), job-runner (1 replica), console-web (2 replicas), plus Ingress and Secrets.

### Option 3: Local Development (No Docker for Go Services)

```bash
# Start only infrastructure
cd deploy/docker && docker compose up -d mysql redis minio && cd ../..

# Build all Go binaries
make build

# Terminal 1: platform-api
export ENX_DATABASE_DSN="root:root@tcp(localhost:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
export ENX_REDIS_ADDR="localhost:6379"
export ENX_OBJECT_STORAGE_ENDPOINT="localhost:9000"
export ENX_JWT_SECRET="dev-secret"
export ENX_DEVICE_TOKEN_SECRET="dev-device-secret"
export ENX_SESSION_TOKEN_SECRET="dev-session-secret"
./bin/platform-api

# Terminal 2: session-gateway
export ENX_SESSION_TOKEN_SECRET="dev-session-secret"
export ENX_REDIS_ADDR="localhost:6379"
./bin/session-gateway

# Terminal 3: job-runner
export ENX_DATABASE_DSN="root:root@tcp(localhost:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
./bin/job-runner

# Terminal 4: console-web
cd apps/console-web && pnpm install && pnpm dev

# Terminal 5: agent-core
./bin/enx-agent
```

### Smoke Test

```bash
bash scripts/smoke-test.sh
```

Validates: health → login → tenant → profiles → download link → enrollment → session → audit.

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build all Go binaries to `./bin/` |
| `make run-api` | Run platform-api locally |
| `make run-gateway` | Run session-gateway locally |
| `make run-runner` | Run job-runner locally |
| `make run-web` | Run console-web dev server |
| `make run-desktop` | Run agent-desktop in dev mode |

---

## Project Status

Phase 0-6 core features are implemented. See the [development roadmap](docs/development-roadmap.md) for detailed completion status per module.

| Module | Completion | Key Capabilities |
|--------|-----------|-----------------|
| Platform API | 100% | Auth, CRUD, RBAC, Webhooks, Metrics, License, Device Token Rotation |
| Session Gateway | 85% | WS relay, event dedup, session token auth |
| Job Runner | 90% | 7 workers (cleanup, build, scan, expire, flush with FS fallback) |
| Agent Core | 95% | LLM Router (7), 10 tools, governance, offline mode, SQLite |
| Console Web | 95% | 12+ pages, i18n, unified API client, error boundary |
| Agent Desktop | 85% | Tray, 5-page UI, spawn agent-core, 10 IPC channels |
| Database | 100% | 25 tables (13 core + 12 extension) |
| K8s Deployment | 90% | Helm chart with 4 services + Ingress |

## Documentation

- Product proposal: [`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)
- Development roadmap: [`docs/development-roadmap.md`](docs/development-roadmap.md)
- Commercialization plan: [`docs/commercialization-plan.md`](docs/commercialization-plan.md)

## License

See [`LICENSE`](LICENSE).
