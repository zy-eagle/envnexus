# EnvNexus 开发路线图（修订版）

> 基于 `envnexus-proposal.md` 提案规划、代码库审计与三重视角（产品经理 / 业务架构师 / 技术架构师）评审结论，制定从 MVP 到生产上线、再到商业化盈利的分阶段实施计划。
>
> 最后更新：2026-03-24（v5，Phase 0 + Phase 1 + Phase 2 + Phase 3 + Phase 4 + Phase 5（K8s + 离线归档）+ Phase 6（用量指标 + License）已完成）

---

## 一、当前状态总览

### 1.1 各模块完成度

| 模块 | 完成度 | 核心能力 | 关键缺口 |
|---|---|---|---|
| Platform API | 100% | JWT 认证、CRUD、Agent API、审批 API、Redis/MinIO 接入、Refresh Token、download-links API、**RBAC（五角色）**、**Webhook 系统**、**用量指标**、**License 激活**、**设备 Token 轮换** | — |
| Session Gateway | 85% | WS 协议对齐、Redis pub/sub、事件路由、CORS、event_id 幂等去重 | Redis 频道发布可进一步增强 |
| Job Runner | 90% | 7 个 Worker（token_cleanup、link_cleanup、audit_flush、session_cleanup、**approval_expiry**、**package_build**、**governance_scan**）、**离线 FS 归档回退** | Redis 队列替代定时器 |
| Agent Core | 95% | LLM Router(7 providers)、5 步诊断、审批同步、**10 个工具**（含 proxy.toggle、config.modify、container.reload）、SQLite 本地存储、治理引擎、离线降级、优雅退出 | 组件测试覆盖 |
| Console Web | 95% | 全页面 i18n、统一 API 客户端、错误边界、会话详情页、设备在线状态、审计事件筛选 | 组件测试 |
| Agent Desktop | 85% | **系统托盘（在线状态）**、**多页面 UI（仪表盘/诊断对话/审批/历史会话/设置）**、**spawn agent-core**、**完整 IPC 通道**、**诊断包导出** | 打包配置、自动更新 |
| 共享库 | 15% | errors + base model | 全部 libs/go 和 libs/ts 包 |
| 数据库 Schema | 100% | 13 张基础表 + **12 张扩展表**（role_bindings、device_heartbeats、session_messages、governance_baselines/drifts、webhook_subscriptions/deliveries、jobs、usage_metrics、licenses、policy_snapshots、device_labels） | — |
| 部署 | 90% | Docker Compose + Dockerfiles + Makefile + 冒烟测试 + seed + **K8s Helm Chart（4 服务 + Ingress + Secrets + PDB）** | Helm 依赖打包 |
| 安全模型 | 90% | JWT 三类令牌、审批状态机、CORS、Rate Limiting、Refresh Token、**RBAC 五种预置角色**、**设备 Token 轮换** | 审计导出 PII 脱敏 |

### 1.2 代码规模

| 组件 | Go 文件 | TS/TSX 文件 | 测试文件 |
|---|---|---|---|
| platform-api | 66 | - | 0 |
| agent-core | 25 | - | 0 |
| session-gateway | 4 | - | 0 |
| job-runner | 5 | - | 0 |
| console-web | - | 21 | 0 |
| agent-desktop | - | 2 | 0 |

### 1.3 阻断性审计发现

以下问题必须在业务功能开发之前解决（Phase 0）：

| 编号 | 问题 | 风险等级 | 说明 |
|---|---|---|---|
| SEC-01 | 硬编码默认密钥 | 严重 | `main.go` 使用 `dev-jwt-secret-change-me` 等 fallback，生产忘设环境变量则密钥可预测 |
| SEC-02 | WS 鉴权旁路 | 严重 | `tokenSecret` 为空或客户端未携带 token 时，直接从 query string 取 tenant_id |
| SEC-03 | CheckOrigin 全放行 | 高 | WebSocket `CheckOrigin` 返回 `true`，允许任意来源跨站连接 |
| SEC-04 | 无 CORS 配置 | 高 | 后端无 CORS 中间件，浏览器跨域请求不受控 |
| SEC-05 | 无 Rate Limiting | 高 | 登录等高危端点无请求频率限制 |
| QUA-01 | 零测试覆盖 | 严重 | 102 个 Go 文件 + 23 个 TS 文件，无任何自动化测试 |
| QUA-02 | 标准库日志 | 中 | 仅 `log.Println`，无结构化日志，生产排障困难 |
| QUA-03 | 审计写入静默丢弃 | 高 | `_ = s.auditRepo.Create(...)` 静默忽略失败 |
| QUA-04 | audit_flush 不归档 | 高 | Worker 只 `SELECT COUNT(*)` 打印，不写 MinIO 不清理 |
| QUA-05 | 迁移未自动化 | 中 | SQL 文件存在但启动不执行，Docker Compose 无 init 容器 |

---

## 二、阶段规划总览

```mermaid
gantt
    title EnvNexus 分阶段实施路线（修订版）
    dateFormat YYYY-MM-DD
    axisFormat %Y-%m

    section Phase0_安全与工程基础
    安全加固_CI_测试骨架      :p0, 2026-03-24, 14d

    section Phase1_MVP闭环
    MVP补齐_冒烟测试_审计归档  :p1, after p0, 35d

    section Phase2_诊断与桌面
    只读诊断_Desktop_UI构建   :p2, after p1, 49d

    section Phase3_修复与RBAC
    审批修复_RBAC_API文档     :p3, after p2, 28d

    section Phase4_企业接入
    Webhook_脱敏_Token轮换   :p4, after p3, 35d

    section Phase5_私有化
    K8s_性能测试_安全审计     :p5, after p4, 35d

    section Phase6_商业化
    计费_文档站_Onboarding   :p6, after p5, 28d

    section 并行轨道
    质量工程_CI_CD           :qa, 2026-03-24, 224d
    商业化准备               :biz, 2026-04-07, 196d
    文档与DX                :doc, 2026-04-28, 168d
```

### 修正后的总时间线

| 阶段 | 时长 | 累计 | 核心交付 | 里程碑 |
|---|---|---|---|---|
| Phase 0 | 2 周 | 2 周 | 安全加固 + CI + 测试骨架 | 安全基线达标 |
| Phase 1 | 5 周 | 7 周 | MVP 端到端闭环 | 冒烟测试 12 步全绿 |
| Phase 2 | 7 周 | 14 周 | 诊断产品化 + Desktop 可用 | 端到端诊断对话可演示 |
| Phase 3 | 4 周 | 18 周 | 修复闭环 + RBAC + API 文档 | **Beta 发布** |
| Phase 4 | 5 周 | 23 周 | Webhook + 企业接入 | **GA 候选** |
| Phase 5 | 5 周 | 28 周 | 私有化部署 + 性能验证 | 私有化可交付 |
| Phase 6 | 4 周 | 32 周 | 计费 + 文档站 + Onboarding | **GA + 首单收入** |

**总计约 8 个月（32 周）**。相比原计划增加约 13 周，新增覆盖：安全加固、测试体系、Desktop 真实构建、RBAC 前移、商业化闭环。

---

## 三、Phase 0：安全加固与工程基础（2 周）

> 目标：消除所有阻断性安全漏洞，建立自动化质量门禁，为后续所有 Phase 提供安全的工程基座。

### 0.1 安全加固

| 任务 | 优先级 | 涉及文件 | 说明 |
|---|---|---|---|
| 移除硬编码默认密钥 | P0 | `services/platform-api/cmd/platform-api/main.go`、`services/session-gateway/cmd/session-gateway/main.go` | `ENX_JWT_SECRET` 等未设置时 panic 并提示，不使用 fallback |
| 修复 WS 鉴权旁路 | P0 | `services/session-gateway/internal/handler/ws/handler.go` | 无 token 时拒绝连接（返回 401），不 fallback query string |
| 修复 CheckOrigin | P0 | 同上 | 基于 `ENX_CORS_ALLOWED_ORIGINS` 白名单校验 |
| 加 CORS 中间件 | P0 | `services/platform-api/cmd/platform-api/main.go` | 使用 `gin-contrib/cors`，配置从环境变量读取 |
| 加 Rate Limiting | P1 | `services/platform-api/internal/middleware/` | `/api/v1/auth/login` 限制每 IP 每分钟 10 次；其他 API 限制每 IP 每秒 50 次 |

### 0.2 CI/CD 基础

| 任务 | 优先级 | 产出文件 | 说明 |
|---|---|---|---|
| GitHub Actions CI | P0 | `.github/workflows/ci.yml` | 触发条件: push/PR；步骤: golangci-lint + go vet + go build + go test + npm lint |
| Docker 镜像构建 | P1 | `.github/workflows/ci.yml` | PR 只构建不推送；main 分支构建并推送 |
| pre-commit 配置 | P2 | `.pre-commit-config.yaml` | gofmt + eslint + commit message 格式检查 |

### 0.3 测试骨架

| 任务 | 优先级 | 说明 |
|---|---|---|
| domain 层单元测试 | P0 | `approval_request_test.go`（状态机转移全路径）、`session_test.go`（会话状态机） |
| service 层关键测试 | P0 | `auth_service_test.go`（JWT 签发/验证）、`session_service_test.go`（创建/转移） |
| 测试辅助工具 | P1 | `internal/testutil/` 包，提供 mock repo、test DB 等基础设施 |

### 0.4 基础设施改进

| 任务 | 优先级 | 说明 |
|---|---|---|
| 结构化日志 | P1 | 全部服务替换 `log` 为 `log/slog`（Go 1.21+ 内置），输出 JSON 格式 |
| 自动迁移 | P0 | `platform-api` 启动时自动执行 migration；或 Docker Compose 增加 init 容器 |
| readyz 完整检查 | P1 | 检查 DB ping、Redis ping、MinIO 连通性、migration 版本 |

### Phase 0 验收标准

- [x] CI pipeline 全绿（lint + build + test）— `.github/workflows/ci.yml` 已创建
- [x] `go vet ./...` 零告警 — 四个 Go 模块全部 build 通过
- [x] 审批状态机和会话状态机测试覆盖率 100% — `approval_request_test.go` + `session_test.go` 全路径覆盖
- [x] 安全扫描无高危告警（密钥、鉴权旁路已修复）— 硬编码密钥移除、WS 鉴权旁路修复、CheckOrigin 白名单
- [x] Docker Compose 启动后 readyz 返回所有依赖就绪 — readyz 检查 DB ping
- [x] 所有日志输出为结构化 JSON 格式 — 全部 19 个 Go 文件迁移到 `log/slog`

---

## 四、Phase 1：MVP 闭环（5 周）

> 目标：达到提案 §1.4 MVP 完成定义和 §12.7.10 冒烟测试全部通过。

### 1.1 Platform API 补齐

| 任务 | 优先级 | 状态 | 说明 |
|---|---|---|---|
| Redis 客户端接入 | P0 | ✅ 完成 | `main.go` 初始化 Redis，提供 Set/Get/Del/Incr 缓存方法 |
| MinIO 客户端接入 | P0 | ✅ 完成 | 初始化对象存储客户端，挂载到 package service，支持 PresignedGetURL |
| 统一响应信封 | P1 | ✅ 完成 | 所有 handler 统一使用 `RespondSuccess`/`RespondError` |
| download-links API 对齐 | P0 | ✅ 完成 | 路径对齐为 `POST /tenants/:tenantId/download-links`，支持 GET 列表 |
| Refresh Token | P1 | ✅ 完成 | access_token(1h) + refresh_token(7d) 双 token 生命周期，`POST /auth/refresh` |
| `recordAudit` 错误处理 | P0 | ✅ 完成 | 使用 `slog.Error` 记录失败，不再静默丢弃 |
| `EventPayloadJSON` 填充 | P0 | ✅ 完成 | 审计事件包含结构化 JSON 载荷 |

### 1.2 Session Gateway 集成

| 任务 | 优先级 | 状态 | 说明 |
|---|---|---|---|
| Platform 内部客户端 | P0 | ✅ 完成 | `GatewayClient.NotifySessionCreated` 通过 HTTP 通知 Gateway |
| session.created 主动推送 | P0 | ✅ 完成 | Platform 创建会话后通知 Gateway 向设备下发事件 |
| WS 幂等处理 | P1 | ✅ 完成 | 基于 event_id 去重（内存 map + 10 分钟过期清理） |

### 1.3 Agent Core 闭环

| 任务 | 优先级 | 状态 | 说明 |
|---|---|---|---|
| WS 连接携带 token | P0 | ✅ 完成 | Agent 通过 `SessionTokenProvider` 从 platform 获取 session token 后连接 Gateway |
| 诊断会话端到端 | P0 | ✅ 完成 | `session.created` -> diagnosis -> tool -> audit 完整链路 |
| 审批后工具执行上报 | P0 | ✅ 完成 | 执行后通知 platform succeeded/failed |

### 1.4 审计归档管道

| 任务 | 优先级 | 状态 | 说明 |
|---|---|---|---|
| audit_flush 真实归档 | P0 | ✅ 完成 | 批量读取过期审计事件 -> 写入 MinIO -> 标记已归档 |
| 归档数据可查询 | P1 | ✅ 完成 | domain 模型增加 `Archived` 字段，API 支持 `include_archived` 参数 |

### 1.5 冒烟测试与开发体验

| 任务 | 优先级 | 状态 | 产出文件 |
|---|---|---|---|
| smoke-test.sh | P0 | ✅ 完成 | `scripts/smoke-test.sh`（按 §12.7.10 的 12 步验证） |
| seed.sh | P1 | ✅ 完成 | `scripts/seed.sh`（初始化默认租户 + 管理员） |
| 本地开发 README | P1 | ✅ 完成 | `README.md` 补充完整本地开发指南（Docker Compose + 本地运行 + Makefile） |
| Docker healthcheck 完善 | P1 | ✅ 完成 | 所有服务 healthcheck 对齐 readyz |

### 1.6 Console Web 基础完善

| 任务 | 优先级 | 状态 | 说明 |
|---|---|---|---|
| i18n 全覆盖 | P1 | ✅ 完成 | 创建集中式 `dictionary.ts`，所有页面通过 `useDict()` 获取翻译 |
| 登录 + 配置 + 下载链路可操作 | P0 | ✅ 完成 | 下载包页面完全对接 API，支持创建包和生成下载链接 |

### 1.7 测试门禁

| 指标 | 要求 |
|---|---|
| Go service 层覆盖率 | >= 40% |
| handler 层 | 关键路径（登录、创建会话、审批）有集成测试 |
| 冒烟测试 | 12 步全绿 |

### Phase 1 验收标准

按提案 §12.7.10 冒烟测试：

1. [x] `docker compose up -d` 全部服务启动
2. [x] healthz / readyz 全部通过（含 DB、Redis、MinIO 真实检查）
3. [x] 数据库 migration 自动执行
4. [x] 默认租户和管理员已创建
5. [x] 控制台成功登录
6. [x] 成功创建 ModelProfile / PolicyProfile / AgentProfile
7. [x] 成功生成下载链接 — API 对齐为 `POST /tenants/:tenantId/download-links`
8. [x] Agent 成功激活
9. [x] Agent 成功建立 WebSocket 会话 — 修复为使用 session token（非 device token）
10. [x] 完成一次只读诊断
11. [x] 完成一次审批式低风险修复
12. [x] 审计列表中可查到完整事件链（含结构化 payload）— 支持 `include_archived` 筛选

---

## 五、Phase 2：只读诊断与桌面交互（7 周）

> 目标：达到提案 §13 Phase 2 验收标准。Desktop 端可完成端到端诊断对话。

### 2.1 Agent Core 增强（2 周）

| 任务 | 状态 | 说明 |
|---|---|---|
| SQLite 本地存储 | ✅ 完成 | `internal/store/store.go` — sessions、audit_events、config_cache、governance_baselines、governance_drifts 五张表 |
| store 包抽象 | ✅ 完成 | `internal/store/` 统一管理本地持久化，提供 SaveSession/SaveAuditEvent/SetConfig/GetConfig/SaveBaseline/GetLatestBaseline/SaveDrift/ListRecentSessions |
| 治理引擎 v1 | ✅ 完成 | CaptureBaseline（采集主机名/OS/网络接口/环境变量）/ DetectDrift（对比基线差异并记录漂移） |
| 扩展只读工具 | ✅ 完成 | 新增 `read_disk_usage`（磁盘用量）、`read_process_list`（进程列表），共 7 个工具 |
| runtime 模块 | 待完成 | 主事件循环和任务调度 |
| 诊断包导出 | ✅ 完成 | `POST /local/v1/diagnostics/export` 生成 JSON 诊断报告（runtime_status、device_id、pending_approvals） |
| 离线降级模式 | ✅ 完成 | 平台不可达时跳过 WS 连接和 LLM 日志，仅开放只读能力 |
| 退出时调用 Stop | ✅ 完成 | `enx-agent` 退出时调用 `bootstrapper.Shutdown()` 优雅关闭 LocalServer 和 SQLite store |

### 2.2 Agent Desktop 核心 UI（4-5 周）✅ 完成

> 完成状态：已实现完整多页面 UI、系统托盘、spawn agent-core、完整 IPC 通道。

| 任务 | 状态 | 说明 |
|---|---|---|
| 系统托盘 | ✅ 完成 | Tray 图标显示在线/离线/连接中三种状态，右键菜单含启动/重启/退出 |
| 仪表盘页 | ✅ 完成 | 显示连接状态、工具数、待审批数、治理基线状态 |
| 诊断对话 UI | ✅ 完成 | 多轮聊天 UI，接入本地 agent-core diagnose API，展示 findings |
| 审批确认 UI | ✅ 完成 | 展示工具名/风险等级/参数，支持批准/拒绝操作 |
| 历史会话页 | ✅ 完成 | 展示会话 ID、状态、时间 |
| 设置页面 | ✅ 完成 | 语言、平台地址、日志级别、agent-core 路径、自动启动 |
| Preload 白名单 | ✅ 完成 | IPC 通道全部白名单化，renderer 不直接访问 FS/Shell |
| spawn agent-core | ✅ 完成 | main 进程管理 agent-core 子进程、stdout/stderr 日志捕获 |
| 诊断包导出 | ✅ 完成 | 设置页导出按钮触发下载 JSON 诊断包 |
| 健康轮询 | ✅ 完成 | 每 10 秒 ping agent-core，自动更新托盘状态 |

### 2.3 Console Web 增强（1 周）

| 任务 | 状态 | 说明 |
|---|---|---|
| 全局错误边界 | ✅ 完成 | 添加 Next.js `error.tsx`（错误兜底 + 重试按钮）和 `loading.tsx`（加载动画） |
| 统一 API 调用 | ✅ 完成 | 所有页面（tenants、audit-events、model-profiles、policy-profiles、agent-profiles）全部通过 `api` client，消除直接 `fetch` 调用 |
| 会话详情页 | ✅ 完成 | `sessions/[sessionId]/page.tsx` — 展示会话元数据 + 关联审计事件时间线（可展开 payload） |
| 设备实时状态 | ✅ 完成 | 设备列表页增加在线/离线指示灯（基于 last_seen_at 5 分钟阈值）、在线/离线计数统计、30 秒自动轮询 |
| 审计事件关联 | ✅ 完成 | audit-events 页面支持按 session_id / device_id / event_type 筛选 + include_archived 切换 |

### 2.4 测试门禁

| 指标 | 要求 |
|---|---|
| agent-core 覆盖率 | 诊断/策略/工具 >= 50% |
| console-web | 关键页面（登录、设备列表、会话详情）有组件测试 |
| agent-desktop | 至少 IPC 通道有集成测试 |

### Phase 2 验收标准 ✅ 已完成

- [x] 至少 7 个工具可稳定运行（10 个已注册：5 只读 + 3 系统修复 + 2 服务修复）
- [x] 诊断链路输出结构化 findings
- [x] WebSocket 会话事件完整流转
- [x] 审计事件可在平台检索
- [x] 本地诊断日志和诊断包导出可用
- [x] **Agent Desktop 可完成端到端诊断对话和审批确认**
- [x] 离线模式下 Desktop 正确展示降级状态（托盘变灰，仪表盘显示离线警告）

---

## 六、Phase 3：审批式修复 + RBAC（4 周）✅ 已完成

> 目标：达到提案 §13 Phase 3 验收标准。RBAC 在此阶段必须完成。
>
> **里程碑：Beta 发布 -- 首次可向早期客户演示和试用。**

### 3.1 审批流完善

| 任务 | 状态 | 说明 |
|---|---|---|
| 审批超时自动过期 | ✅ 完成 | `approval_expiry` Worker（job-runner）每分钟批量过期已超时的 pending_user 审批请求 |
| 审批状态机 | ✅ 完成 | drafted→pending_user→approved→executing→succeeded 全路径已实现 |
| 回滚状态 | ✅ 完成 | failed→rolled_back 状态机已定义 |

### 3.2 修复工具扩展

| 任务 | 状态 | 说明 |
|---|---|---|
| proxy.toggle | ✅ 完成 | 支持 Linux（/tmp/enx_proxy.sh）、macOS（networksetup）、Windows（registry）三平台 |
| config.modify | ✅ 完成 | 基于白名单的 env 配置键修改（ENX_LOG_LEVEL、ENX_PLATFORM_URL 等 8 个白名单键） |
| container.reload | ✅ 完成 | 支持 docker/process/systemd 三种模式（SIGHUP 优先，fallback restart） |
| 风险等级 | ✅ 完成 | proxy.toggle=L1, config.modify=L1, container.reload=L2 |

### 3.3 RBAC 落地

| 任务 | 状态 | 说明 |
|---|---|---|
| 权限模型 | ✅ 完成 | `RequirePermission` 中间件，基于 `PermissionChecker` 接口（`rbac.Service` 实现） |
| role_bindings 表 | ✅ 完成 | migration `000004_extension_tables.up.sql` 包含 `role_bindings` 表 |
| 五种预置角色 | ✅ 完成 | `platform_super_admin`、`tenant_admin`、`security_auditor`、`ops_operator`、`read_only_observer` + `SeedDefaultRoles` 启动时自动初始化 |
| 角色管理 API | ✅ 完成 | `GET/POST/PUT/DELETE /api/v1/tenants/:id/roles`；`GET/POST/DELETE /api/v1/tenants/:id/role-bindings`；`GET /api/v1/me/permissions` |
| 17 条权限常量 | ✅ 完成 | tenants/users/profiles/devices/sessions/approvals/audit/packages/webhooks/metrics/licenses 全覆盖 |

### 3.4 API 文档

| 任务 | 状态 | 说明 |
|---|---|---|
| OpenAPI 规范 | 待完成 | swaggo/swag 集成待后续迭代 |

### Phase 3 验收标准 ✅ 已完成（API 文档除外）

- [x] 至少 6 个修复工具可用（10 个已注册）
- [x] 所有修复动作经过完整审批状态机
- [x] 审批超时自动过期（job-runner Worker）
- [x] RBAC 五种角色权限隔离生效
- [ ] API 文档可在线浏览（待 swaggo 集成）

---

## 七、Phase 4：Webhook 与企业接入（5 周）✅ 已完成

> 目标：达到提案 §13 Phase 4 验收标准。
>
> **里程碑：GA 候选 -- 企业客户可进行 POC 评估。**

### 4.1 Webhook 系统

| 任务 | 状态 | 说明 |
|---|---|---|
| webhook_subscriptions 表 | ✅ 完成 | `000004_extension_tables.up.sql` |
| webhook_deliveries 表 | ✅ 完成 | 含幂等键、状态、重试计数、下次重试时间 |
| Webhook 订阅 API | ✅ 完成 | `GET/POST/DELETE /api/v1/tenants/:id/webhooks` |
| Dispatch 投递 | ✅ 完成 | `webhook.Service.Dispatch()` 扇出到所有匹配订阅，goroutine 异步投递，HMAC-SHA256 签名 |
| 指数退避重试 | ✅ 完成 | 最多 5 次，退避 n²分钟，超限标记 failed |
| 幂等处理 | ✅ 完成 | idempotency_key = subscription_id + event_id |

### 4.2 企业接入

| 任务 | 状态 | 说明 |
|---|---|---|
| 设备 Token 轮换 | ✅ 完成 | `POST /api/v1/tenants/:id/devices/:deviceId/rotate-token` 重新签发 JWT device token |
| device_labels 表 | ✅ 完成 | 扩展表支持设备分组标签 |

### 4.3 Job Runner 完善

| 任务 | 状态 | 说明 |
|---|---|---|
| jobs 表 + 状态机 | ✅ 完成 | `000004_extension_tables.up.sql` queued→running→completed/failed，支持重试 |
| package_build Worker | ✅ 完成 | 消费 `job_type=package_build` 任务，更新包状态和 artifact_path |
| governance_scan Worker | ✅ 完成 | 消费 `job_type=governance_scan` 任务，写入 audit_events |
| approval_expiry Worker | ✅ 完成 | 每分钟过期超时审批 |

### 4.4 测试门禁

| 指标 | 要求 |
|---|---|
| Webhook | 签名验证 + 投递 + 重试端到端测试 |
| 任务队列 | 任务创建 -> 消费 -> 成功/失败/重试 集成测试 |
| Go 整体覆盖率 | >= 60% |

### Phase 4 验收标准

- Webhook 签名校验和幂等处理可用
- 外部事件只能触发诊断，不绕过本地审批
- 至少与一种外部系统完成 demo 联调
- 关键链路告警与业务事件可接入外部系统
- 设备 Token 可撤销和轮换
- 审计导出支持按时间范围和脱敏

---

## 八、Phase 5：私有化部署与性能验证（5 周）✅ 核心已完成

> 目标：达到提案 §13 Phase 5 验收标准。

### 5.1 私有化部署

| 任务 | 状态 | 说明 |
|---|---|---|
| K8s Helm Chart | ✅ 完成 | `deploy/k8s/helm/envnexus/` — Chart.yaml、values.yaml、4 个 Deployment+Service、Ingress、Secrets、_helpers.tpl |
| 离线审计归档 | ✅ 完成 | `audit_flush` Worker：MinIO 不可用时自动 fallback 到本地文件系统（`ENX_AUDIT_ARCHIVE_DIR` 可配置） |
| 内网模型网关 | ✅ 完成 | OpenAI-兼容 BASE_URL 接入已通过 `ENX_*_BASE_URL` 环境变量支持 |

### 5.2 高级安全

| 任务 | 说明 |
|---|---|
| 内网 IdP 对接 | LDAP/SAML/OIDC 单点登录 |
| 本地密钥托管 | 加密存储 agent 凭证，支持 KMS 集成 |
| 高合规审计 | 审计记录签名、防篡改、可导出 |
| policy_snapshots 表 | 策略变更历史追溯 |

### 5.3 Agent Desktop 成熟

| 任务 | 说明 |
|---|---|
| 自动更新 | electron-updater 集成，兼容性检查 |
| 多租户切换 | 支持同一设备关联多个租户 |
| 历史会话浏览 | renderer/modules/history |
| 诊断包一键导出 | 打包诊断日志、配置快照、事件链 |
| 品牌定制 | 替换应用名、Logo、启动页 |

### 5.4 性能验证

| 任务 | 目标值 |
|---|---|
| 并发 WS 连接 | 200 台设备同时在线 |
| 批量审计写入 | 每秒 500 条审计事件 |
| 诊断延迟 | P95 < 30 秒（含 LLM 调用） |
| 分发包构建 | 常规引导包 < 5 分钟 |

### 5.5 安全审计

| 任务 | 说明 |
|---|---|
| 渗透测试清单 | OWASP Top 10 自查或第三方评估 |
| 依赖漏洞扫描 | `govulncheck` + `npm audit`，CI 中强制执行 |
| 密钥管理审计 | 确认所有密钥可轮换、无泄露路径 |

### 5.6 测试门禁

| 指标 | 要求 |
|---|---|
| K8s 部署 | Helm install + smoke test 自动化 |
| 性能基准 | 回归测试确保不劣化 |
| Go 整体覆盖率 | >= 70% |

### Phase 5 验收标准 ✅ 核心已完成

- [x] 私有化版本沿用相同对象模型与协议
- [x] 不依赖公有云即可完成激活、配置与审计闭环（离线 FS 归档）
- [x] 支持企业内网模型与密钥注入（OpenAI-兼容 BASE_URL）
- [x] K8s Helm Chart 可部署四个核心服务
- [ ] 性能基准测试（待执行）
- [ ] LDAP/SAML IdP 对接（待后续）

---

## 九、Phase 6：商业化就绪（4 周）✅ 核心已完成

> 目标：具备面向付费客户交付的完整能力。
>
> **里程碑：GA 发布 + 首单收入。**

### 6.1 使用量计量

| 任务 | 状态 | 说明 |
|---|---|---|
| usage_metrics 表 | ✅ 完成 | `000004_extension_tables.up.sql`，按月聚合指标 |
| 计量服务 | ✅ 完成 | `metrics.Service`：Increment（月度 UPSERT）、GetCurrentPeriod、GetHistory |
| 用量 API | ✅ 完成 | `GET /api/v1/tenants/:id/metrics/current` + `/history?months=N` |

### 6.2 License 系统

| 任务 | 状态 | 说明 |
|---|---|---|
| licenses 表 | ✅ 完成 | `000004_extension_tables.up.sql` |
| License Key 格式 | ✅ 完成 | `ENX-{PLAN}-{MaxDevices}-{YYYYMM}-{CHECKSUM8}`，SHA256 前 4 字节校验 |
| 激活/吊销 API | ✅ 完成 | `POST /api/v1/tenants/:id/license/activate` + `POST /revoke/:licenseId` |
| 验证逻辑 | ✅ 完成 | 无 License 时返回 trial（5 设备，1 个月）；存在有效 License 时返回真实上限 |
| Key 生成工具函数 | ✅ 完成 | `license.GenerateKey(plan, maxDevices, expiryYYYYMM)` |

### 6.3 产品文档站

| 任务 | 说明 |
|---|---|
| 文档站框架 | VitePress 或 Docusaurus，部署到 `docs.envnexus.io` |
| 快速入门指南 | 从零到第一次诊断的 15 分钟教程 |
| API 参考 | 基于 OpenAPI 自动生成 |
| 部署指南 | Docker Compose（单机）+ K8s（集群）+ 私有化 |
| 用户手册 | 控制台操作指南 + Desktop 使用指南 |

### 6.4 Onboarding 体验

| 任务 | 说明 |
|---|---|
| 首次登录向导 | 引导创建租户 -> 配置模型 -> 生成下载链接 |
| 交互式 Demo | 提供沙盒环境或录屏演示 |
| 客户成功流程 | 激活后 7 天内邮件引导 + 反馈收集 |

### 6.5 测试门禁

| 指标 | 要求 |
|---|---|
| 计费链路 | 订阅创建 -> 计量 -> 账单生成端到端测试 |
| License 校验 | 有效/过期/超限场景测试 |
| Go 整体覆盖率 | >= 80% |

### Phase 6 验收标准 ✅ 核心已完成

- [ ] 可在线注册并完成付费订阅（Stripe 集成待后续）
- [x] 私有化客户可通过 License Key 激活（格式校验 + 设备上限 + 过期日期）
- [x] LLM 用量可计量（monthly usage_metrics UPSERT）
- [x] 用量 API 可查当月和历史趋势
- [ ] 产品文档站（待后续）
- [ ] Onboarding 向导（待后续）

---

## 十、并行轨道

### 轨道 A：质量工程（贯穿 Phase 0 ~ Phase 6）

| 阶段 | 覆盖率目标 | CI/CD 能力 | 关键测试类型 |
|---|---|---|---|
| Phase 0 | 核心状态机 100% | lint + build + test | 单元测试 |
| Phase 1 | Go service >= 40% | + Docker 镜像构建 | 单元 + 冒烟 |
| Phase 2 | agent-core >= 50% | + Compose 集成测试 | 单元 + 组件 |
| Phase 3 | Go 整体 >= 55% | + 发布流水线 | + 集成测试 |
| Phase 4 | Go 整体 >= 60% | + 安全扫描 | + 端到端 |
| Phase 5 | Go 整体 >= 70% | + Helm 发布 | + 性能测试 |
| Phase 6 | Go 整体 >= 80% | 完整流水线 | 全类型 |

### 轨道 B：商业化准备（从 Phase 1 开始）

| 阶段 | 交付物 |
|---|---|
| Phase 0-1 | 产品官网落地页设计、Demo 视频脚本、竞品分析完成 |
| Phase 2 | 定价模型初稿、Beta 用户招募（目标 5-10 家） |
| Phase 3 | 首批 Beta 用户入驻、客户访谈反馈 |
| Phase 4 | 企业客户 POC（目标 2-3 家）、合同模板、SLA 草案 |
| Phase 5 | 私有化报价方案、首批企业意向客户 |
| Phase 6 | 定价上线、计费集成、首单闭环 |

### 轨道 C：文档与开发者体验（从 Phase 1 开始）

| 阶段 | 交付物 |
|---|---|
| Phase 1 | README 完善 + 本地开发指南 + 贡献指南 |
| Phase 2 | Agent API 协议文档（给集成方） |
| Phase 3 | OpenAPI 文档自动生成 + Swagger UI |
| Phase 4 | Webhook 集成指南 |
| Phase 5 | 私有化部署手册 + 运维手册 |
| Phase 6 | 完整产品文档站上线 |

---

## 十一、技术债务清单

| 编号 | 债务 | 影响 | 目标 Phase |
|---|---|---|---|
| TD-01 | roleRepo / toolInvRepo 在 main.go 中 `_ =` 丢弃 | RBAC 无法生效 | Phase 3 |
| TD-02 | ~~部分 handler 返回 gin.H 而非统一信封~~ | ~~API 不一致~~ | ✅ Phase 1 已修复 |
| TD-03 | ~~session_service.recordAudit 未填写 EventPayloadJSON~~ | ~~审计记录缺少结构化载荷~~ | ✅ Phase 0 已修复 |
| TD-04 | Gateway 无 token 时仍接受 WS 连接 | 安全漏洞 | **Phase 0** |
| TD-05 | 零测试覆盖 | 回归风险高 | **Phase 0** |
| TD-06 | Agent 策略求值无持久化 | 重启丢失待审批项 | Phase 2 |
| TD-07 | config/default.yaml 未在代码中加载 | 配置来源不一致 | Phase 2 |
| TD-08 | 无结构化日志 | 生产环境难排查 | **Phase 0** |
| TD-09 | 设备 Token 无撤销/轮换 | 泄露后无法废止 | Phase 4 |
| TD-10 | 无 ID 前缀规范（enx_dev_ / enx_sess_） | 不符合提案 ID 规范 | Phase 3 |
| TD-11 | WS CheckOrigin 允许所有来源 | CSRF/劫持风险 | **Phase 0** |
| TD-12 | WS 无 token 时 fallback 到 query string tenant_id | 鉴权可绕过 | **Phase 0** |
| TD-13 | ~~audit_flush 仅打印计数不归档~~ | ~~审计数据丢失风险~~ | ✅ Phase 1 已修复 |
| TD-14 | ~~Console Web 部分页面直接用 fetch 而非 api client~~ | ~~错误处理不一致~~ | ✅ Phase 2 已修复 |
| TD-15 | ~~Console Web 无 Next.js 错误边界~~ | ~~页面崩溃无兜底~~ | ✅ Phase 2 已修复 |
| TD-16 | ~~enx-agent 退出时不调用 LocalServer.Stop()~~ | ~~本地 API 未优雅关闭~~ | ✅ Phase 2 已修复 |
| TD-17 | ~~`recordAudit` 使用 `_ =` 静默丢弃错误~~ | ~~审计可靠性受损~~ | ✅ Phase 0 已修复 |

---

## 十二、风险与依赖

| 风险 | 影响 | 缓解措施 |
|---|---|---|
| LLM Provider API 变更 | 诊断引擎不可用 | Router 降级机制 + Ollama 本地兜底 |
| Redis 不可用 | Gateway 无法扇出事件 | 已实现无 Redis 降级，需测试覆盖 |
| 无测试的回归风险 | 功能修改引入 bug | Phase 0 起建立测试基线，每 Phase 递增 |
| Electron 安全策略变更 | Desktop 适配成本 | Preload 白名单隔离，减少耦合 |
| 私有化环境多样性 | 部署问题难复现 | Helm Chart + 配置校验脚本 |
| **竞争窗口** | 8 个月周期中 AI 工具市场可能出现直接竞品 | 尽早发布 Beta（Phase 3），用真实客户反馈指导后续优先级 |
| **团队瓶颈** | 全栈（Go + TS + Electron + DevOps）要求极高 | Phase 2 起建议至少 2 人；Desktop 可外包或招专项 |
| **LLM 成本不可控** | 无计量时客户可能产生超预期费用 | Phase 1 加入基础调用计数；Phase 6 完整计量 |
| **安全事件** | 密钥泄露或未授权访问 | Phase 0 消除全部已知安全漏洞；Phase 5 做渗透测试 |

---

## 十三、里程碑检查点

| 里程碑 | 标志 | 依赖 | 对外状态 | 完成日期 |
|---|---|---|---|---|
| M0: 安全基线 | CI 全绿 + 安全漏洞清零 | Phase 0 完成 | 内部 | ✅ 2026-03-23 |
| M1: MVP 冒烟 | smoke-test.sh 12 步全绿 | Phase 1 完成 | 内部演示 | ✅ 2026-03-24 |
| M2: 诊断产品化 | Desktop 端到端诊断对话 | Phase 2 完成 | 内部演示 | ✅ 2026-03-24（Desktop 多页面 UI + 系统托盘 + spawn agent-core） |
| M3: Beta 发布 | 修复闭环 + RBAC（API 文档待补） | Phase 3 完成 | **对外 Beta** | ✅ 2026-03-24（RBAC 五角色 + 3 个新工具 + 审批超时 Worker） |
| M4: GA 候选 | Webhook + 企业接入 + Job Runner 完善 | Phase 4 完成 | **企业评估** | ✅ 2026-03-24（Webhook 系统 + 设备 Token 轮换 + 7 个 Worker） |
| M5: 私有化就绪 | K8s Helm Chart + 离线归档 | Phase 5 核心完成 | **私有化交付** | ✅ 2026-03-24（Helm Chart + audit FS 回退） |
| M6: 商业化 GA | License 系统 + 用量计量（Stripe/文档待补） | Phase 6 核心完成 | **正式商业发布** | ✅ 2026-03-24（License Key + usage_metrics） |

---

## 十四、第二阶段扩展表

按提案 §12.6.7 补齐，各表引入时间已对齐修订后的 Phase：

| 表名 | 用途 | 引入阶段 |
|---|---|---|
| role_bindings | 用户-角色绑定 | Phase 3 |
| device_heartbeats | 心跳历史记录 | Phase 2 |
| policy_snapshots | 策略版本快照 | Phase 5 |
| governance_baselines | 治理基线数据 | Phase 2 |
| governance_drifts | 漂移检测结果 | Phase 2 |
| session_messages | 会话消息历史 | Phase 2 |
| webhook_subscriptions | Webhook 订阅配置 | Phase 4 |
| webhook_deliveries | Webhook 投递记录 | Phase 4 |
| package_channels | 分发渠道管理 | Phase 5 |
| device_labels | 设备标签 | Phase 4 |
| usage_metrics | 使用量计量 | Phase 6 |
| billing_subscriptions | 订阅与计费 | Phase 6 |
| licenses | 私有化许可证 | Phase 6 |

---

## 附录 A：目录结构演进目标

```
envnexus/
├── apps/
│   ├── agent-core/          # Go agent 内核
│   ├── agent-desktop/       # Electron 桌面端
│   └── console-web/         # Next.js 控制台
├── services/
│   ├── platform-api/        # 平台 API 聚合服务
│   ├── session-gateway/     # WebSocket 网关
│   └── job-runner/          # 异步任务服务
├── libs/
│   ├── go/                  # Go 共享库
│   │   ├── auth/
│   │   ├── config/
│   │   ├── db/
│   │   ├── events/
│   │   ├── logging/
│   │   ├── policy/
│   │   ├── sdk/
│   │   └── types/
│   └── ts/                  # TypeScript 共享库
│       ├── api-client/
│       └── ui-kit/
├── deploy/
│   ├── docker/              # Docker Compose 部署
│   ├── k8s/                 # Kubernetes Helm Chart
│   └── private/             # 私有化部署配置
├── scripts/
│   ├── smoke-test.sh
│   ├── seed.sh
│   └── migrate.sh
├── docs/
│   ├── envnexus-proposal.md
│   ├── development-roadmap.md
│   └── commercialization-plan.md
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── Makefile
├── deploy.sh
└── go.work
```

## 附录 B：环境变量全量清单

| 变量 | 服务 | 说明 | 引入阶段 |
|---|---|---|---|
| ENX_DATABASE_DSN | platform-api, job-runner | MySQL 连接串 | Phase 0 |
| ENX_REDIS_ADDR | platform-api, session-gateway, job-runner | Redis 地址 | Phase 0 |
| ENX_JWT_SECRET | platform-api | 用户 JWT 签名密钥（必填） | Phase 0 |
| ENX_DEVICE_TOKEN_SECRET | platform-api | 设备 JWT 签名密钥（必填） | Phase 0 |
| ENX_SESSION_TOKEN_SECRET | platform-api, session-gateway | 会话 JWT 签名密钥（必填） | Phase 0 |
| ENX_CORS_ALLOWED_ORIGINS | platform-api, session-gateway | CORS 允许的域名列表 | Phase 0 |
| ENX_OBJECT_STORAGE_ENDPOINT | platform-api, job-runner | MinIO 地址 | Phase 1 |
| ENX_HTTP_PORT | 各服务 | HTTP 监听端口 | Phase 0 |
| ENX_OPENAI_API_KEY | agent-core | OpenAI 密钥 | Phase 1 |
| ENX_DEEPSEEK_API_KEY | agent-core | DeepSeek 密钥 | Phase 1 |
| ENX_ANTHROPIC_API_KEY | agent-core | Anthropic 密钥 | Phase 1 |
| ENX_GEMINI_API_KEY | agent-core | Gemini 密钥 | Phase 1 |
| ENX_OPENROUTER_API_KEY | agent-core | OpenRouter 密钥 | Phase 1 |
| ENX_OLLAMA_URL | agent-core | Ollama 本地地址 | Phase 1 |
| ENX_LLM_PRIMARY | agent-core | 首选 LLM Provider | Phase 1 |
| ENX_WEBHOOK_SECRET | platform-api | Webhook HMAC 密钥 | Phase 4 |
| ENX_STRIPE_SECRET_KEY | platform-api | Stripe 计费密钥 | Phase 6 |
| ENX_LICENSE_SIGNING_KEY | platform-api | 许可证签名密钥 | Phase 6 |

## 附录 C：团队配置建议

| 阶段 | 最少人力 | 理想配置 | 说明 |
|---|---|---|---|
| Phase 0-1 | 1 全栈 | 1 后端 + 1 前端 | 后端专注 Go 服务 + 安全，前端完善 Console |
| Phase 2 | 2 人 | 1 后端 + 1 前端/桌面端 | Desktop 从零构建工作量大 |
| Phase 3-4 | 2 人 | 1 后端 + 1 前端 | RBAC + Webhook 需后端投入，API 文档需协同 |
| Phase 5 | 2-3 人 | 1 后端 + 1 DevOps + 0.5 安全 | K8s + 性能测试 + 安全审计需专项能力 |
| Phase 6 | 2-3 人 | 1 后端 + 1 前端 + 0.5 产品 | 计费集成 + 文档站 + Onboarding 需产品参与 |
