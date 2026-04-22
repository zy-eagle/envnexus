---
total_tasks: 10
parallel_groups: 4
critical_path: [T1 → T2 → T4 → T8]
frontend_impact: yes
---

## Group 1 (parallel — no prerequisites)
- [x] **T1** [backend] 在 `marketplace` 服务中增加 `CreateMarketplaceItem` 和 `UpdateMarketplaceItem` 接口，支持处理 `multipart/form-data` 文件上传至 MinIO。 · outputs: `services/platform-api/internal/service/marketplace/service.go`, `services/platform-api/internal/handler/http/marketplace_handler.go`
- [x] **T2** [backend] 增加 `DeleteMarketplaceItem` 接口，删除数据库记录及 MinIO 文件。 · depends: T1 · outputs: `services/platform-api/internal/service/marketplace/service.go`, `services/platform-api/internal/handler/http/marketplace_handler.go`

## Group 2 (parallel — after Group 1)
- [x] **T3** [frontend] 在 `apps/console-web` 市场页面增加“分类过滤 (Type Filter)”的 Tabs 或下拉框。 · depends: T1 · outputs: `apps/console-web/src/app/tenants/[tenantId]/marketplace/page.tsx`
- [x] **T4** [frontend] 创建“发布组件 (Publish Component)”的表单 UI（支持选择类型、填写名称/描述/版本、上传文件）。 · depends: T1 · outputs: `apps/console-web/src/app/tenants/[tenantId]/marketplace/publish/page.tsx` 或弹窗组件。

## Group 3 (parallel — no prerequisites)
- [x] **T5** [extension] 初始化 VSCode 插件项目结构 `apps/ide-extension`，配置 `package.json`、`tsconfig.json`、`webpack.config.js` 等。 · outputs: `apps/ide-extension/package.json`, `apps/ide-extension/src/extension.ts`
- [x] **T6** [extension] 实现 Device Flow 认证逻辑（注册 `EnvNexus: Login` 命令，调用 `init`、打开浏览器、轮询 `poll`，并使用 `context.secrets` 存储 Token）。 · depends: T5 · outputs: `apps/ide-extension/src/auth.ts`
- [x] **T7** [extension] 实现 Token 刷新逻辑（在 Token 过期或 401 时调用 `POST /api/v1/device-auth/refresh`）。 · depends: T6 · outputs: `apps/ide-extension/src/auth.ts`

## Group 4 (parallel — after Group 3)
- [x] **T8** [extension] 实现组件同步逻辑（注册 `EnvNexus: Sync` 命令，调用 `GET /api/v1/ide-sync/manifest`）。 · depends: T6 · outputs: `apps/ide-extension/src/sync.ts`
- [x] **T9** [extension] 实现本地文件写入（将拉取到的 MCP 配置写入 `.cursor/mcp.json`，Skills 写入 `.cursor/skills/` 等目录）。 · depends: T8 · outputs: `apps/ide-extension/src/sync.ts`
- [x] **T10** [extension] 编写打包脚本，使用 `vsce package` 生成 `.vsix` 文件，并提供 README 说明。 · depends: T9 · outputs: `apps/ide-extension/README.md`, `apps/ide-extension/package.json`