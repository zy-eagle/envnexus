# EnvNexus Project Context & Convention Guide

## 1. Project Type
**project_type**: `legacy`

## 2. Architecture & Tech Stack
- **Monorepo Structure**: Uses Go Workspaces (`go.work`) with multiple services and apps.
  - `apps/`: Frontend and client applications (Next.js, Electron).
  - `services/`: Backend microservices in Go.
  - `libs/`: Shared libraries.
- **Backend (Go)**:
  - Framework: Gin (`github.com/gin-gonic/gin`)
  - Database ORM: GORM (`gorm.io/gorm`)
  - Architecture Pattern: **Domain-Driven Design (DDD) / Clean Architecture**.
    - `domain/`: Entities, value objects, repository interfaces.
    - `repository/`: Implementations (e.g., MySQL via GORM).
    - `service/`: Application logic orchestrating domains and repos.
    - `handler/`: HTTP layer (Gin handlers).
  - Dependency Injection: Manual DI in `cmd/*/main.go`.
  - Logging: Standard library `log/slog` with JSON handler.
- **Frontend (Web)**:
  - Framework: Next.js (React 18)
  - Styling: Tailwind CSS
- **Desktop Client**:
  - Framework: Electron + TypeScript

## 3. Coding Conventions (Strict Constraints)
- **Go Backend**:
  - **Do not put business logic in Handlers**. Handlers should only parse requests, call services, and return responses.
  - **Database Operations**: Always check `err` returned by GORM.
  - **Error Handling**: Follow existing patterns.
  - **Dependencies**: Do not introduce new web frameworks or ORMs. Stick to Gin and GORM.
- **General**:
  - Since this is marked as a `legacy` project, **DO NOT** force new conventions that contradict the existing structure.
  - If adding a new service, replicate the folder structure (`cmd`, `internal/domain`, `internal/handler`, `internal/repository`, `internal/service`) of existing services like `platform-api`.

## 4. AI Pitfall Guide (Self-Learned Rules)
*(This section will be populated automatically when `/od learn` is triggered after tasks)*
- [Pending initial learning cycle]