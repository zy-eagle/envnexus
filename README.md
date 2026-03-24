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
│  └──────────────┘     └─────┬──┬──┬───┬───┘     └──────────┬───────────┘       │
│                             │  │  │   │                     │                   │
│              ┌──────────────┘  │  │   └────────────────┐    │                   │
│              │                 │  └──────────────┐     │    │                   │
│              v                 v                  v     │    v                   │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐ │ ┌──────────┐          │
│  │  MySQL 8      │  │   Redis      │  │    MinIO     │ │ │ Redis    │          │
│  │  (主数据库)    │  │  (缓存/队列)  │  │ (对象存储)   │ │ │ (pub/sub)│          │
│  │  :3306        │  │  :6379       │  │  :9000       │ │ └──────────┘          │
│  └───────────────┘  └──────────────┘  └──────────────┘ │                       │
│                             │                           │                       │
│                             v                           v                       │
│                     ┌───────────────┐        ┌──────────────────┐              │
│                     │  job-runner   │        │  Feishu / Lark   │              │
│                     │  (Go Workers) │        │  (飞书开放平台)    │              │
│                     │  :8082        │        │  Bot + Card      │              │
│                     └───────────────┘        └──────────────────┘              │
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

---  Conversational path via Feishu (飞书) ---

0f. Operator sends "/bind dev_ABC" in Feishu group chat
    Bot binds chat → device mapping via ChatBridge
       │
       ▼
1f. Operator types "网络连不上了" in Feishu group chat
    Feishu ──webhook──> platform-api /webhook/feishu/event
    BotService creates session, gateway notifies agent-core
       │
       ▼
2f. Diagnosis progresses → EventSink pushes real-time updates:
    "🔄 采集中..." → "🧠 AI 分析中..." → [诊断结果卡片]
       │
       ▼
3f. If repair needed → [审批卡片] pushed to chat with ✅/❌ buttons
    Operator clicks [Approve] → Feishu ──POST──> /webhook/feishu/card
       │
       ▼
4f. Continues from step 8 (agent-core executes the tool)
    Tool progress + result pushed back to same Feishu chat
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

### Integrations

| Module | Target | Description |
|--------|--------|-------------|
| **Feishu (Lark) Conversational Bot** | Feishu Open Platform | Bidirectional conversational integration. Bind a Feishu group chat to a device via `/bind`, then send natural-language messages to trigger diagnosis. The system pushes real-time progress (collecting, analyzing, results), interactive approval cards (approve/reject in-chat), tool execution status, and session completion summaries. ChatBridge maps chats↔devices↔sessions with Redis-backed persistence. Only 3 env vars required. |

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

**Testing & Quality**
- **Low integration/E2E test coverage**: only unit tests are present; no CI integration tests, no Playwright/Cypress E2E for console-web, no desktop UI tests
- **No OpenAPI/Swagger documentation**: API documentation exists only as code comments; no generated spec for third-party consumers
- **No performance benchmarks**: there is no load-testing suite or profiling baseline for platform-api or session-gateway

**Enterprise Features**
- **No SSO (LDAP/SAML/OIDC)**: authentication relies solely on built-in JWT; enterprise identity providers cannot be used
- **No billing/metering integration**: SaaS payment (Stripe/Paddle) and per-tenant usage billing are planned but not built
- **No multi-language LLM prompts**: diagnosis prompts and tool descriptions are English-only; localized prompt engineering is not implemented

**Operations & Observability**
- **DB-polling job queue**: job-runner uses MySQL polling instead of Redis Streams/NATS; throughput ceiling exists at ~1000 jobs/min
- **No distributed tracing**: OpenTelemetry integration is absent; cross-service request correlation requires manual log searching
- **No Prometheus/Grafana stack**: platform-api exposes a `/readyz` but no `/metrics` endpoint; no pre-built dashboards
- **No audit log retention policy UI**: archival is automatic via job-runner, but retention rules are not configurable through the console

**Security & Compliance**
- **No penetration test report**: the platform has not undergone third-party security assessment
- **No secret rotation automation**: JWT/device/session secrets require manual rotation via environment variables

**Desktop & Agent**
- **Single-tenant agent**: agent-core connects to one platform instance at a time; multi-platform switching is not supported
- **No macOS notarization**: agent-desktop builds for macOS are unsigned; users must bypass Gatekeeper

**Ecosystem**
- **Limited IM integrations**: only Feishu (Lark) is currently supported; Slack, DingTalk, WeCom, and Microsoft Teams integrations are not yet built
- **No plugin/extension marketplace**: tool registry is compile-time only; runtime plugin loading is not supported

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
│   │   │   ├── integration/
│   │   │   │   └── feishu/      #   Feishu bot, interactive cards, event webhook, commands
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

### Feishu (Lark) Conversational Integration

EnvNexus supports conversational interaction with agents directly in Feishu group chats. After binding a chat to a device, any message sent in the chat is treated as a diagnosis request — results, approval cards, and tool execution status are pushed back in real-time.

**Setup (3 env vars only)**:

1. Create a Feishu Custom App at [Feishu Open Platform](https://open.feishu.cn/)
2. Enable **Bot** capability and **Event Subscription**
3. Set Event Request URL: `https://your-domain/webhook/feishu/event`
4. Set Card Action URL: `https://your-domain/webhook/feishu/card`
5. Subscribe to `im.message.receive_v1` event
6. Set 3 environment variables:

| Variable | Description |
|----------|-------------|
| `ENX_FEISHU_APP_ID` | Feishu app ID |
| `ENX_FEISHU_APP_SECRET` | Feishu app secret |
| `ENX_FEISHU_VERIFICATION_TOKEN` | Event callback verification token |

No chat IDs need to be configured — chats bind to devices dynamically via `/bind`.

**Conversational usage**:

```text
User:  /bind dev_01J8XYZABC
Bot:   ✅ 绑定成功！设备: dev_01J8XYZABC (my-server / linux)
       现在可以直接在群里发消息进行诊断...

User:  网络连不上了
Bot:   🚀 已向设备 dev_01J8XYZABC 发起诊断
       会话: sess_01J8XYZDEF
       诊断进行中，结果会自动推送到本群...
Bot:   🔄 诊断已启动，正在采集系统信息...
Bot:   📊 正在采集: network_config
Bot:   🧠 AI 正在分析采集的数据...
Bot:   [诊断结果卡片: 发现 DNS 配置异常，建议修复]
Bot:   [审批卡片: 工具 dns.reset | 风险 L1 | ✅批准 ❌拒绝]

User:  (clicks ✅ Approve on card)
Bot:   ✅ 审批已通过，正在执行修复...
Bot:   ⚙️ 正在执行: dns.reset ...
Bot:   ✅ 工具 dns.reset 执行成功
Bot:   [会话完成卡片]
```

**Bot commands**:

| Command | Description |
|---------|-------------|
| `/bind <device_id>` | Bind this chat to a device — enables conversational diagnosis |
| `/unbind` | Unbind the current device |
| `/who` | Show current binding info |
| `/status` | Platform health status |
| `/devices` | List registered devices (with `/bind` hint) |
| `/pending` | View pending approvals for current session |
| `/approve <id>` | Approve a repair action |
| `/deny <id> [reason]` | Deny a repair action |
| `/audit [device_id]` | Recent audit events (defaults to bound device) |

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
| Platform API | 100% | Auth, CRUD, RBAC, Webhooks, Metrics, License, Device Token Rotation, Feishu Integration |
| Session Gateway | 100% | WS relay, event dedup, session token auth, Redis pub/sub horizontal scaling |
| Job Runner | 100% | 7 workers, atomic job claiming (FOR UPDATE SKIP LOCKED), audit flush with FS fallback |
| Agent Core | 100% | LLM Router (7), 10 tools, governance, offline mode, SQLite, runtime scheduler |
| Console Web | 100% | 12+ pages, i18n, unified API client, error boundary |
| Agent Desktop | 100% | Tray, 5-page UI, spawn agent-core, 10 IPC channels, auto-update (electron-updater) |
| Feishu Integration | 100% | Conversational diagnosis via chat binding, real-time event push, approval cards, drift alerts |
| Database | 100% | 25 tables (13 core + 12 extension) |
| K8s Deployment | 100% | Helm chart with 4 services + Ingress, pinned dependency versions |

## Documentation

- Product proposal: [`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)
- Development roadmap: [`docs/development-roadmap.md`](docs/development-roadmap.md)
- Commercialization plan: [`docs/commercialization-plan.md`](docs/commercialization-plan.md)

## License

See [`LICENSE`](LICENSE).
