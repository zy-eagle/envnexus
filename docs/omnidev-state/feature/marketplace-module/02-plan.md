---
total_tasks: 11
parallel_groups: 4
critical_path: [T1 → T2 → T4 → T6 → T8]
frontend_impact: yes
---

## Group 1 (parallel — no prerequisites)
- [x] **T1** [backend] 创建市场模块和设备授权的数据库模型 (`MarketplaceItem`, `TenantSubscription`, `DeviceAuthCode`, `IdeClientToken`) · outputs: `services/platform-api/internal/domain/marketplace.go`, `services/platform-api/internal/domain/device_auth.go`
- [x] **T2** [backend] 创建市场模块和设备授权的 Repository 层 · depends: T1 · outputs: `services/platform-api/internal/repository/mysql_marketplace_repo.go`, `services/platform-api/internal/repository/mysql_device_auth_repo.go`

## Group 2 (parallel — after Group 1)
- [x] **T3** [backend] 创建设备授权服务层 (Device Flow: Init, Poll, Confirm, Refresh Token) · depends: T2 · outputs: `services/platform-api/internal/service/device_auth/service.go`
- [x] **T4** [backend] 创建市场服务层 (组件列表、组件下载、订阅、取消订阅、订阅列表) · depends: T2 · outputs: `services/platform-api/internal/service/marketplace/service.go`

## Group 3 (parallel — after Group 2)
- [x] **T5** [backend] 创建设备授权 HTTP 处理器 (`/api/v1/device-auth/*`，含 refresh 接口) 和 IDE 同步的 Access Token 鉴权中间件 (RequireIDEAuth) · depends: T3 · outputs: `services/platform-api/internal/handler/http/device_auth_handler.go`, `services/platform-api/internal/middleware/ide_auth.go`
- [x] **T6** [backend] 创建市场模块 HTTP 处理器 (含 `/api/v1/tenants/:tid/marketplace/items/:id/download` 文件下载接口) · depends: T4 · outputs: `services/platform-api/internal/handler/http/marketplace_handler.go`
- [x] **T7** [backend] 创建 IDE 同步 API 接口 (`GET /api/v1/ide-sync/manifest`)，使用 Access Token 鉴权 · depends: T4, T5 · outputs: `services/platform-api/internal/handler/http/ide_sync_handler.go`

## Group 4 (parallel — after Group 3)
- [x] **T8** [frontend] 创建设备授权确认 UI (接收 user_code 并确认授权) · depends: T5 · outputs: `apps/console-web/src/app/device-auth/confirm/page.tsx`
- [x] **T9** [frontend] 创建已授权设备管理 UI (开发者设置页面，展示并撤销 IDE Tokens) · depends: T5 · outputs: `apps/console-web/src/app/tenants/[tenantId]/developer-settings/page.tsx`
- [x] **T10** [frontend] 创建市场模块 UI (浏览与订阅，支持直接下载 `plugin` 类型的 `.vsix` 文件) · depends: T6 · outputs: `apps/console-web/src/app/tenants/[tenantId]/marketplace/page.tsx`
- [x] **T11** [frontend] 更新侧边栏/导航菜单，加入市场模块和开发者设置 · depends: T9, T10 · outputs: `apps/console-web/src/components/Sidebar.tsx`