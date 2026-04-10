# EnvNexus

[中文版本](README.zh-CN.md)

**EnvNexus** is a **self-healing local environment platform** for enterprise endpoints. It covers common local environment issues for both knowledge workers and developers, including VPN, proxy, DNS, certificate, disk, database, Docker, port, and runtime dependency failures, and replaces unrestricted remote shell access with a structured flow: **AI diagnoses → generates a remediation plan → humans approve → the local agent executes with rollback**. Users describe problems in natural language or paste error screenshots, and the agent handles diagnosis, planning, remediation, and verification. It also supports **controlled remote file system access**, allowing approved browsing of endpoint files, text log previews, and downloads of original files in any format, while **proactively detecting** local environment risks through watchlists and built-in health rules.

---

## The Problem

Local environment issues on enterprise endpoints are frequent, repetitive, and directly disruptive to work, yet the usual response is still either remote takeover by IT or a ticket queue followed by manual troubleshooting. One is risky, the other is slow.

EnvNexus takes a different approach built specifically for local endpoint environment issues:

- **Focused on enterprise endpoint users** — covers local environment issues across both office users and developers, especially network access, proxies, certificates, disk, services, databases, containers, and runtime dependencies
- **Default read-only** — the agent only runs diagnostic tools unless a write action is explicitly approved
- **Plan-based repair** — diagnosis produces a structured remediation plan (ordered DAG with risk levels, rollback strategies, and verification steps); users approve the entire plan, not individual commands
- **Layered approval** — L0 auto-pass, L1 plan-level, L2 plan + confirm, L3 per-step approval; configurable via policy profiles
- **Controlled file access and download** — after approval, support teams can browse the remote endpoint file system, preview text logs, and download original files of any format for troubleshooting and evidence collection
- **Smart watchlist** — users describe what to monitor in natural language ("watch VPN, proxy, disk usage, MySQL, and Docker"); the LLM decomposes this into structured checks and the agent continuously patrols
- **Proactive discovery** — built-in rule packs + platform-pushed policies + rules learned from past fixes help surface issues before they block work
- **Full audit trail** — every session, diagnosis, plan, approval, execution, and verification is recorded and queryable
- **Local-first execution** — the AI engine runs on the endpoint; the platform orchestrates but never executes directly

### What EnvNexus Is NOT

- Not a remote desktop or arbitrary shell tool
- Not an RMM agent that can run any command
- Not a broad all-in-one endpoint management suite
- Does not bypass local policy — the local agent always has the final say

---

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                     PLATFORM SIDE                            │
│                  (Docker Compose / K8s)                       │
│                                                              │
│   console-web ──> platform-api ──> session-gateway           │
│   (Next.js 14)    (Go / Gin)       (Go / WebSocket)         │
│      :3000          :8080             :8081                   │
│                       │                                      │
│              ┌────────┼────────┐                             │
│           MySQL    Redis    MinIO        job-runner           │
│                                          (7 workers)         │
│                                                              │
│                    Feishu / Lark Bot (conversational)         │
└──────────────────────────┬───────────────────────────────────┘
                           │ HTTPS / WSS
┌──────────────────────────┼───────────────────────────────────┐
│                    ENDPOINT SIDE                              │
│                                                              │
│   agent-desktop (Electron 30) ──IPC──> agent-core (Go)       │
│   - System tray, Chat UI,              - LLM Router (7)      │
│     Plan Approval, Watchlist,          - 33+ Structured Tools │
│     Health Dashboard                   - Diagnosis Engine     │
│                                        - Remediation Planner  │
│                                        - Watchlist Engine     │
│                                        - Governance Engine    │
│                                        - SQLite local store   │
│                                        - OTA Self-Updater     │
└──────────────────────────────────────────────────────────────┘
```

**Platform side**: multi-tenant control plane with admin console, REST API, WebSocket gateway, and background job workers. Backed by MySQL, Redis, and MinIO.

**Endpoint side**: a Go execution core (`agent-core`) running locally on managed devices, paired with an Electron desktop shell. The core handles AI diagnosis (7 LLM providers), remediation plan generation, 33+ structured tools, policy enforcement, watchlist-based proactive monitoring, governance (baseline + drift detection), and offline degraded mode.

**Integrations**: Feishu (Lark) conversational bot — bind a group chat to a device, then diagnose via natural language with real-time progress and in-chat approval cards.

---

## Key Design Decisions

| Area | Approach |
|------|----------|
| **Security** | No arbitrary shell; structured tools only; all writes require approval; 4-layer JWT token system; AES-256-GCM local config encryption |
| **Multi-tenancy** | Database, cache, storage, and audit are all tenant-scoped |
| **AI Integration** | LLM Router with 7 providers (OpenAI, Anthropic, DeepSeek, Gemini, OpenRouter, Ollama, local); structured output; tool-calling pattern |
| **Resilience** | Agent degrades gracefully offline (read-only tools + local SQLite); MinIO falls back to local filesystem |
| **Access Control** | RBAC with 5 preset roles and 17 permissions; rate limiting per IP |
| **Deployment** | Docker Compose, Helm chart, or bare-metal; private deployment ready (no cloud dependency) |

---

## Quick Start

**Prerequisites**: Docker 20.10+, Docker Compose v2

```bash
git clone https://github.com/zy-eagle/envnexus.git
cd envnexus
./deploy.sh start
```

The script auto-detects host IP, generates secrets, computes source-code hashes to rebuild only changed services, and starts everything.

**Default access**: `http://localhost:3000` — Login: `admin@envnexus.io` / `admin123`

See [Deployment](#deployment-options) for manual Docker Compose, Kubernetes Helm, and local development setups.

---

## Deployment Options

| Method | Command | Best For |
|--------|---------|----------|
| **Smart Deploy** (recommended) | `./deploy.sh start` | One-command setup with change detection |
| **Full Rebuild** | `./deploy.sh full` | Force rebuild all services |
| **Docker Compose** | `cd deploy/docker && docker compose up -d` | Manual control |
| **Kubernetes** | `helm install envnexus deploy/k8s/helm/envnexus ...` | Production K8s clusters |
| **Local Dev** | `make build` + run binaries | Development without Docker for Go services |

Default ports: console-web `:3000`, platform-api `:8080`, session-gateway `:8081`, job-runner `:8082`, agent-core `:17700` (localhost only).

---

## Repository Layout

```text
envnexus/
├── apps/
│   ├── console-web/        # Next.js 14 admin console
│   ├── agent-desktop/      # Electron 30 desktop shell
│   └── agent-core/         # Go local execution core
├── services/
│   ├── platform-api/       # Go central API (Gin + GORM, DDD)
│   ├── session-gateway/    # Go WebSocket gateway
│   └── job-runner/         # Go background workers
├── libs/shared/            # Shared Go library
├── deploy/
│   ├── docker/             # Docker Compose deployment
│   └── k8s/helm/envnexus/ # Kubernetes Helm chart
├── scripts/                # Smoke test, seed data
├── docs/                   # User manual, product whitepaper
└── Makefile
```

---

## Project Status

All core modules are feature-complete (Phase 0–6 implemented):

| Module | Status | Highlights |
|--------|--------|------------|
| Platform API | Complete | Auth, RBAC, Webhooks, Metrics, License, Feishu Integration |
| Session Gateway | Complete | WS relay, event dedup, Redis pub/sub scaling |
| Job Runner | Complete | 7 workers, atomic job claiming, audit archival |
| Agent Core | Complete | 7 LLM providers, 33+ tools, diagnosis engine, governance, offline mode, OTA update |
| Console Web | Complete | 12+ pages, i18n (zh/en), unified API client |
| Agent Desktop | Complete | Tray, chat UI, approvals, auto-update |
| Feishu Integration | Complete | Conversational diagnosis, real-time push, approval cards |
| Infrastructure | Complete | 25 DB tables, Helm chart, Docker Compose |

### Known Gaps

- **Testing**: Low integration/E2E test coverage; no OpenAPI spec; no performance benchmarks
- **Enterprise**: No SSO (LDAP/SAML/OIDC); no billing/metering; English-only LLM prompts
- **Observability**: No distributed tracing (OpenTelemetry); no Prometheus/Grafana; DB-polling job queue
- **Ecosystem**: Only Feishu integration (no Slack/DingTalk/WeCom/Teams); no runtime plugin loading

---

## Documentation

- **User Manual**: [`docs/user-manual.md`](docs/user-manual.md) — end-user and admin operations guide
- **Product Whitepaper**: [`docs/product-manual.md`](docs/product-manual.md) — commercial positioning and solutions

## License

See [`LICENSE`](LICENSE).
