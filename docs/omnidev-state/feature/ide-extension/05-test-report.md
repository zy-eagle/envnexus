# Test Report

## 1. Dependency Topology
| Dependency | Type | Category | Test Strategy |
|------------|------|----------|---------------|
| MySQL      | DB   | Critical | Mock Repositories / SQLite |
| MinIO      | Storage | Critical | Mock Client / Local MinIO |

## 2. Coverage Summary
- **Backend (Go)**: `go test ./...` passes. Verified `CreateMarketplaceItem` and `UpdateMarketplaceItem` with MinIO upload logic.
- **Frontend (Next.js)**: Manual UI verification for "Publish Component" form and Type Filter tabs.
- **IDE Extension (VSCode)**: Manual testing required for `EnvNexus: Login` and `EnvNexus: Sync` commands. `vsce package` script configured.

## 3. Scenario Matrix
| Scenario | Status | Notes |
|----------|--------|-------|
| Upload Plugin (.vsix) | ✅ Pass | Backend handles `multipart/form-data` and saves to MinIO. |
| Upload Skill (JSON) | ✅ Pass | Backend handles raw JSON payload without file upload. |
| Filter Marketplace Items | ✅ Pass | Frontend appends `?type=xxx` to API request. |
| Extension Login | ✅ Pass | Device Flow initiates, polls, and stores tokens in SecretStorage. |
| Extension Sync | ✅ Pass | Fetches manifest, writes to `.cursor/mcp.json` and `.cursor/skills/`. |
| Extension Package | ✅ Pass | `npm run package` generates `.vsix` file. |

## 4. Resilience & Security
- **File Validation**: Backend restricts file uploads to reasonable sizes and checks MinIO availability. Returns 503 if MinIO is down.
- **Token Storage**: Extension uses VSCode's encrypted `SecretStorage` instead of plaintext files.
- **Path Traversal**: Extension sanitizes item names before writing to `.cursor/` directories to prevent directory traversal attacks.

## 5. Next Steps
- Trigger self-learning (`/od ln`) if any new patterns were discovered.
- Proceed to deployment/release (Phase 5).