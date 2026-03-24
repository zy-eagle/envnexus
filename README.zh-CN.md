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
│  └──────────────┘     └─────┬──┬──┬───────┘     └──────────┬───────────┘       │
│                             │  │  │                         │                   │
│              ┌──────────────┘  │  └──────────────┐          │                   │
│              v                 v                  v          v                   │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐          │
│  │  MySQL 8      │  │   Redis      │  │    MinIO     │  │ Redis    │          │
│  │  (主数据库)    │  │  (缓存/队列)  │  │ (对象存储)   │  │ (pub/sub)│          │
│  │  :3306        │  │  :6379       │  │  :9000       │  │          │          │
│  └───────────────┘  └──────────────┘  └──────────────┘  └──────────┘          │
│                             │                                                   │
│                             v                                                   │
│                     ┌───────────────┐                                           │
│                     │  job-runner   │                                           │
│                     │  (Go Workers) │                                           │
│                     │  :8082        │                                           │
│                     └───────────────┘                                           │
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
| **agent-core** | Go | 本地执行内核。以 `enx-agent` 进程运行在受管终端上，通过 localhost:17700 暴露 API。核心组件：LLM Router（7 个 Provider：OpenAI、Anthropic、DeepSeek、Gemini、OpenRouter、Ollama、本地模型）、工具注册表（10 个结构化工具）、诊断引擎（5 步流水线）、策略引擎、治理引擎（基线采集+漂移检测）、审计客户端（缓冲上传）、SQLite 本地存储。支持离线降级模式。 |
| **agent-desktop** | Electron 30 | 桌面 UI 外壳。系统托盘显示连接状态（在线/离线/连接中）。5 个页面：仪表盘（状态卡片）、诊断对话（多轮聊天）、待审批操作（批准/拒绝）、历史会话、设置（语言、平台地址、日志级别、Agent 路径）。管理 agent-core 子进程生命周期。10 个 IPC 通道通过 contextBridge 暴露。 |

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

---

## 优势与不足

### 优势

- **安全优先设计**：无任意 Shell 暴露，仅结构化工具执行，所有写操作必须审批
- **多租户隔离**：数据库、缓存、对象存储、审计都按租户隔离
- **AI 原生诊断**：LLM Router 支持 7 个 Provider 后端，结构化输出，tool-calling 模式
- **离线可用**：平台不可达时 agent-core 优雅降级（只读工具可用，本地 SQLite 持久化）
- **全量审计链**：每个动作都被记录、可查询、可归档（MinIO 或本地文件系统回退）
- **可扩展工具系统**：实现一个 Go 接口即可添加新工具
- **私有化就绪**：Helm Chart、离线 License Key、本地 LLM（Ollama）、无云依赖

### 当前不足

- **缺少自动化测试**：除单元测试外，集成测试和端到端测试覆盖较低
- **无 OpenAPI/Swagger 文档**：API 文档仅存在于代码中
- **无 LDAP/SAML/OIDC**：企业 SSO 集成尚未实现
- **无 Stripe 计费**：SaaS 支付集成已规划但未构建
- **无桌面端自动更新**：Electron auto-updater 未集成
- **共享库较少**：`libs/shared/` 仅包含基础模型，未充分提取
- **审计导出无 PII 脱敏**：数据脱敏管道尚未实现
- **未使用 Redis 任务队列**：job-runner 使用 DB 轮询而非 Redis 队列

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
│           ├── llm/              #   Router + 7 个 Provider
│           ├── tools/            #   10 个结构化工具（system/network/service/cache）
│           ├── governance/       #   基线采集 + 漂移检测
│           ├── diagnosis/        #   5 步诊断引擎
│           ├── store/            #   SQLite 本地持久化
│           ├── session/          #   WebSocket 客户端
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
│   ├── envnexus-proposal.md      # 完整产品方案
│   ├── development-roadmap.md    # Phase 0-6 路线图及完成状态
│   └── commercialization-plan.md # 商业化计划
├── Makefile
└── README.md
```

---

## 部署指南

### 方式一：Docker Compose（推荐用于开发和单机部署）

**前置条件**：Docker 24+、Docker Compose v2、Git

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
| Platform API | 100% | 认证、CRUD、RBAC、Webhook、用量指标、License、设备 Token 轮换 |
| Session Gateway | 85% | WS 中继、事件去重、session token 认证 |
| Job Runner | 90% | 7 个 Worker（清理、构建、扫描、过期、归档 + FS 回退） |
| Agent Core | 95% | LLM Router（7 家）、10 个工具、治理引擎、离线模式、SQLite |
| Console Web | 95% | 12+ 个页面、i18n、统一 API 客户端、错误边界 |
| Agent Desktop | 85% | 托盘、5 页 UI、spawn agent-core、10 个 IPC 通道 |
| 数据库 | 100% | 25 张表（13 核心 + 12 扩展） |
| K8s 部署 | 90% | Helm Chart（4 服务 + Ingress） |

## 文档

- 产品方案：[`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)
- 开发路线图：[`docs/development-roadmap.md`](docs/development-roadmap.md)
- 商业化计划：[`docs/commercialization-plan.md`](docs/commercialization-plan.md)

## 许可证

参见 [`LICENSE`](LICENSE)。
