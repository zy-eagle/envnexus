# EnvNexus Project Context & Convention Guide

## 1. Project Type
**project_type**: `legacy`

## 2. Architecture & Tech Stack
- **Monorepo Structure**: Uses Go Workspaces (`go.work`) with multiple services and apps.
  - `apps/`: Frontend and client applications (Next.js, Electron).
  - `services/`: Backend microservices in Go.
  - `libs/`: Shared libraries.
- **Backend (Go)**: 
  - Framework: Gin (`github.com/gin-gonic/gin` v1.12.0)
  - Database ORM: GORM (`gorm.io/gorm`)
  - Architecture Pattern: **Domain-Driven Design (DDD) / Clean Architecture**.
    - `domain/`: Entities, value objects, repository interfaces.
    - `repository/`: Implementations (e.g., MySQL via GORM).
    - `service/`: Application logic orchestrating domains and repos.
    - `handler/`: HTTP layer (Gin handlers).
  - Dependency Injection: Manual DI in `cmd/*/main.go`.
  - Logging: Standard library `log/slog` with JSON handler.
- **Frontend (Web)**: 
  - Framework: Next.js 14.2.3 (React 18)
  - Styling: Tailwind CSS 3.4.19
  - Language: TypeScript 5
- **Desktop Client**:
  - Framework: Electron + TypeScript
- **Database**: MySQL 8.0
- **Cache**: Redis 7
- **Object Storage**: MinIO

## 3. Coding Conventions (Strict Constraints)
- **Go Backend**:
  - **Do not put business logic in Handlers**. Handlers should only parse requests, call services, and return responses.
  - **Database Operations**: Always check `err` returned by GORM.
  - **Error Handling**: Follow existing patterns.
  - **Dependencies**: Do not introduce new web frameworks or ORMs. Stick to Gin and GORM.
- **General**:
  - Since this is marked as a `legacy` project, **DO NOT** force new conventions that contradict the existing structure.
  - If adding a new service, replicate the folder structure (`cmd`, `internal/domain`, `internal/handler`, `internal/repository`, `internal/service`) of existing services like `platform-api`.

## Stack & Layers

- `stack_type`: `fullstack`
- `frontend_framework`: `next@14.2.3` (React 18)
- `frontend_root`: `apps/console-web/`
- `frontend_patterns`: API client at `src/lib/api.ts`, i18n dictionary at `src/lib/i18n/`, App Router (`src/app/`), components at `src/components/`, PascalCase components
- `backend_framework`: `gin@1.12.0` (Go 1.25.0)
- `backend_root`: `services/platform-api/`
- `desktop_framework`: `electron@30` (TypeScript)
- `desktop_root`: `apps/agent-desktop/`
- `agent_framework`: Go (standalone binary)
- `agent_root`: `apps/agent-core/`
- `database`: `mysql@8.0`
- `cache`: `redis@7`
- `object_storage`: `minio@latest`

## 4. AI Pitfall Guide (Self-Learned Rules)

### Windows Process Lifecycle (agent-core / agent-desktop)
1. **Self-update spawn path**: After OTA self-update on Windows, the new binary path may differ from the original. The desktop shell must coordinate spawn paths via `core_install_path.json`; hardcoding the original path causes `spawn UNKNOWN` errors. *(3 fix commits: 99c2817, 32058e1, cf66946)*
2. **AV warmup on Windows**: After downloading a new agent binary, a brief "warmup" period is needed before spawning to avoid antivirus false-positive blocks.

### TypeScript Strictness (agent-desktop / console-web)
3. **Type narrowing pitfalls**: Electron IPC return types and settings loaders often produce union types that TypeScript strict mode rejects. Always use explicit type guards or `as` casts when the runtime type is guaranteed. *(TS2367, TS2322 — commits 428f1d1, 4b0a747)*
4. **ES5 target compatibility**: Avoid `Set` spread (`[...new Set()]`) when TypeScript target is ES5 — use `Array.from()` instead. *(commit 14661dd)*

### LLM Provider Quirks (agent-core)
5. **DeepSeek `reasoning_content` field**: DeepSeek may return an empty `content` field with the actual response in `reasoning_content`. The LLM router must check both fields and extract JSON from mixed prose/reasoning output. *(commits 9d58a9f, ca2e553)*
6. **Tool name format**: Some LLM APIs reject tool names containing dots. Replace dots with underscores in tool names before sending to the LLM. *(commit 070e005)*
7. **HTTP client timeout**: LLM calls can take 30-60s. Set HTTP client timeout to at least 55s; shorter timeouts cause premature cancellation. *(commits 46cdb06, 21c5c62)*

### Docker / Deployment
8. **Go workspace `replace` directives**: When building Go services in Docker, the `libs/shared/` directory must be included in the build context because `go.mod` uses `replace` directives pointing to local paths. *(commit 97b4dda)*
9. **MinIO presigned URLs**: Use a dedicated public-facing MinIO client for generating presigned download URLs; the internal client produces URLs with internal hostnames that external clients cannot resolve. *(commit e5053d0)*
10. **Electron builder in Docker**: Building Electron installers inside Docker requires careful CLI flag handling; the Docker environment may lack display servers and native dependencies. *(commit bad8b66)*

### Database / GORM
11. **Soft-delete unique indexes**: GORM soft-delete (`deleted_at IS NULL`) must be part of unique indexes; otherwise "duplicate name" errors occur when re-creating a record with the same name as a soft-deleted one. *(commit 79c3909)*
12. **Foreign key constraints**: Drop foreign keys in migrations if the application handles referential integrity in code; FK constraints cause cascading failures during bulk operations and make migration rollbacks harder. *(commit d54f408)*
13. **Domain struct JSON tags**: Always add `json:"..."` tags to domain structs that are serialized in API responses; missing tags cause incorrect field names in JSON output. *(commit a767705)*

### Frontend (console-web)
14. **i18n key deduplication**: When adding new navigation items or pages, always check existing i18n dictionaries for key conflicts before adding new keys. *(commit 2a729da)*
15. **Form state on copy**: When implementing "copy/duplicate" for a form entity, explicitly clear or reset payload fields that should not carry over (e.g., execution results, status). *(commits 13fcb95, ccb4d8f)*
16. **JWT stale data**: Do not rely solely on JWT claims for user identity in business logic (e.g., approval checks). Fetch fresh user data from DB when the operation is security-sensitive. *(commit afe1e95)*

### API Design
17. **Route path conflicts**: Avoid registering routes that shadow each other (e.g., `/devices/:id/heartbeat` vs `/devices/activate`). Use distinct prefixes or HTTP methods. *(commit 5ae50b2)*
18. **Duplicate resource errors**: Return user-friendly conflict messages (HTTP 409) instead of raw database errors when a unique constraint is violated. *(commit 48189fb)*