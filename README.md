# EnvNexus

[中文版本](README.zh-CN.md)

EnvNexus is an AI-native platform for environment governance, secure local diagnosis, and guided repair. It combines a multi-tenant platform, a desktop client, and a local execution core to deliver tenant-specific distribution, policy-driven diagnosis, approval-based repair, and end-to-end auditability.

## Project Status

This repository is currently in the specification-first stage.

- The primary source of truth is [`docs/envnexus-proposal.md`](docs/envnexus-proposal.md).
- The implementation scaffold is planned but not fully generated yet.
- The deployment and runtime model below describes the target operating model defined by the proposal.

## What EnvNexus Solves

EnvNexus is designed for scenarios where users or operators need to diagnose and repair environment issues without exposing unrestricted remote control.

Typical use cases:

- Self-service diagnosis and guided repair for end users
- Remote support with approval-based local actions
- Enterprise endpoint governance with tenant-specific policies
- Hybrid or private deployments that require local model or key control

Core principles:

- Default read-only, diagnose before repair
- All write actions require explicit approval
- Local policy overrides cloud-side suggestions
- Full audit trail with rollback and accountability
- Platform orchestrates, local agent executes safely

## Product Overview

EnvNexus is composed of three major layers:

- Platform layer: tenant management, model profiles, policy profiles, download links, device registration, audit query, and package metadata
- Endpoint layer: local diagnosis, approval-based repair, audit collection, and secure execution
- Integration layer: `WebSocket`, `Webhook`, and private-network integration entry points

The MVP target is to close the full loop between one platform host and one managed device:

`login -> tenant configuration -> signed download link -> activation -> device registration -> read-only diagnosis -> approval-based low-risk repair -> audit reporting`

## Target Architecture

The proposal fixes the following technology choices for the first deliverable:

- Web console: `Next.js + TypeScript`
- Backend services: `Go`
- Service boundary: `platform-api`, `session-gateway`, `job-runner`
- Desktop shell: `Electron + React + TypeScript`
- Local execution core: `Go agent-core`
- Main database: `MySQL 8`
- Cache and short-lived state: `Redis`
- Object storage: `MinIO` (S3-compatible)
- Local agent state: `SQLite + Files`
- First deployment model: `Docker Compose`

High-level runtime topology:

```text
Admin Browser
    |
    v
console-web
    |
    v
platform-api
    +--> MySQL
    +--> Redis
    +--> MinIO
    +--> session-gateway
    \--> job-runner

agent-desktop
    |
    v
agent-core (localhost API)
    +--> platform-api
    +--> session-gateway
    \--> SQLite + Files
```

## Planned Repository Layout

The proposal defines a monorepo layout for implementation:

```text
envnexus/
  apps/
    console-web/
    agent-desktop/
    agent-core/
  services/
    platform-api/
    session-gateway/
    job-runner/
  libs/
  deploy/
    docker/
  docs/
```

At the moment, this repository mainly contains the proposal document and legal/project metadata.

## Core MVP Capabilities

The first implementation is expected to include:

- Console login and tenant configuration
- `ModelProfile`, `PolicyProfile`, and `AgentProfile` management
- Tenant-specific signed download links
- Device activation, configuration pull, and heartbeat
- Local UI plus `WebSocket` session flow
- Read-only diagnostic tools
- A small set of approval-based low-risk repair actions
- Audit event reporting and querying
- Single-host deployment with `Docker Compose`

## Deployment Modes

The proposal defines three operating modes:

- `Hosted`: platform-managed control plane and storage
- `Hybrid`: shared platform control plane with enterprise-side key/model boundaries
- `Private`: customer-managed full deployment using the same protocol and object model

The first delivery target is a single-host MVP deployment:

- One Linux host runs the platform stack
- One managed endpoint runs `agent-core` and `agent-desktop`
- Platform services are started with `Docker Compose`

## Planned Deployment Guide

These steps describe the target deployment model from the proposal. They document how the project is intended to be deployed once the scaffold is generated.

### Platform Services

Required platform components:

- `console-web`
- `platform-api`
- `session-gateway`
- `job-runner`
- `mysql`
- `redis`
- `minio`

Expected startup order:

1. `mysql`
2. `redis`
3. `minio`
4. migration job
5. `platform-api`
6. `session-gateway`
7. `job-runner`
8. `console-web`

Expected public/default ports:

- `console-web`: `3000`
- `platform-api`: `8080`
- `session-gateway`: `8081`
- `mysql`: `3306`
- `redis`: `6379`
- `minio-api`: `9000`
- `minio-console`: `9001`
- `agent-core local api`: `127.0.0.1:17700`

Platform deployment requirements:

- All service configuration must come from environment variables or `env_file`
- All container logs must go to stdout
- Persistent data must be mounted to host volumes
- Readiness must depend on migration and upstream dependencies

## Planned Runtime Flow

Once implemented, the expected runtime flow is:

1. Start platform services with `Docker Compose`
2. Complete initial console login and tenant setup
3. Configure model, policy, and agent profiles
4. Generate a tenant-specific signed download link
5. Download and start the desktop agent on the managed device
6. Activate the device and pull remote configuration
7. Start a diagnosis session from local UI or remote session entry
8. Require explicit approval before any write action
9. Report audit events and store the full execution trail

## Observability And Operations

The proposal requires first-class observability for both platform and endpoint flows:

- Structured logs with request, tenant, device, session, trace, approval, and job identifiers
- Metrics for API traffic, device activation success, approval flow, tool execution, and audit delivery
- Trace or equivalent correlation for key end-to-end flows
- Audit records for approvals, execution, rollback, distribution, and exports
- Diagnostic bundle export from the desktop side with sensitive data redacted

Initial non-functional targets defined by the proposal:

- `1000` registered devices per single-instance platform
- `200` concurrently online devices
- `50` active sessions
- Audit online query retention: `180` days
- Audit archive retention: `1` year
- Standard package build target: within `5` minutes
- `RTO <= 4h`
- `RPO <= 15m`

## Security Model

EnvNexus is not designed as unrestricted remote control software.

Mandatory guardrails include:

- No arbitrary shell exposure in the default product shape
- Structured tool execution only
- Policy evaluation before execution
- Approval gate before high-risk actions
- Append-only audit semantics
- Tenant isolation across database, cache, tasks, and object storage

## Documentation

- Main proposal: [`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)

## License

See [`LICENSE`](LICENSE).
