# Blueprint: IDE Extension & Marketplace Upload

## 1. 需求背景 (Background & Requirements)

在完成了市场模块的基础设施和设备授权 (Device Flow) 后，我们需要：
1. **开发真正的 IDE 插件客户端 (VSCode/Cursor Extension)**：该插件需要能通过 Device Flow 登录，获取 Token，并拉取市场中订阅的组件（如 MCP、Skills、Rules），将其同步到本地工作区的 `.cursor/` 目录下。
2. **支持市场组件的上传与分类管理**：允许开发者或管理员将开发好的插件（`.vsix`）、MCP 配置、Skill 描述等上传到市场，分类存放，供其他租户或用户订阅下载。

## 2. 架构设计 (Architecture Design)

### 2.1 IDE 插件 (VSCode Extension)
- **位置**: `apps/ide-extension`
- **技术栈**: TypeScript, VSCode Extension API
- **核心功能**:
  - **身份认证 (Auth)**: 注册命令 `EnvNexus: Login`。调用后端 `POST /api/v1/device-auth/init` 获取 `user_code` 和 `verification_uri_complete`。使用 `vscode.env.openExternal` 引导用户去浏览器确认。同时在后台轮询 `POST /api/v1/device-auth/poll` 直到获取 `access_token` 和 `refresh_token`。
  - **安全存储 (Storage)**: 使用 `context.secrets` (VSCode SecretStorage) 安全存储令牌，防止明文泄露。
  - **组件同步 (Sync)**: 注册命令 `EnvNexus: Sync`。携带 `access_token` 调用后端 `GET /api/v1/ide-sync/manifest`。解析返回的 JSON，根据类型将内容写入当前工作区的 `.cursor/mcp.json`、`.cursor/rules/`、`.cursor/skills/` 等文件中。
  - **打包发布 (Package)**: 使用 `vsce package` 将插件打包为 `.vsix` 文件。

### 2.2 市场组件上传与管理 (Marketplace Upload & Admin)
- **存储方案**: 使用现有的 **MinIO** 对象存储来保存上传的文件（如 `.vsix` 插件包、图标等）。
- **后端 API (`services/platform-api`)**:
  - `POST /api/v1/tenants/:tenantId/marketplace/items`: 接收 `multipart/form-data`，包含文件和元数据（名称、类型、描述、版本等），将文件上传至 MinIO，并将记录保存到 `marketplace_items` 表。
  - `PUT /api/v1/tenants/:tenantId/marketplace/items/:id`: 更新组件信息或重新上传文件。
  - `DELETE /api/v1/tenants/:tenantId/marketplace/items/:id`: 删除组件及 MinIO 中的文件。
- **前端 UI (`apps/console-web`)**:
  - 在“组件市场”页面新增“发布组件 (Publish)”按钮（仅限管理员或有权限的开发者）。
  - 创建一个表单弹窗或新页面，支持选择组件类型（Plugin, MCP, Skill, Rule, Subagent），填写基本信息，并上传文件。
  - 市场列表页增加按“分类 (Type)”过滤的 Tab 或下拉框。

## 3. 安全与体验考量 (Security & UX)
- **文件校验**: 后端在接收上传时，需校验文件类型和大小（例如 `.vsix` 限制在 50MB 以内，JSON 配置限制在 1MB 以内）。
- **下载鉴权**: 前端下载插件时，后端需生成 MinIO 的 Presigned URL，确保只有订阅了该组件的租户才能下载。
- **无缝体验**: 插件在轮询授权时，应在 VSCode 状态栏 (StatusBar) 显示“正在等待授权...”，授权成功后弹出提示框 (InformationMessage)。

## 4. 影响范围 (Impact Scope)
- 新增 `apps/ide-extension` 目录。
- 修改 `services/platform-api` 的 `marketplace` 服务，增加 MinIO 上传逻辑。
- 修改 `apps/console-web` 的市场页面，增加上传和分类过滤 UI。