# EnvNexus

[English Version](README.md)

EnvNexus 是一个面向环境治理、安全本地诊断与引导式修复的 AI 原生平台。它将多租户平台、桌面客户端和本地执行内核组合在一起，提供租户专属分发、策略驱动诊断、审批式修复以及端到端审计能力。

## 项目状态

当前仓库处于“方案先行”阶段。

- 当前主规格文档为 [`docs/envnexus-proposal.md`](docs/envnexus-proposal.md)
- 项目实现骨架尚未完整生成
- 下文中的部署方式和运行方式，描述的是方案中定义的目标落地形态

## 项目要解决什么问题

EnvNexus 面向这样一类场景：用户或运维人员需要定位并修复环境问题，但又不能把产品做成“无限制远控工具”。

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

- 平台层：负责租户、模型、策略、下载链接、设备注册、审计查询和包管理元数据
- 终端层：负责本地诊断、审批式修复、审计采集和安全执行
- 集成层：负责 `WebSocket`、`Webhook` 和私有网络接入

首版 MVP 的目标，是在一台平台主机和一台受管终端之间跑通完整闭环：

`登录 -> 租户配置 -> 签名下载链接 -> 首次激活 -> 设备注册 -> 只读诊断 -> 审批式低风险修复 -> 审计上报`

## 目标架构

方案已经固定首版技术选型：

- Web 控制台：`Next.js + TypeScript`
- 后端服务：`Go`
- 后端边界：`platform-api`、`session-gateway`、`job-runner`
- 桌面端界面层：`Electron + React + TypeScript`
- 本地执行内核：`Go agent-core`
- 主数据库：`MySQL 8`
- 缓存与短状态：`Redis`
- 对象存储：`MinIO`
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
agent-core (localhost API)
    +--> platform-api
    +--> session-gateway
    \--> SQLite + Files
```

## 规划中的仓库结构

方案定义的目标仓库结构如下：

```text
envnexus/
  apps/
    console-web/
    agent-desktop/
    agent-core/
  services/
    platform-api/
    session-gateway/
    job-runner/
  libs/
  deploy/
    docker/
  docs/
```

当前仓库现阶段主要包含方案文档和项目元信息，代码骨架将在后续生成。

## 首版 MVP 能力

首版实现目标包括：

- 控制台登录与租户基础配置
- `ModelProfile`、`PolicyProfile`、`AgentProfile` 管理
- 租户专属签名下载链接
- 设备首次激活、配置拉取、心跳
- 本地 UI 与 `WebSocket` 会话
- 首批只读诊断工具
- 少量审批式低风险修复动作
- 审计事件上报与查询
- 基于 `Docker Compose` 的单机部署

## 部署形态

方案固定了三种部署模式：

- `Hosted`：平台侧统一托管控制面和存储
- `Hybrid`：平台共享控制面，企业侧保留密钥或模型边界
- `Private`：客户自管完整部署，但沿用相同协议和对象模型

首个交付目标是单机 MVP 部署：

- 一台 Linux 主机运行平台侧组件
- 一台受管终端运行 `agent-core` 和 `agent-desktop`
- 平台侧通过 `Docker Compose` 启动

## 规划中的部署方式

以下内容描述的是方案中约束的目标部署方式，也就是项目代码骨架生成后的部署目标。

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

## 规划中的运行方式

项目实现后，目标运行流程如下：

1. 通过 `Docker Compose` 启动平台服务
2. 完成控制台初始化登录与租户配置
3. 配置模型、策略和 AgentProfile
4. 生成租户专属签名下载链接
5. 在受管终端下载并启动桌面 Agent
6. 完成设备首次激活并拉取远程配置
7. 通过本地 UI 或远程会话入口发起诊断
8. 所有写操作都必须先经过显式审批
9. 执行结果和审计事件回传到本地或平台侧

## 可观测性与运维

方案要求平台侧和终端侧都具备一等公民级别的可观测性：

- 结构化日志，统一携带请求、租户、设备、会话、Trace、审批、任务等标识
- 指标覆盖 API、设备激活、审批链路、工具执行、审计上报等核心路径
- 关键链路必须具备 Trace 或等价关联能力
- 审批、执行、回滚、分发、导出等关键动作都必须进入审计链
- 桌面端必须支持脱敏后的诊断包导出

方案定义的首版非功能目标包括：

- 单实例平台支持 `1000` 台已注册设备
- 支持 `200` 台同时在线设备
- 支持 `50` 个同时活跃会话
- 审计在线查询保留 `180` 天
- 审计归档保留 `1` 年
- 常规引导包构建目标不超过 `5` 分钟
- `RTO <= 4h`
- `RPO <= 15m`

## 安全模型

EnvNexus 不是传统意义上的“任意远控软件”。

方案中固定的核心安全约束包括：

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
