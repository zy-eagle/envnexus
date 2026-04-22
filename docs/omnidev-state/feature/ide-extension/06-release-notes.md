# Release Notes: IDE Extension & Marketplace Upload

## 🚀 New Features

### 1. VSCode / Cursor Extension (`apps/ide-extension`)
- **Device Authorization Flow**: Users can now authenticate their IDE using the `EnvNexus: Login` command. The extension securely stores the resulting access and refresh tokens in VSCode's encrypted `SecretStorage`.
- **Component Synchronization**: The `EnvNexus: Sync` command fetches the user's subscribed marketplace items and automatically writes them to the local workspace:
  - MCP configurations are merged into `.cursor/mcp.json`.
  - Skills are written to `.cursor/skills/<name>/SKILL.md`.
  - Rules are written to `.cursor/rules/<name>.mdc`.
- **Packaging**: The extension is ready to be packaged into a `.vsix` file using `npm run package`.

### 2. Marketplace Component Upload (Backend & Frontend)
- **MinIO Integration**: Platform Super Admins can now upload plugin files (`.vsix`) directly to the marketplace. The backend securely stores these artifacts in MinIO.
- **Publish UI**: Added a "Publish Component" form in the Console Web (`apps/console-web`). It supports uploading files for plugins or pasting raw JSON payloads for skills, rules, and MCPs.
- **Type Filtering**: The Marketplace list view now includes tabs to filter items by type (Plugin, MCP, Skill, Rule, Subagent).

## 🔒 Security Enhancements
- **Secure Token Storage**: IDE tokens are no longer stored in plaintext files; they leverage the OS-level keychain via VSCode API.
- **Path Traversal Prevention**: The extension sanitizes component names before writing them to the `.cursor/` directory.
- **Upload Validation**: The backend validates file uploads and gracefully handles MinIO unavailability (returns 503).

## 🛠️ Technical Details
- Added `POST /api/v1/tenants/:tenantId/marketplace/items` for `multipart/form-data` uploads.
- Added `PUT` and `DELETE` endpoints for marketplace item management.
- Extended `api.ts` in the frontend to support `FormData` requests without explicit `Content-Type` headers.