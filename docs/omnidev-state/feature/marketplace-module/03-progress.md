---
current_phase: 3
active_group: 4
---

## State Snapshot

### Group 3 (已完成)
- [x] **T5** [backend] 创建设备授权 HTTP 处理器 (`/api/v1/device-auth/*`，含 refresh 接口) 和 IDE 同步的 Access Token 鉴权中间件 (RequireIDEAuth)
- [x] **T6** [backend] 创建市场模块 HTTP 处理器 (含 `/api/v1/tenants/:tid/marketplace/items/:id/download` 文件下载接口)
- [x] **T7** [backend] 创建 IDE 同步 API 接口 (`GET /api/v1/ide-sync/manifest`)，使用 Access Token 鉴权

### Group 4 (进行中)
- [ ] **T8** [frontend] 创建设备授权确认 UI (接收 user_code 并确认授权)
- [ ] **T9** [frontend] 创建已授权设备管理 UI (开发者设置页面，展示并撤销 IDE Tokens)
- [ ] **T10** [frontend] 创建市场模块 UI (浏览与订阅，支持直接下载 `plugin` 类型的 `.vsix` 文件)
- [ ] **T11** [frontend] 更新侧边栏/导航菜单，加入市场模块和开发者设置

### 阻塞与问题 (Blockers & Issues)
- 无

### 下一步 (Next Action)
- 实现 T8, T9, T10, T11：在 `apps/console-web/src/app/` 目录下创建对应的前端页面，并更新 `Sidebar.tsx`。