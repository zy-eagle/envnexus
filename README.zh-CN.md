# EnvNexus

[English Version](README.md)

**EnvNexus** 是一个面向企业终端用户的**本地环境自愈平台**。它覆盖普通办公用户和开发者常见的本地环境故障，例如 VPN、代理、DNS、证书、磁盘、数据库、Docker、端口和运行时依赖问题，并用结构化方式替代传统的无限制远程 Shell 访问：**AI 自动诊断 → 生成修复计划 → 用户审批方案 → Agent 本地执行（含回滚）**。用户只需用自然语言描述问题或粘贴报错截图，Agent 即可完成诊断、规划、修复与验证；同时支持**受控的远程文件系统查看与下载**，可在审批后查看终端上的日志和原始文件，并支持**主动巡检**，在问题爆发前识别本地环境隐患。

---

## 解决什么问题

企业终端上的本地环境故障往往高频、反复且直接影响工作效率，但传统处理方式通常只有两种：开远控让 IT 接管电脑，或提工单等待人工排查。前者不安全，后者不高效。

EnvNexus 提供的是一套更适合企业终端用户本地环境问题的处理方式：

- **面向企业终端用户** — 覆盖普通办公用户与开发者的本地环境问题，优先解决网络访问、代理、证书、磁盘、服务、数据库、容器和运行时依赖异常
- **默认只读** — Agent 仅执行诊断工具，除非写操作被显式审批
- **计划式修复** — 诊断结果生成结构化修复计划（有序 DAG，含风险等级、回滚策略和验证步骤）；用户审批的是整个方案，而非单条命令
- **分层审批** — L0 自动通过、L1 计划级审批、L2 计划+确认、L3 逐步审批；可通过策略配置调整
- **受控文件查看与下载** — 支持在审批后浏览远程终端文件系统、预览文本日志、下载任意格式的原始文件，典型用于日志排障与证据采集
- **智能关注点（Watchlist）** — 用户可用自然语言定义 VPN、代理、磁盘、MySQL、Docker 等关注项，Agent 自动拆解并持续巡检
- **主动发现** — 内置规则包 + 平台下发策略 + 从历史修复中学习的规则，在故障爆发前发现隐患
- **全量审计** — 每个会话、诊断、计划、审批、执行和验证都被记录并可查询
- **本地优先执行** — AI 引擎在终端本地运行，平台负责编排但不直接执行命令

### EnvNexus 不是什么

- 不是远程桌面或任意 Shell 工具
- 不是可以执行任意命令的 RMM Agent
- 不是大而全的统一终端管理平台
- 不会绕过本地策略——本地 Agent 始终拥有最终决定权

---

## 架构

```text
┌──────────────────────────────────────────────────────────────┐
│                       平台侧（服务端）                         │
│                   (Docker Compose / K8s)                      │
│                                                              │
│   console-web ──> platform-api ──> session-gateway           │
│   (Next.js 14)    (Go / Gin)       (Go / WebSocket)         │
│      :3000          :8080             :8081                   │
│                       │                                      │
│              ┌────────┼────────┐                             │
│           MySQL    Redis    MinIO        job-runner           │
│                                          (7 个 Worker)       │
│                                                              │
│                    飞书 / Lark Bot（对话式集成）                │
└──────────────────────────┬───────────────────────────────────┘
                           │ HTTPS / WSS
┌──────────────────────────┼───────────────────────────────────┐
│                      终端侧（客户端）                          │
│                                                              │
│   agent-desktop (Electron 30) ──IPC──> agent-core (Go)       │
│   - 系统托盘、诊断对话、                - LLM Router（7 家）    │
│     计划审批、关注点管理、              - 33+ 结构化工具        │
│     健康仪表盘                          - 诊断引擎             │
│                                        - 修复计划引擎         │
│                                        - 关注点巡检引擎       │
│                                        - 治理引擎             │
│                                        - SQLite 本地存储      │
│                                        - OTA 自更新器         │
└──────────────────────────────────────────────────────────────┘
```

**平台侧**：多租户控制面，包含管理控制台、REST API、WebSocket 网关和后台任务调度。基础设施依赖 MySQL、Redis 和 MinIO。

**终端侧**：Go 编写的本地执行内核（`agent-core`）运行在受管设备上，配合 Electron 桌面 UI。内核负责 AI 诊断（支持 7 家 LLM Provider）、修复计划生成、33+ 结构化工具执行、策略执行、关注点巡检（Watchlist）、治理（基线采集 + 漂移检测）以及离线降级模式。

**集成**：飞书对话式 Bot——将群聊绑定到设备后，通过自然语言触发诊断，实时推送进度和群内审批卡片。

---

## 核心设计决策

| 领域 | 方案 |
|------|------|
| **安全** | 无任意 Shell；仅结构化工具；所有写操作需审批；4 层 JWT 令牌体系；AES-256-GCM 本地配置加密 |
| **多租户** | 数据库、缓存、存储、审计均按租户隔离 |
| **AI 集成** | LLM Router 支持 7 家 Provider（OpenAI、Anthropic、DeepSeek、Gemini、OpenRouter、Ollama、本地模型）；结构化输出；tool-calling 模式 |
| **韧性** | Agent 离线时优雅降级（只读工具 + 本地 SQLite）；MinIO 不可用时回退到本地文件系统 |
| **访问控制** | RBAC 5 种预置角色 + 17 条权限；按 IP 限频 |
| **部署** | Docker Compose、Helm Chart 或裸机部署；私有化就绪（无云依赖） |

---

## 快速开始

**前置条件**：Docker 20.10+、Docker Compose v2

```bash
git clone https://github.com/zy-eagle/envnexus.git
cd envnexus
./deploy.sh start
```

脚本自动检测宿主机 IP、生成密钥、计算源码 hash 仅重建有变更的服务，一键启动全部组件。

**默认访问**：`http://localhost:3000` — 登录：`admin@envnexus.io` / `admin123`

更多部署方式参见[部署选项](#部署选项)。

---

## 部署选项

| 方式 | 命令 | 适用场景 |
|------|------|---------|
| **智能部署**（推荐） | `./deploy.sh start` | 一键部署，自动变更检测 |
| **全量重建** | `./deploy.sh full` | 强制重建所有服务 |
| **Docker Compose** | `cd deploy/docker && docker compose up -d` | 手动控制 |
| **Kubernetes** | `helm install envnexus deploy/k8s/helm/envnexus ...` | 生产 K8s 集群 |
| **本地开发** | `make build` + 运行二进制 | Go 服务不使用 Docker 的开发模式 |

默认端口：console-web `:3000`、platform-api `:8080`、session-gateway `:8081`、job-runner `:8082`、agent-core `:17700`（仅本地访问）。

---

## 仓库结构

```text
envnexus/
├── apps/
│   ├── console-web/        # Next.js 14 管理控制台
│   ├── agent-desktop/      # Electron 30 桌面端
│   └── agent-core/         # Go 本地执行内核
├── services/
│   ├── platform-api/       # Go 核心 API（Gin + GORM，DDD 架构）
│   ├── session-gateway/    # Go WebSocket 网关
│   └── job-runner/         # Go 后台任务服务
├── libs/shared/            # Go 共享库
├── deploy/
│   ├── docker/             # Docker Compose 部署
│   └── k8s/helm/envnexus/ # Kubernetes Helm Chart
├── scripts/                # 冒烟测试、初始化数据
├── docs/                   # 用户手册、产品白皮书
└── Makefile
```

---

## 项目状态

所有核心模块功能完备（Phase 0–6 已实现）：

| 模块 | 状态 | 亮点 |
|------|------|------|
| Platform API | 已完成 | 认证、RBAC、Webhook、用量指标、License、飞书集成 |
| Session Gateway | 已完成 | WS 中继、事件去重、Redis pub/sub 水平扩展 |
| Job Runner | 已完成 | 7 个 Worker、原子任务抢占、审计归档 |
| Agent Core | 已完成 | 7 家 LLM、33+ 工具、诊断引擎、治理引擎、离线模式、OTA 更新 |
| Console Web | 已完成 | 12+ 页面、i18n（中/英）、统一 API 客户端 |
| Agent Desktop | 已完成 | 托盘、诊断对话、审批管理、自动更新 |
| 飞书集成 | 已完成 | 对话式诊断、实时推送、审批卡片 |
| 基础设施 | 已完成 | 25 张表、Helm Chart、Docker Compose |

### 已知不足

- **测试**：集成/E2E 测试覆盖不足；无 OpenAPI 规范；无性能基准
- **企业级**：无 SSO（LDAP/SAML/OIDC）；无计费/计量；LLM 提示词仅支持英文
- **可观测性**：无分布式追踪（OpenTelemetry）；无 Prometheus/Grafana；数据库轮询任务队列
- **生态**：仅支持飞书集成（无 Slack/钉钉/企业微信/Teams）；不支持运行时插件加载

---

## 文档

- **用户操作手册**：[`docs/user-manual.md`](docs/user-manual.md) — 终端用户和管理员操作指南
- **商业产品白皮书**：[`docs/product-manual.md`](docs/product-manual.md) — 产品定位与商业方案

## 许可证

参见 [`LICENSE`](LICENSE)。
