# EnvNexus

[English Version](README.md)

EnvNexus 是一个面向环境治理、安全本地诊断与引导式修复的 AI 原生平台。它将多租户平台、桌面客户端和本地执行内核组合在一起，提供租户专属分发、策略驱动诊断、审批式修复以及端到端审计能力。

## 项目状态

EnvNexus 正在积极开发中（Phase 1 MVP 已完成，Phase 2 进行中）。

- 产品方案：[`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)
- 开发路线图：[`docs/development-roadmap.md`](docs/development-roadmap.md)
- 商业化计划：[`docs/commercialization-plan.md`](docs/commercialization-plan.md)

## 项目要解决什么问题

EnvNexus 面向这样一类场景：用户或运维人员需要定位并修复环境问题，但又不能把产品做成"无限制远控工具"。

典型场景包括：

- 终端用户自助诊断与引导式修复
- 技术支持远程协助排障，但本地修改必须经过审批
- 企业终端环境治理，按租户下发模型与策略
- 混合部署或私有化部署中，对本地模型和密钥有强边界要求的场景

核心原则：

- 默认只读，先诊断后修复
- 所有写操作必须显式审批
- 本地策略优先于云端建议
- 全量审计，可回滚、可追责
- 平台负责编排，终端负责安全执行

## 产品概览

EnvNexus 由三个核心层次组成：

- 平台层：负责租户管理、模型配置、策略配置、下载链接、设备注册、审计查询和包管理元数据
- 终端层：负责本地诊断、审批式修复、审计采集和安全执行
- 集成层：负责 `WebSocket`、`Webhook` 和私有网络接入

MVP 目标是在一台平台主机和一台受管终端之间跑通完整闭环：

`登录 -> 租户配置 -> 签名下载链接 -> 首次激活 -> 设备注册 -> 只读诊断 -> 审批式低风险修复 -> 审计上报`

## 目标架构

已固定的技术选型：

- Web 控制台：`Next.js 14 + TypeScript`
- 后端服务：`Go`
- 后端边界：`platform-api`、`session-gateway`、`job-runner`
- 桌面端界面层：`Electron 30 + React + TypeScript`
- 本地执行内核：`Go agent-core`
- 主数据库：`MySQL 8`
- 缓存与短状态：`Redis`
- 对象存储：`MinIO`（S3 兼容）
- Agent 本地状态：`SQLite + Files`
- 首版部署方式：`Docker Compose`

高层运行拓扑如下：

```text
管理端浏览器
    |
    v
console-web
    |
    v
platform-api
    +--> MySQL
    +--> Redis
    +--> MinIO
    +--> session-gateway
    \--> job-runner

agent-desktop
    |
    v
agent-core（本地 API）
    +--> platform-api
    +--> session-gateway
    \--> SQLite + Files
```

## 仓库结构

```text
envnexus/
  apps/
    console-web/         # Next.js 14 管理控制台
    agent-desktop/       # Electron 桌面端
    agent-core/          # Go 本地执行内核
  services/
    platform-api/        # Go REST API（Gin + GORM）
    session-gateway/     # Go WebSocket 网关
    job-runner/          # Go 后台任务服务
  libs/
    shared/              # Go 共享库
  deploy/
    docker/              # Docker Compose 部署配置
  scripts/
    smoke-test.sh        # MVP 12 步冒烟测试
    seed.sh              # 初始化默认租户和管理员
  docs/
```

## 当前已实现能力

**Phase 0 — 安全加固与工程基础（已完成）**

- 移除硬编码默认密钥，未设置时启动报错
- 修复 WebSocket 鉴权旁路和 CheckOrigin 全放行
- 添加 CORS 中间件和 Rate Limiting
- 全服务迁移至结构化日志（`log/slog`）
- 数据库迁移自动执行，readyz 覆盖 DB / Redis / MinIO 检查
- 建立 CI/CD 流水线（GitHub Actions）
- 审批状态机和会话状态机单元测试

**Phase 1 — MVP 闭环（已完成）**

- 控制台登录与租户配置，`ModelProfile`、`PolicyProfile`、`AgentProfile` CRUD
- Refresh Token（Access 1h + Refresh 7d），`POST /auth/refresh`
- 租户专属签名下载链接，`POST /tenants/:id/download-links`
- Agent WebSocket 鉴权修复（使用 session token 接入 Gateway）
- 审批式工具执行上报（succeeded / failed）
- `audit_flush` 真实归档至 MinIO，审计支持 `include_archived` 查询
- WebSocket 事件幂等去重（event_id + 10 分钟 TTL）
- Console Web 集中式 i18n（`dictionary.ts`），全页面中英双语

**Phase 2 — 诊断产品化（Agent Core + Console Web 已完成，Desktop 待开发）**

- SQLite 本地存储（`internal/store/`），覆盖会话、审计、配置缓存、治理基线和漂移
- 治理引擎 v1：`CaptureBaseline` / `DetectDrift`，采集主机名、网络接口、环境变量
- 扩展只读工具，共 7 个（新增 `read_disk_usage`、`read_process_list`）
- `POST /local/v1/diagnostics/export` 诊断包导出
- 离线降级模式：平台不可达时仅开放只读能力
- 优雅退出：`Shutdown()` 正确关闭 LocalServer 和 SQLite
- Console Web 全局错误边界（`error.tsx` / `loading.tsx`）
- 所有页面统一通过 `api` client 调用，消除散落的直接 `fetch`
- 会话详情页：展示会话元数据 + 审计事件时间线（可展开 payload）
- 设备在线/离线实时状态：绿色脉冲指示灯 + 30 秒轮询
- 审计事件多字段筛选：session_id / device_id / event_type / include_archived

## 本地开发

### 前置条件

- Go 1.21+
- Node.js 20+ 和 pnpm
- Docker 和 Docker Compose

### Docker Compose 快速启动

```bash
# 1. 克隆仓库
git clone https://github.com/zy-eagle/envnexus.git
cd envnexus

# 2. 配置环境变量
cp deploy/docker/.env.example deploy/docker/.env
# 编辑 .env，将 ENX_JWT_SECRET、ENX_DEVICE_TOKEN_SECRET、ENX_SESSION_TOKEN_SECRET 设置为安全随机值

# 3. 启动所有服务
cd deploy/docker
docker compose up -d

# 4. 等待健康检查通过后，初始化默认数据
cd ../..
bash scripts/seed.sh

# 5. 打开控制台
# http://localhost:3000
# 登录账号：admin@envnexus.io / admin123
```

### 本地运行（不使用 Docker）

```bash
# 仅启动基础设施（MySQL、Redis、MinIO）
cd deploy/docker && docker compose up -d mysql redis minio && cd ../..

# 编译所有 Go 服务
make build

# 运行 platform-api
export ENX_DATABASE_DSN="root:root@tcp(localhost:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
export ENX_REDIS_ADDR="localhost:6379"
export ENX_OBJECT_STORAGE_ENDPOINT="localhost:9000"
export ENX_JWT_SECRET="dev-secret-change-in-prod"
export ENX_DEVICE_TOKEN_SECRET="dev-device-secret"
export ENX_SESSION_TOKEN_SECRET="dev-session-secret"
./bin/platform-api

# 运行 session-gateway（另开终端）
export ENX_SESSION_TOKEN_SECRET="dev-session-secret"
export ENX_REDIS_ADDR="localhost:6379"
./bin/session-gateway

# 运行 job-runner（另开终端）
export ENX_DATABASE_DSN="root:root@tcp(localhost:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
./bin/job-runner

# 运行 console-web（另开终端）
cd apps/console-web && pnpm install && pnpm dev

# 运行 agent-core（另开终端）
./bin/enx-agent
```

### 冒烟测试

```bash
bash scripts/smoke-test.sh
```

冒烟测试验证完整 MVP 闭环：健康检查、登录、租户配置、Profile 创建、下载链接生成、Agent 注册、会话创建、审计追踪。

### Makefile 目标

| 目标 | 说明 |
|------|------|
| `make build` | 编译所有 Go 二进制文件到 `./bin/` |
| `make run-api` | 本地运行 platform-api |
| `make run-gateway` | 本地运行 session-gateway |
| `make run-runner` | 本地运行 job-runner |
| `make run-web` | 启动 console-web 开发服务器 |
| `make run-desktop` | 以开发模式运行 agent-desktop |

## 部署形态

方案定义了三种部署模式：

- `Hosted`：平台侧统一托管控制面和存储
- `Hybrid`：平台共享控制面，企业侧保留密钥或模型边界
- `Private`：客户自管完整部署，沿用相同协议和对象模型

首个交付目标是单机 MVP 部署：

- 一台 Linux 主机运行平台侧组件
- 一台受管终端运行 `agent-core` 和 `agent-desktop`
- 平台侧通过 `Docker Compose` 启动

## 部署指南

### 平台服务组件

平台侧最小组件集合：

- `console-web`
- `platform-api`
- `session-gateway`
- `job-runner`
- `mysql`
- `redis`
- `minio`

预期启动顺序：

1. `mysql`
2. `redis`
3. `minio`
4. migration job
5. `platform-api`
6. `session-gateway`
7. `job-runner`
8. `console-web`

预期默认端口：

- `console-web`: `3000`
- `platform-api`: `8080`
- `session-gateway`: `8081`
- `mysql`: `3306`
- `redis`: `6379`
- `minio-api`: `9000`
- `minio-console`: `9001`
- `agent-core local api`: `127.0.0.1:17700`

平台部署约束：

- 所有业务服务必须通过环境变量或 `env_file` 注入配置
- 所有容器日志必须输出到标准输出
- 所有持久化数据必须映射到宿主机卷
- 服务的 ready 状态必须受 migration 和上游依赖约束

## 运行流程

1. 通过 `Docker Compose` 启动平台服务
2. 完成控制台初始化登录与租户配置
3. 配置模型、策略和 AgentProfile
4. 生成租户专属签名下载链接
5. 在受管终端下载并启动桌面 Agent
6. 完成设备首次激活并拉取远程配置
7. 通过本地 UI 或远程会话入口发起诊断
8. 所有写操作都必须先经过显式审批
9. 执行结果和审计事件回传到平台侧

## 可观测性与运维

方案要求平台侧和终端侧都具备一等公民级别的可观测性：

- 结构化日志，统一携带请求、租户、设备、会话、Trace、审批、任务等标识
- 指标覆盖 API 流量、设备激活成功率、审批链路、工具执行、审计上报等核心路径
- 关键链路具备 Trace 或等价关联能力
- 审批、执行、回滚、分发、导出等关键动作都必须进入审计链
- 桌面端支持脱敏后的诊断包导出

方案定义的首版非功能目标：

- 单实例平台支持 `1000` 台已注册设备
- 支持 `200` 台同时在线设备
- 支持 `50` 个同时活跃会话
- 审计在线查询保留 `180` 天
- 审计归档保留 `1` 年
- 常规引导包构建目标不超过 `5` 分钟
- `RTO <= 4h`
- `RPO <= 15m`

## 安全模型

EnvNexus 不是传统意义上的"任意远控软件"。

方案中固定的核心安全约束：

- 默认产品形态下不暴露任意 Shell
- 只允许结构化工具执行
- 所有动作都必须先经过策略评估
- 高风险动作必须经过审批
- 审计采用 append-only 语义
- 数据库、缓存、任务、对象存储都必须执行租户隔离

## 文档

- 主方案文档：[`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)

## 许可证

参见 [`LICENSE`](LICENSE)。
