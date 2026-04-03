# 开发计划: Agent Loop 架构 + 远程命令下发审批系统

## 任务总览

共 13 个模块，分两大阶段。

---

## 阶段一: Agent Loop (ReAct) 架构 + 安全加固 (已完成 ✅)

### M1: Tool 接口扩展 & Schema 定义
- [x] M1.1~M1.4 (全部完成)

### M2: LLM Router 协议层扩展
- [x] M2.1~M2.4 (全部完成)

### M3: LLM Provider 适配
- [x] M3.1~M3.6 (全部完成)

### M4: Agent Loop 引擎
- [x] M4.1~M4.6 (全部完成)

### M5: API Server 新端点
- [x] M5.1~M5.4 (全部完成)

### M6: Desktop 前端适配
- [x] M6.1~M6.7 (全部完成)

### M7: 安全加固 & Bug 修复
- [x] M7.1~M7.9 (全部完成)

---

## 阶段二: 远程命令下发审批系统 (已完成 ✅)

> 设计文档: `docs/omnidev-state/main/04-design.md` (v2.3)
> 规格文档: `docs/command-approval-spec.md` (v1.2)
>
> 核心理念: **运维下发命令 → 上级审批（控制台 + IM） → 设备执行**
> - 审批人在**控制台**的"待审批"页面或 **IM 卡片**中审批
> - agent-desktop 不参与远程命令审批（移除"待审批"页面）
> - 客户端本地对话确认保持不变

### M8: 数据基础 — 新增模型 & Migration ✅
> 依赖: 无 | 优先级: P0

- [x] M8.1 新增 `domain/command_task.go` — CommandTask 模型
- [x] M8.2 新增 `domain/command_execution.go` — CommandExecution 模型
- [x] M8.3 新增 `domain/approval_policy.go` — ApprovalPolicy 模型
- [x] M8.4 新增 `domain/im_provider.go` — IMProvider 模型
- [x] M8.5 新增 `domain/user_notification_channel.go` — UserNotificationChannel 模型
- [x] M8.6 Repository 实现 (GORM) — 5 个 Repository (接口 + MySQL 实现)
- [x] M8.7 SQL Migration 脚本 — `000011_command_approval.up.sql`
- [x] M8.8 新增 `infrastructure/crypto.go` — AES-256-GCM 加密/解密

### M9: 命令任务核心服务 ✅
> 依赖: M8 | 优先级: P0

- [x] M9.1 `service/command/command_service.go` — 创建/审批/拒绝/取消/执行
- [x] M9.2 `service/command/risk_evaluator.go` — 风险等级评估 (系统检测 + EffectiveRisk)
- [x] M9.3 `service/command/approval_policy_service.go` — 审批策略 CRUD + 匹配
- [x] M9.4 `handler/http/command_task_handler.go` — 命令任务 REST API
- [x] M9.5 `handler/http/approval_policy_handler.go` — 审批策略 REST API
- [x] M9.6 路由注册 + 权限校验 (main.go wiring)
- [x] M9.7 DTO 定义 (command_task, approval_policy, im_provider)

### M10: 通知路由 — IM 审批卡片 ✅
> 依赖: M8 | 优先级: P0

- [x] M10.1 `service/notification/types.go` — Notifier 接口 + 通知类型
- [x] M10.2 `service/notification/router.go` — NotificationRouter 路由核心
- [x] M10.3 `handler/http/im_provider_handler.go` — IM 配置 + 用户通知渠道 API
- [ ] M10.4 `service/notification/feishu_notifier.go` — 飞书 Notifier (重构现有)
- [ ] M10.5 飞书卡片回调扩展 — 支持命令任务审批

### M11: agent-core 远程命令执行 ✅
> 依赖: M9 | 优先级: P0

- [x] M11.1 WebSocket 新增 `command.execute` 事件处理 (agent-core `ws_client.go`)
- [x] M11.2 命令执行器 — shell 类型 (跨平台 Windows/Linux)
- [x] M11.3 WebSocket 新增 `command.result` 事件回报 (agent-core → session-gateway → platform-api)
- [x] M11.4 Session-gateway 新增 `command.result` 转发 + device event endpoint
- [x] M11.5 Platform 侧接收结果 → 更新 CommandExecution → 更新 Task 状态

### M12: 控制台 UI + Desktop 清理 ✅
> 依赖: M9, M10 | 优先级: P1

- [x] M12.1 侧边栏新增 "命令任务"、"待审批"、"审批策略" 入口 + 图标
- [x] M12.2 命令任务列表页 (状态筛选 + 创建模态框 + 详情查看)
- [x] M12.3 待审批页面 — 审批人登录控制台，审批/拒绝 + 角标
- [x] M12.4 审批策略管理页 (CRUD + 审批规则/职责分离配置)
- [x] M12.5 i18n 字典更新 (中/英) — nav + commandTasks + pendingApprovals + approvalPolicies
- [x] M12.6 **agent-desktop 移除"待审批"页面** — 侧边栏、页面、仪表盘卡片、IPC、JS 全部清理

### M13: 飞书 Bot 扩展 + 更多 IM 渠道 (待开发)
> 依赖: M10, M11 | 优先级: P2

- [ ] M13.1 飞书 Bot: `/exec`, `/exec-batch`, `/tasks`, `/task` 命令
- [ ] M13.2 飞书 Bot: `/bindme`, `/verify` 自助绑定
- [ ] M13.3 审批卡片扩展 (命令任务审批 — 显示申请人/命令/设备/风险)
- [ ] M13.4 企业微信集成 `integration/wechat_work/`
- [ ] M13.5 钉钉集成 `integration/dingtalk/`

---

## 环境变量变更

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `ENX_ENCRYPTION_KEY` | IM 凭据加密密钥 (32 bytes hex) | deploy.sh 自动生成 |

## 数据库变更

| 操作 | 表 |
|------|-----|
| CREATE TABLE | `command_tasks` |
| CREATE TABLE | `command_executions` |
| CREATE TABLE | `approval_policies` |
| CREATE TABLE | `im_providers` |
| CREATE TABLE | `user_notification_channels` |
