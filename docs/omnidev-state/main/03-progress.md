---
status: completed
current_step: 62
failed_tests: 0
last_updated: "2026-04-18 16:30:00"
---

## State Snapshot

- **Currently doing**: 全部里程碑 M1-M6 已完成（含第二轮差距补齐）。项目功能开发结束，进入集成测试和部署阶段。
- **Completed**:
  - **M1 (T1-T12)**: 修复计划引擎 — types、DAG、planner、snapshot、executor、policy CheckPlan、API 端点、loop 集成、Desktop SSE/IPC、单元测试 + 回归测试
  - **M2 (T13-T17)**: 智能诊断升级 — 复杂度评估、分层证据收集、迭代推理、诊断→计划联动（NeedsRemediation）、回归测试
  - **M3 (T18-T32)**: Watchlist 主动巡检 — types、store（SQLite CRUD）、evaluator（5 种条件类型）、decomposer（LLM 自然语言→WatchItems）、scheduler（定时触发）、builtin rules（9 条规则）、alerter（→ 修复闭环）、manager、governance engine 集成、API 端点（7 条路由）、bootstrap 装配、Desktop Watchlist 页面 + NL 输入 + 健康大盘 + IPC、单元测试（22）+ 回归测试（8）
  - **M4 (T33-T40, T61)**: 远程文件取证 — file_download 工具（敏感路径阻断、MinIO 预签名上传）、文件访问 API（browse/preview/download）、platform-api 文件访问 domain/service/repo/handler、session-gateway 文件事件、console-web 文件浏览器页面、migration 000016、文件取证单元测试（11）
  - **M5 (T41-T50, T62)**: 多模态 & 批量操作 — LLM 多模态支持（ContentPart、VisionProvider 接口）、OpenAI Vision provider、Desktop 截图粘贴/拖拽上传（T44）、Router.CompleteMultimodal、/local/v1/chat 多模态短路路径、设备组 domain/repo/service/handler、**批量任务真实下发（T46，BatchService：分批 + 成功率门槛 + 批次间延迟 + 取消）**、console-web 设备组 + 批量任务页面、migration 000017、多模态测试（5）+ 批量测试（4）
  - **M6 (T51-T60, T62)**: 平台增强 — 健康评分聚合服务、治理规则引擎（CRUD、condition/action）、工具权限（按租户/角色）、**agent 平台规则拉取（T53，PlatformSync：定时轮询 + ruleToWatchItem 翻译 + source=platform 回写 + stale 清理）**、治理规则 HTTP handler、console-web 健康大盘 + 治理规则 + 工具权限页面、**策略 Profile 增强（T57：工具白/黑名单 + 路径访问控制 UI 编辑器）**、migration 000018、规则测试（5）+ 工具权限测试（2）+ platform_sync 测试（4）
- **Blockers/Issues**: 无
- **Next Action**: 计划文档/进度文档已全部刷新；所有里程碑真实完成；后续进入 QA / 部署阶段。

## History Summary

- [2026-04-03] 完成远程命令审批模块 (M8-M12)，覆盖后端服务、数据库迁移、API handler、WebSocket 事件、前端页面、桌面端清理；所有 Go 服务编译通过。
- [2026-04-10] 完成 M1（修复计划引擎）+ M2（智能诊断升级）。17 个任务、29 个测试通过。
- [2026-04-10] 完成 M3（Watchlist 主动巡检）。15 个任务（T18-T32），30 个新测试通过。新增包：`governance/watchlist/`（types、store、evaluator、decomposer、scheduler、builtin_rules、alerter、manager）。修改：`governance/engine.go`（WatchlistManager 集成）、`api/server.go`（watchlist 路由）、`bootstrap/bootstrap.go`（watchlist 装配）、`store/store.go`（DB() 访问器）。Desktop：watchlist 页面、NL 添加关注点、健康大盘卡片、IPC handler。
- [2026-04-18] 完成 M4（远程文件取证）+ M5（多模态 & 批量操作）+ M6（DevSecOps 加固）框架层。30 个任务（T33-T60）、3 个 SQL migration 初版完成。所有 Go 服务编译通过。
- [2026-04-18 PM] **补齐第二轮实现缺口**：
  - T44：Desktop 图片粘贴/拖拽上传（`chat-attachments` UI、`addImageBlob`、`sendChatRequest` 改造为 `[]ContentPart`）；`api/server.go` `ChatRequest.Content` 改为 `json.RawMessage`，新增 `extractTextAndParts` + 多模态短路调用 `Router.CompleteMultimodal`；`router.Router.CompleteMultimodal` 按 VisionProvider 顺序选型。
  - T46：`services/platform-api/internal/service/command/batch_service.go` — 真实分批下发执行器（batch_size、SuccessRateThreshold、InterBatchDelay、Cancel 控制、`waitForBatch` 按 executions 终态聚合），`device_group_handler.CreateBatchTask` 装配 + `POST /batch-tasks/:id/cancel`，`main.go` 装配 BatchService；单元测试 4 个。
  - T53：`apps/agent-core/internal/governance/watchlist/platform_sync.go` — 定时拉取 `/agent/v1/governance/sync`，`ruleToWatchItem` 翻译 + `source=platform` 自动回写 + stale 清理 + `ToolPermissions()` 暴露给策略引擎；`bootstrap.go` 启动；单元测试 4 个。
  - T57：`console-web/policy-profiles/page.tsx` — `PolicyFields` 扩展 `tool_whitelist / tool_blacklist / allowed_paths / denied_paths`，新增 `ListEditor` 组件，模态框增加"Tool Permissions"和"File Path Access Control"两节；modal 改为 max-w-2xl + 滚动。
  - 迁移文件改名避免版本冲突：`000005/6/7_* → 000016/17/18_*`。
  - `docs/omnidev-state/main/02-plan.md` 从历史乱码版本重写为干净 UTF-8。