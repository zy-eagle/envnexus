# EnvNexus

[English Version](README.md)

**EnvNexus** 是一个面向环境治理、安全本地诊断与引导式修复的 AI 原生平台。它将多租户控制面、Electron 桌面客户端和本地 Go 执行内核组合在一起，提供租户专属分发、策略驱动诊断、审批式修复以及端到端审计能力。

---

## EnvNexus 解决什么问题？

传统远程支持工具向终端授予无限制的 Shell 访问权限，在企业环境中带来安全、合规和审计风险——尤其当数百台设备由分布式团队管理时。

EnvNexus 采用完全不同的方式：

- **默认只读**：Agent 仅执行只读诊断工具，除非写操作被显式审批
- **审批门禁**：每个修复动作（代理切换、配置修改、容器重载）都必须经过多步审批状态机
- **策略驱动**：每个租户定义允许在其受管设备上使用的模型、工具和风险等级
- **全量审计**：每个会话、工具调用、审批决策和回滚都被记录并可查询
- **本地优先执行**：AI 诊断引擎在终端本地运行，平台负责编排但不直接在设备上执行命令

### 典型使用场景

- 终端用户自助诊断："我的网络断了"→ AI 诊断 → 建议修复 → 用户审批 → Agent 执行
- 有审计的远程支持：支持工程师发起会话 → Agent 采集数据 → 修复需要运维审批
- 企业终端治理：10 个租户的 500 台设备，各有不同的模型/策略/工具配置
- 私有化部署：客户在本地管理一切，包括 LLM 密钥和审计存储

### EnvNexus 不是什么

- **不是**远程桌面或任意 Shell 工具
- **不是**可以执行任意命令的 RMM Agent
- **不会**绕过本地策略——本地 Agent 始终拥有最终决定权

---

## 架构概览

### 系统架构图

```text
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              平台侧（服务端）                                    │
│                        (Docker Compose / K8s 部署)                               │
│                                                                                 │
│  ┌──────────────┐     ┌───────────────────┐     ┌──────────────────────┐       │
│  │ console-web  │────>│   platform-api    │────>│    session-gateway   │       │
│  │ (Next.js 14) │     │   (Go / Gin)      │     │   (Go / WebSocket)  │       │
│  │  :3000       │     │   :8080           │     │   :8081              │       │
│  └──────────────┘     └─────┬──┬──┬───┬───┘     └──────────┬───────────┘       │
│                             │  │  │   │                     │                   │
│              ┌──────────────┘  │  │   └────────────────┐    │                   │
│              │                 │  └──────────────┐     │    │                   │
│              v                 v                  v     │    v                   │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐ │ ┌──────────┐          │
│  │  MySQL 8      │  │   Redis      │  │    MinIO     │ │ │ Redis    │          │
│  │  (主数据库)    │  │  (缓存/队列)  │  │ (对象存储)   │ │ │ (pub/sub)│          │
│  │  :3306        │  │  :6379       │  │  :9000       │ │ └──────────┘          │
│  └───────────────┘  └──────────────┘  └──────────────┘ │                       │
│                             │                           │                       │
│                             v                           v                       │
│                     ┌───────────────┐        ┌──────────────────┐              │
│                     │  job-runner   │        │  飞书 / Lark     │              │
│                     │  (Go Workers) │        │  (飞书开放平台)    │              │
│                     │  :8082        │        │  Bot + 卡片交互   │              │
│                     └───────────────┘        └──────────────────┘              │
└─────────────────────────────────────────────────────────────────────────────────┘
                              │ HTTPS/WSS
                              │
┌─────────────────────────────┼───────────────────────────────────────────────────┐
│                    终端侧（客户端）                                               │
│                                                                                 │
│  ┌──────────────────┐     ┌───────────────────────────────────────────┐         │
│  │  agent-desktop   │────>│              agent-core                   │         │
│  │  (Electron 30)   │ IPC │          (Go / localhost:17700)           │         │
│  │                  │     │                                           │         │
│  │  - 系统托盘      │     │  ┌─────────┐ ┌──────────┐ ┌──────────┐  │         │
│  │  - 仪表盘        │     │  │ LLM     │ │ 工具     │ │ 治理     │  │         │
│  │  - 诊断对话      │     │  │ Router  │ │ Registry │ │ 引擎     │  │         │
│  │  - 审批管理      │     │  │(7 家)   │ │(10 工具) │ │(基线+漂移)│  │         │
│  │  - 设置          │     │  └─────────┘ └──────────┘ └──────────┘  │         │
│  └──────────────────┘     │  ┌─────────┐ ┌──────────┐ ┌──────────┐  │         │
│                           │  │ 策略    │ │ 审计     │ │ 诊断     │  │         │
│                           │  │ 引擎    │ │ 客户端   │ │ 引擎     │  │         │
│                           │  └─────────┘ └──────────┘ └──────────┘  │         │
│                           │  ┌──────────────────────────────────┐    │         │
│                           │  │        SQLite + 本地文件           │    │         │
│                           │  └──────────────────────────────────┘    │         │
│                           └───────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 数据流向

```text
                    ┌─────────────────── 控制面 ────────────────────┐
                    │                                                │
  管理员浏览器 ──>  │ console-web ──HTTP──> platform-api ──SQL──> MySQL │
                    │                           │                    │
                    │                      ┌────┴────┐              │
                    │                      │         │              │
                    │                    Redis     MinIO            │
                    │                   (缓存)   (安装包            │
                    │                           & 审计归档)          │
                    │                                                │
                    │ job-runner <──轮询 DB──> MySQL                 │
                    │  (7 个 Worker: 清理、构建、扫描、过期、归档)      │
                    └──────────────────────────┬────────────────────┘
                                               │
                           ┌───────────────────┼───────────────────┐
                           │  HTTP (REST)       │  WebSocket        │
                           │  - 设备注册         │  - 会话事件       │
                           │  - 心跳上报         │  - 工具执行结果    │
                           │  - 配置拉取         │  - 审批流转       │
                           │  - 审计上传         │                   │
                           v                    v                   │
                    ┌──────────────────────────────────────────────┐│
                    │              agent-core                       ││
                    │                                               ││
                    │  1. 启动 → 注册/加载身份                       ││
                    │  2. 拉取远程配置 + 策略                        ││
                    │  3. WebSocket 连接到 session-gateway           ││
                    │  4. 心跳循环（每 60 秒）                       ││
                    │  5. 接收诊断请求                               ││
                    │  6. LLM Router → 选择 Provider → 结构化输出    ││
                    │  7. 工具执行（只读直接执行，写入需审批）          ││
                    │  8. 审计事件 → 平台 + 本地 SQLite               ││
                    │  9. 治理：基线采集 + 漂移检测                    ││
                    └──────────────────────────────────────────────┘│
                           ^                                        │
                           │ IPC (Electron contextBridge)           │
                    ┌──────┴──────┐                                 │
                    │agent-desktop│    用户在此审批/拒绝 ────────────┘
                    │  (Electron) │    修复操作
                    └─────────────┘
```

### 服务调用链（端到端诊断流程）

```text
1. 用户在 agent-desktop 对话框输入"我的网络连不上了"
       │
       ▼
2. agent-desktop ──IPC──> agent-core 本地 API (POST /local/v1/diagnose)
       │
       ▼
3. agent-core 诊断引擎：
   a. 采集系统上下文（网络配置、DNS、进程列表、磁盘使用）
   b. 将上下文 + 用户问题发送给 LLM Router
   c. LLM Router 选择 Provider（OpenAI / DeepSeek / Anthropic / Ollama / ...）
   d. LLM 返回结构化诊断结果和推荐工具
       │
       ▼
4. 只读工具立即执行（read_network_config、read_system_info 等）
   写入工具 → 创建 ApprovalRequest（审批请求）
       │
       ▼
5. agent-core ──HTTP POST──> platform-api（创建审批请求）
   platform-api 存入 MySQL，生成审计事件
       │
       ▼
6. agent-desktop 轮询待审批列表，展示给用户：
   "工具: proxy.toggle | 风险: L1 | 操作: 启用代理 http://proxy:8080"
   [批准] [拒绝]
       │
       ▼
7. 用户点击 [批准]
   agent-desktop ──IPC──> agent-core ──HTTP POST──> platform-api（审批通过）
       │
       ▼
8. agent-core 执行工具，记录结果
   agent-core ──HTTP POST──> platform-api（审计事件: tool.executed, succeeded）
       │
       ▼
9. 结果显示在 agent-desktop 对话界面
   审计记录可在 console-web 审计事件页面查询

---  飞书对话式路径 ---

0f. 运维人员在飞书群发送 "/bind dev_ABC"
    Bot 通过 ChatBridge 建立群聊 → 设备映射
       │
       ▼
1f. 运维人员在群里输入 "网络连不上了"
    飞书 ──webhook──> platform-api /webhook/feishu/event
    BotService 创建会话，网关通知 agent-core
       │
       ▼
2f. 诊断过程中 → EventSink 实时推送进度：
    "🔄 采集中..." → "🧠 AI 分析中..." → [诊断结果卡片]
       │
       ▼
3f. 如需修复 → [审批卡片] 推送到群，内嵌 ✅/❌ 按钮
    运维人员点击 [批准] → 飞书 ──POST──> /webhook/feishu/card
       │
       ▼
4f. 接续步骤 8（agent-core 执行工具操作）
    工具执行进度和结果同步推送回飞书群
```

---

## 各模块详解

### 平台服务

| 模块 | 语言 | 端口 | 职责 |
|------|------|------|------|
| **platform-api** | Go (Gin) | 8080 | 核心 REST API。负责认证（JWT access/refresh/device/session 四种令牌）、租户 CRUD、Profile 管理（模型/策略/Agent）、设备注册与心跳、会话管理、审批状态机（drafted→pending→approved→executing→succeeded/failed/rolled_back）、审计事件、RBAC（5 角色 17 权限）、Webhook 分发、用量指标、License 验证。25+ 个 API 资源组。 |
| **session-gateway** | Go (Gorilla WS) | 8081 | WebSocket 网关。在平台和 Agent 之间中继会话事件（诊断请求、工具结果、审批流转）。使用 session token 认证。基于 event_id 去重。通过 Redis pub/sub 支持水平扩展。 |
| **job-runner** | Go | 8082 | 后台任务服务。运行 7 个 Worker：`token_cleanup`（过期令牌清理）、`link_cleanup`（过期下载链接清理）、`audit_flush`（审计归档到 MinIO，MinIO 不可用时回退到本地文件系统）、`session_cleanup`（过期会话清理）、`approval_expiry`（审批超时自动过期）、`package_build`（安装包构建）、`governance_scan`（治理扫描）。轮询 MySQL 中的 jobs 表。 |
| **console-web** | Next.js 14 | 3000 | 管理控制台。12+ 个页面：登录、租户管理、设备列表（在线/离线状态实时显示）、会话列表+详情、审计事件（多字段筛选）、模型/策略/Agent Profile、下载包管理。集中式 i18n（中/英双语）。统一 API 客户端。 |

### 终端应用

| 模块 | 语言 | 职责 |
|------|------|------|
| **agent-core** | Go | 本地执行内核。以 `enx-agent` 进程运行在受管终端上，通过 localhost:17700 暴露 API。核心组件：LLM Router（7 个 Provider：OpenAI、Anthropic、DeepSeek、Gemini、OpenRouter、Ollama、本地模型）、工具注册表（10 个结构化工具）、诊断引擎（5 步流水线）、策略引擎、治理引擎（基线采集+漂移检测）、审计客户端（缓冲上传）、SQLite 本地存储、**OTA 自更新器**（检查更新、下载、应用，Windows 平台含杀毒预热）、**AES-256-GCM 配置加密**（基于机器指纹的 PBKDF2 密钥）。支持离线降级模式。心跳中同时上报 Agent Core 版本和分发包版本。 |
| **agent-desktop** | Electron 30 | 桌面 UI 外壳。系统托盘显示连接状态（在线/离线/连接中）。6 个页面：仪表盘（状态卡片）、诊断对话（多轮聊天）、待审批操作（批准/拒绝）、历史会话、设置（语言、平台地址、日志级别、Agent 路径）、**更新管理**（Agent + 桌面端双通道更新横幅+进度）。管理 agent-core 子进程生命周期。10 个 IPC 通道通过 contextBridge 暴露。**加密设置存储**（AES-256-GCM）。更新后通过 `core_install_path.json` 协调进程启动路径。 |

### 集成模块

| 模块 | 对接目标 | 职责 |
|------|---------|------|
| **飞书（Lark）对话式 Bot** | 飞书开放平台 | 双向对话式集成。通过 `/bind` 将飞书群绑定到设备，之后在群内发送自然语言消息即触发诊断。系统实时推送诊断进度（采集、分析、结果）、交互审批卡片（群内一键批准/拒绝）、工具执行状态和会话完成摘要。ChatBridge 支持 Redis 持久化的聊天↔设备↔会话映射。仅需 3 个环境变量。 |

### 工具注册表（agent-core）

| 工具 | 风险等级 | 读/写 | 说明 |
|------|---------|------|------|
| `read_network_config` | L0 | 只读 | 采集 IP 地址、DNS 服务器、路由表 |
| `read_system_info` | L0 | 只读 | 操作系统、主机名、CPU、内存 |
| `read_disk_usage` | L0 | 只读 | 磁盘分区和使用率 |
| `read_process_list` | L0 | 只读 | 运行中的进程（PID、进程名） |
| `flush_dns` | L1 | 写入 | 刷新系统 DNS 缓存 |
| `service.restart` | L2 | 写入 | 重启系统服务 |
| `cache.rebuild` | L1 | 写入 | 重建应用缓存 |
| `proxy.toggle` | L1 | 写入 | 启用/禁用系统代理（支持 Linux/macOS/Windows） |
| `config.modify` | L1 | 写入 | 修改白名单内的配置键 |
| `container.reload` | L2 | 写入 | 重载容器或进程（docker/systemd/SIGHUP） |

### 基础设施

| 组件 | 用途 |
|------|------|
| **MySQL 8** | 主数据库。25 张表（13 核心 + 12 扩展）。存储租户、用户、角色、设备、会话、审计事件、审批请求、Profile、Webhook、任务、用量指标、License。 |
| **Redis** | 缓存（令牌黑名单、频率限制）、session-gateway 的 pub/sub（WebSocket 消息扇出）。 |
| **MinIO** | S3 兼容对象存储。存储 Agent 安装包和审计归档文件。不可用时自动回退到本地文件系统。 |
| **SQLite** | Agent 侧本地持久化。存储会话、审计事件、配置缓存、治理基线和漂移记录。确保离线场景下数据不丢失。 |

### 安全与认证

| 机制 | 说明 |
|------|------|
| **JWT Access Token** | 1 小时过期。console-web 使用。携带 user_id、tenant_id。 |
| **JWT Refresh Token** | 7 天过期。通过 `POST /api/v1/auth/refresh` 换取新 access token。 |
| **JWT Device Token** | 1 年过期。设备注册时签发。agent-core 调用平台 API 时使用。支持通过 `POST /devices/:id/rotate-token` 轮换。 |
| **JWT Session Token** | 30 分钟过期。绑定到特定 WebSocket 会话。 |
| **RBAC** | 5 种预置角色：`platform_super_admin`（平台超管）、`tenant_admin`（租户管理员）、`security_auditor`（安全审计员）、`ops_operator`（运维操作员）、`read_only_observer`（只读观察者）。17 条权限常量。`RequirePermission` 中间件。 |
| **Rate Limiting** | 登录接口：10 次/分钟/IP。通用 API：50 次/秒/IP。 |
| **CORS** | 通过 `ENX_CORS_ALLOWED_ORIGINS` 环境变量配置。 |
| **本地配置加密** | Agent 配置文件（`.enx`）和桌面端设置使用 AES-256-GCM 加密，密钥通过 PBKDF2 从机器指纹（主机名 + 可执行文件路径）派生。`ENX_ENC:` 前缀标识加密文件。 |

---

## 优势与不足

### 优势

- **安全优先设计**：无任意 Shell 暴露，仅结构化工具执行，所有写操作必须审批
- **多租户隔离**：数据库、缓存、对象存储、审计都按租户隔离
- **AI 原生诊断**：LLM Router 支持 7 个 Provider 后端，结构化输出，tool-calling 模式
- **离线可用**：平台不可达时 agent-core 优雅降级（只读工具可用，本地 SQLite 持久化）
- **全量审计链**：每个动作都被记录、可查询、可归档（MinIO 或本地文件系统回退）
- **可扩展工具系统**：实现一个 Go 接口即可添加新工具
- **OTA 自动更新**：Agent Core 自动向平台检查更新，下载并应用新版本二进制（Windows 平台含杀毒安全预热）；桌面端协调更新后的进程启动路径
- **本地配置加密**：Agent 和桌面端配置文件静态加密，使用机器绑定的 AES-256-GCM 密钥
- **私有化就绪**：Helm Chart、离线 License Key、本地 LLM（Ollama）、无云依赖

### 当前不足

**测试与质量**
- **集成/端到端测试覆盖不足**：仅有单元测试；无 CI 集成测试、无 console-web 的 Playwright/Cypress E2E 测试、无桌面端 UI 测试
- **无 OpenAPI/Swagger 文档**：API 文档仅存在于代码注释中，无法供第三方消费方使用生成的规范
- **无性能基准**：platform-api 和 session-gateway 缺少负载测试套件和性能分析基线

**企业级功能**
- **无 SSO（LDAP/SAML/OIDC）**：认证完全依赖内置 JWT，无法接入企业身份提供商
- **无计费/计量集成**：SaaS 支付（Stripe/Paddle）和按租户用量计费已规划但未构建
- **无多语言 LLM 提示词**：诊断提示词和工具描述仅支持英文，尚未实现本地化提示词工程

**运维与可观测性**
- **数据库轮询任务队列**：job-runner 使用 MySQL 轮询而非 Redis Streams/NATS，吞吐量上限约 1000 任务/分钟
- **无分布式追踪**：缺少 OpenTelemetry 集成，跨服务请求关联需手动搜索日志
- **无 Prometheus/Grafana 监控**：platform-api 暴露 `/readyz` 但无 `/metrics` 端点，无预置仪表盘
- **审计日志保留策略不可配**：归档由 job-runner 自动执行，但保留规则无法通过控制台配置

**安全与合规**
- **无渗透测试报告**：平台尚未经过第三方安全评估
- **无密钥自动轮换**：JWT/设备/会话密钥需通过环境变量手动轮换

**桌面端与 Agent**
- **单租户 Agent**：agent-core 一次仅连接一个平台实例，不支持多平台切换
- **macOS 未公证**：agent-desktop 的 macOS 构建未签名，用户需手动绕过 Gatekeeper

**生态**
- **IM 集成有限**：目前仅支持飞书（Lark），Slack、钉钉、企业微信、Microsoft Teams 集成尚未构建
- **无插件/扩展市场**：工具注册表仅支持编译时注入，不支持运行时插件加载

---

## 仓库结构

```text
envnexus/
├── apps/
│   ├── console-web/              # Next.js 14 管理控制台（TypeScript）
│   │   ├── src/app/              #   页面：登录、租户、设备、会话、审计、Profile
│   │   ├── src/components/       #   侧边栏、头部
│   │   └── src/lib/              #   API 客户端、i18n 字典
│   ├── agent-desktop/            # Electron 30 桌面端（TypeScript）
│   │   └── src/
│   │       ├── main/main.ts      #   主进程：托盘、窗口、IPC、子进程管理
│   │       ├── preload/          #   contextBridge（10 个 IPC 通道）
│   │       └── renderer/         #   多页面 HTML UI
│   └── agent-core/               # Go 本地执行内核
│       ├── cmd/enx-agent/        #   入口
│       └── internal/
│           ├── bootstrap/        #   10 步启动序列
│           ├── config/           #   配置管理 + AES-256-GCM 加密
│           ├── llm/              #   Router + 7 个 Provider
│           ├── tools/            #   10 个结构化工具（system/network/service/cache）
│           ├── governance/       #   基线采集 + 漂移检测
│           ├── diagnosis/        #   5 步诊断引擎
│           ├── store/            #   SQLite 本地持久化
│           ├── session/          #   WebSocket 客户端
│           ├── lifecycle/        #   平台心跳、配置拉取、会话管理
│           ├── updater/          #   OTA 自更新（检查、下载、应用）
│           └── api/              #   本地 HTTP 服务 (:17700)
├── services/
│   ├── platform-api/             # Go 核心 API（Gin + GORM）
│   │   ├── cmd/platform-api/     #   入口 + DI 组装
│   │   ├── internal/
│   │   │   ├── domain/           #   领域实体（DDD）：Session、ApprovalRequest、Role、Webhook...
│   │   │   ├── repository/       #   MySQL 仓储（GORM）
│   │   │   ├── service/          #   业务逻辑：auth、rbac、webhook、metrics、license...
│   │   │   ├── handler/          #   HTTP 处理器（控制台 API + Agent API）
│   │   │   ├── middleware/       #   JWT 认证、RBAC、限频、CORS、响应信封
│   │   │   ├── infrastructure/   #   Redis、MinIO、Gateway 客户端
│   │   │   ├── integration/
│   │   │   │   └── feishu/      #   飞书 Bot、交互卡片、事件 Webhook、命令处理
│   │   │   └── dto/              #   请求/响应 DTO
│   │   └── migrations/           #   SQL 迁移脚本（启动时自动执行）
│   ├── session-gateway/          # Go WebSocket 网关
│   │   └── internal/handler/ws/  #   WS 处理器 + event_id 去重
│   └── job-runner/               # Go 后台任务服务
│       └── internal/worker/      #   7 个 Worker
├── libs/shared/                  # Go 共享库（errors、基础模型）
├── deploy/
│   ├── docker/                   # Docker Compose 部署
│   │   ├── docker-compose.yml
│   │   ├── Dockerfile.*          #   各服务 Dockerfile
│   │   └── .env.example
│   └── k8s/helm/envnexus/       # Kubernetes Helm Chart
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/            #   4 个 Deployment + Service、Ingress、Secrets
├── scripts/
│   ├── smoke-test.sh             # MVP 12 步冒烟测试
│   └── seed.sh                   # 初始化默认租户和管理员
├── docs/
│   ├── user-manual.md            # 用户操作手册
│   ├── product-requirements.md   # 产品需求文档 (PRD)
│   ├── product-manual.md         # 商业产品白皮书
│   ├── technical-architecture.md # 技术架构与实现方案
│   ├── development-roadmap.md    # Phase 0-6 路线图及完成状态
│   └── commercialization-plan.md # 商业化计划
├── Makefile
└── README.md
```

---

## 部署指南

### 方式一：智能部署脚本（推荐）

**前置条件**：Docker 20.10+、Docker Compose v2、Git、Bash

`deploy.sh` 脚本提供智能部署功能，包含源码变更检测、自动 `.env` 生成和并行 Agent 构建：

```bash
# 1. 克隆并部署
git clone https://github.com/zy-eagle/envnexus.git
cd envnexus
./deploy.sh start
```

脚本自动完成以下工作：
- 检测宿主机 IP，生成 `deploy/docker/.env` 并自动填充随机密钥
- 计算每个服务源码目录的 SHA256 内容 hash，仅重建有变更的服务
- 交叉编译 Agent 二进制 + Electron 桌面安装包，上传至 MinIO
- 启动所有基础设施（MySQL、Redis、MinIO）和应用服务

**常用命令**：

| 命令 | 说明 |
|------|------|
| `./deploy.sh start` | **智能部署**（推荐）：检测变更，仅重建有变化的服务 |
| `./deploy.sh full` | 强制全量重建所有服务和 Agent 安装包 |
| `./deploy.sh web` | 仅重建并部署前端 |
| `./deploy.sh api` | 仅重建并部署后端服务 |
| `./deploy.sh agents` | 强制重新编译 Agent 安装包并上传 MinIO |
| `./deploy.sh stop` | 停止所有服务（数据保留在 volumes 中） |
| `./deploy.sh status` | 查看服务运行状态 |
| `./deploy.sh logs [服务名]` | 查看服务日志 |
| `./deploy.sh reset` | 删除所有数据，恢复全新状态 |

### 方式一（备选）：手动 Docker Compose

如需手动控制：

```bash
# 1. 克隆仓库
git clone https://github.com/zy-eagle/envnexus.git
cd envnexus

# 2. 配置密钥
cp deploy/docker/.env.example deploy/docker/.env
# 重要：编辑 .env，将以下三项设置为安全随机值：
#   ENX_JWT_SECRET
#   ENX_DEVICE_TOKEN_SECRET
#   ENX_SESSION_TOKEN_SECRET

# 3. 启动所有服务
cd deploy/docker
docker compose up -d

# 4. 验证健康状态
curl http://localhost:8080/healthz    # platform-api
curl http://localhost:8081/healthz    # session-gateway
curl http://localhost:8082/healthz    # job-runner

# 5. 初始化默认数据
cd ../..
bash scripts/seed.sh

# 6. 打开控制台
# http://localhost:3000
# 登录账号：admin@envnexus.io / admin123
```

**服务启动顺序**（Docker Compose 通过 `depends_on` 自动处理）：

```text
mysql → redis → minio → migration → platform-api → session-gateway → job-runner → console-web
```

**默认端口**：

| 服务 | 端口 | 说明 |
|------|------|------|
| console-web | 3000 | 管理控制台 |
| platform-api | 8080 | REST API |
| session-gateway | 8081 | WebSocket |
| job-runner | 8082 | 仅健康检查 |
| MySQL | 3306 | |
| Redis | 6379 | |
| MinIO API | 9000 | |
| MinIO Console | 9001 | |
| agent-core | 17700 | 仅本地访问 |

### 方式二：Kubernetes（Helm Chart）

```bash
# 添加 Bitnami 仓库（MySQL/Redis 依赖）
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# 安装
helm install envnexus deploy/k8s/helm/envnexus \
  --namespace envnexus --create-namespace \
  --set env.ENX_JWT_SECRET="$(openssl rand -hex 32)" \
  --set env.ENX_DEVICE_SECRET="$(openssl rand -hex 32)" \
  --set env.ENX_SESSION_SECRET="$(openssl rand -hex 32)"
```

Helm Chart 部署：platform-api（2 副本）、session-gateway（2 副本）、job-runner（1 副本）、console-web（2 副本），以及 Ingress 和 Secrets。

### 方式三：本地开发（Go 服务不使用 Docker）

```bash
# 仅启动基础设施
cd deploy/docker && docker compose up -d mysql redis minio && cd ../..

# 编译所有 Go 服务
make build

# 终端 1：platform-api
export ENX_DATABASE_DSN="root:root@tcp(localhost:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
export ENX_REDIS_ADDR="localhost:6379"
export ENX_OBJECT_STORAGE_ENDPOINT="localhost:9000"
export ENX_JWT_SECRET="dev-secret"
export ENX_DEVICE_TOKEN_SECRET="dev-device-secret"
export ENX_SESSION_TOKEN_SECRET="dev-session-secret"
./bin/platform-api

# 终端 2：session-gateway
export ENX_SESSION_TOKEN_SECRET="dev-session-secret"
export ENX_REDIS_ADDR="localhost:6379"
./bin/session-gateway

# 终端 3：job-runner
export ENX_DATABASE_DSN="root:root@tcp(localhost:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
./bin/job-runner

# 终端 4：console-web
cd apps/console-web && pnpm install && pnpm dev

# 终端 5：agent-core
./bin/enx-agent
```

### 冒烟测试

```bash
bash scripts/smoke-test.sh
```

验证完整闭环：健康检查 → 登录 → 租户 → Profile → 下载链接 → 注册 → 会话 → 审计。

### 飞书（Lark）对话式集成

EnvNexus 支持在飞书群聊中与 Agent 进行对话式交互。将群聊绑定到设备后，群内发送的任何消息都会被当作诊断请求——诊断结果、审批卡片、工具执行状态会实时推送回群。

**配置（仅需 3 个环境变量）**：

1. 在[飞书开放平台](https://open.feishu.cn/)创建自建应用
2. 开启 **机器人** 能力和 **事件订阅**
3. 设置事件请求地址: `https://你的域名/webhook/feishu/event`
4. 设置卡片操作请求地址: `https://你的域名/webhook/feishu/card`
5. 订阅 `im.message.receive_v1` 事件
6. 仅需设置 3 个环境变量：

| 环境变量 | 说明 |
|---------|------|
| `ENX_FEISHU_APP_ID` | 飞书应用 App ID |
| `ENX_FEISHU_APP_SECRET` | 飞书应用 App Secret |
| `ENX_FEISHU_VERIFICATION_TOKEN` | 事件回调验证 Token |

无需配置群聊 ID — 通过 `/bind` 命令动态绑定。

**对话式使用示例**：

```text
用户:  /bind dev_01J8XYZABC
Bot:   ✅ 绑定成功！设备: dev_01J8XYZABC (my-server / linux)
       现在可以直接在群里发消息进行诊断...

用户:  网络连不上了
Bot:   🚀 已向设备 dev_01J8XYZABC 发起诊断
       会话: sess_01J8XYZDEF
       诊断进行中，结果会自动推送到本群...
Bot:   🔄 诊断已启动，正在采集系统信息...
Bot:   📊 正在采集: network_config
Bot:   🧠 AI 正在分析采集的数据...
Bot:   [诊断结果卡片: 发现 DNS 配置异常，建议修复]
Bot:   [审批卡片: 工具 dns.reset | 风险 L1 | ✅批准 ❌拒绝]

用户:  (在卡片上点击 ✅ 批准)
Bot:   ✅ 审批已通过，正在执行修复...
Bot:   ⚙️ 正在执行: dns.reset ...
Bot:   ✅ 工具 dns.reset 执行成功
Bot:   [会话完成卡片]
```

**Bot 命令**：

| 命令 | 说明 |
|------|------|
| `/bind <device_id>` | 绑定设备到当前群 — 开启对话式诊断 |
| `/unbind` | 解绑当前设备 |
| `/who` | 查看当前绑定信息 |
| `/status` | 查看平台运行状态 |
| `/devices` | 列出已注册设备（附绑定提示） |
| `/pending` | 查看当前会话的待审批请求 |
| `/approve <id>` | 批准修复操作 |
| `/deny <id> [原因]` | 拒绝修复操作 |
| `/audit [device_id]` | 查看审计事件（默认使用绑定设备） |

### Makefile 目标

| 目标 | 说明 |
|------|------|
| `make build` | 编译所有 Go 二进制文件到 `./bin/` |
| `make run-api` | 本地运行 platform-api |
| `make run-gateway` | 本地运行 session-gateway |
| `make run-runner` | 本地运行 job-runner |
| `make run-web` | 启动 console-web 开发服务器 |
| `make run-desktop` | 以开发模式运行 agent-desktop |

---

## 项目状态

Phase 0-6 核心功能已实现。详细的逐模块完成状态请参阅[开发路线图](docs/development-roadmap.md)。

| 模块 | 完成度 | 核心能力 |
|------|--------|---------|
| Platform API | 100% | 认证、CRUD、RBAC、Webhook、用量指标、License、设备 Token 轮换、飞书集成 |
| Session Gateway | 100% | WS 中继、事件去重、session token 认证、Redis pub/sub 水平扩展 |
| Job Runner | 100% | 7 个 Worker、原子任务抢占（FOR UPDATE SKIP LOCKED）、审计归档 + FS 回退 |
| Agent Core | 100% | LLM Router（7 家）、10 个工具、治理引擎、离线模式、SQLite、运行时调度器 |
| Console Web | 100% | 12+ 个页面、i18n、统一 API 客户端、错误边界 |
| Agent Desktop | 100% | 托盘、5 页 UI、spawn agent-core、10 个 IPC 通道、自动更新（electron-updater） |
| 飞书集成 | 100% | 对话式诊断（群绑定设备）、实时事件推送、审批卡片、漂移告警 |
| 数据库 | 100% | 25 张表（13 核心 + 12 扩展） |
| K8s 部署 | 100% | Helm Chart（4 服务 + Ingress、锁定依赖版本） |

## 文档

- 用户操作手册：[`docs/user-manual.md`](docs/user-manual.md) — 终端用户和管理员操作指南
- 产品需求文档：[`docs/product-requirements.md`](docs/product-requirements.md) — PRD 功能规格
- 商业产品白皮书：[`docs/product-manual.md`](docs/product-manual.md) — 产品定位与商业方案
- 技术架构文档：[`docs/technical-architecture.md`](docs/technical-architecture.md) — 系统设计与实现细节
- 开发路线图：[`docs/development-roadmap.md`](docs/development-roadmap.md) — Phase 0-6 完成状态
- 商业化计划：[`docs/commercialization-plan.md`](docs/commercialization-plan.md) — 商业化计划

## 许可证

参见 [`LICENSE`](LICENSE)。
