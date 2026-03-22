# EnvNexus 商业产品方案文档

## 0. 品牌统一方案

### 0.1 最终推荐商业产品名称

- 中文名称：`环枢`
- 英文名称：`EnvNexus`
- 英文缩写：`ENX`

推荐理由：

- `环枢` 直接表达“环境治理中枢”，比偏安全守护的命名更贴近产品核心价值
- `Env` 直接锚定环境、运行环境、系统环境与开发环境
- `Nexus` 表达连接、中枢、枢纽与治理中心，具备明显的平台感
- `ENX` 简洁、易记，适合用于产品图标、CLI、设备标识和内部模块代号

### 0.2 品牌架构命名建议

- 平台：`EnvNexus Platform`
- 终端 Agent：`EnvNexus Agent`
- 私有化版本：`EnvNexus Private`
- 企业版：`EnvNexus Enterprise`
- 云端控制台：`EnvNexus Console`
- 设备端守护进程：`enx-agent`
- 本地命令行：`enxctl`

### 0.3 Logo 设计方向

推荐 Logo 关键词：

- `Shield`
- `Halo`
- `Core`
- `Pulse`
- `Node`

推荐视觉方案：

- 以“护盾”作为基础轮廓，表达安全和守护
- 在护盾内部融入抽象的“核心节点”或“中枢线路”，表达 Agent 与环境治理
- 顶部加入一枚简化的光环或能量弧线，体现 `EnvNexus` 的平台感和统一调度能力
- 可把字母 `A` 与护盾轮廓融合，形成高识别度图形符号

推荐图形结构：

```text
            HaloArc
               |
      +-------------------+
      |   ShieldOutline   |
      |        A          |
      |     CoreNode      |
      |    /   |    \     |
      | System | Network  |
      |        | Runtime  |
      +-------------------+
```

图形关系说明：

- `ShieldOutline` 是外层主体轮廓
- `CoreNode` 位于中心
- `CoreNode` 向 `System`、`Network`、`Runtime` 三个方向延展
- `HaloArc` 位于护盾上方
- 字母 `A` 与护盾主体融合，形成品牌识别点

设计建议：

- 主图标：抽象字母 `A` + 护盾外形 + 中心节点
- 辅助图形：环形脉冲或光晕，强化“智能守护中枢”感
- 风格：简洁、硬朗、现代企业级，不做卡通化
- 颜色建议：
  - 主色：深蓝或暗钴蓝
  - 强调色：银灰、冰蓝或青色
  - 企业感版本：深蓝 + 金属银

### 0.4 对外品牌口径

对外一句话定义：

`EnvNexus is an AI-native platform for environment governance, secure local diagnosis, and guided repair.`

中文口径：

`环枢是一套面向环境治理、安全诊断与审批式修复的 AI 原生平台。`

### 0.5 名称使用建议

PRD、技术方案、商业方案统一使用以下命名：

- 正式品牌名：`EnvNexus`
- 中文品牌名：`环枢`
- 技术简称：`ENX`

后续所有文档、页面、接口、安装包、演示材料均建议统一成该命名体系。

## 1. 项目概述

目标是构建一个以 `EnvNexus` 为统一品牌的开放平台驱动跨平台 Agent 产品：管理员先在平台网站完成模型、策略和交互方式配置，平台再生成租户专属下载链接，终端用户下载后首次启动自动绑定租户配置，后续无需再手动填写模型参数或 API Key。

产品需支持：

- Windows、Linux、macOS
- 物理机、虚拟机、Docker 容器、WSL
- `Webhook`、`WebSocket`、本地直接交互
- 公有云托管、混合托管、完全私有化部署

核心原则：

- 默认只读，先诊断后修复
- 所有写操作必须显式授权
- 本地策略优先于云端策略
- 模型与密钥管理支持多模式
- 全量审计，支持回滚与追责
- 平台负责编排，终端负责安全执行

### 1.1 首个可交付目标

为了保证后续可以直接按文档生成代码并部署运行，本文档将 `首个交付目标` 固定为：

- 以 `Docker Compose` 方式部署一套单机可运行的 `EnvNexus Platform MVP`
- 以独立二进制或安装包形式运行 `EnvNexus Agent`
- 在一台平台主机和一台受管终端之间，完整跑通“登录配置 -> 下载激活 -> 设备注册 -> 只读诊断 -> 审批式低风险修复 -> 审计上报”的闭环

这意味着首版优化优先级不是“把所有最终形态都写全”，而是先把以下问题写死：

- 首版到底包含哪些进程
- 哪些服务在 MVP 中合并实现
- 哪些协议首发必须支持
- 如何初始化数据库、缓存、对象存储和配置
- 生成后的项目怎样通过一套最小 smoke test

### 1.2 首版范围

首版必须支持：

- 平台控制台登录
- 租户、模型、策略、AgentProfile 的基础配置
- 租户专属下载链接
- 设备首次激活、配置拉取、心跳上报
- 本地 UI + `WebSocket` 会话
- 首批只读诊断工具
- 首批低风险审批式修复工具
- 审计事件上报与查询
- 单机 `Docker Compose` 部署
- **控制台前端中英双语支持**

### 1.3 首版非目标

首版明确不做：

- 完整 `Webhook` 自动化闭环执行
- 高危修复动作
- `macOS` 深度适配
- 企业级批量分发、`MDM`、域控集成
- 完整 SaaS 计费体系
- 大规模多区域高可用部署

### 1.4 MVP 完成定义

只有同时满足以下条件，才能视为 `MVP 完成`：

- 平台可通过 `Docker Compose` 一键启动
- 数据库迁移和初始化数据可自动执行
- 控制台可登录并完成首批配置
- 终端 Agent 可成功激活并拉取配置
- 可发起一次诊断会话并得到结构化结果
- 可执行一次低风险修复并完成审批、执行、审计闭环
- 所有核心服务具备健康检查接口和最小日志
- 文档内定义的 smoke test 全部通过

## 2. 目标需求与设计结论

### 已确认需求

- 用户登录开放平台网站后配置模型信息
- 与用户交互需支持 `Webhook`、`WebSocket`、私有化环境、本地直接交互
- 应用程序通过平台完成配置后生成下载链接
- 后期终端用户无需再次配置模型和密钥
- 必须保证修改本地环境前给出提示
- 终端程序应尽量免安装或简易安装

### 设计结论

- 产品本质是“开放平台 + 租户化 Agent + 安全执行引擎”
- 平台必须从一开始支持三种密钥模式：`Hosted`、`Hybrid`、`Private`
- 分发方式以“租户专属下载链接”作为主路径
- 终端 Agent 不应暴露任意 Shell，必须基于结构化工具执行
- 最终形态可覆盖平台托管与私有化部署，但实施上仍应分阶段落地

## 3. 产品定位

这是一个以 `EnvNexus` 为品牌统一体的“开放平台驱动的环境治理、诊断与修复系统”，不是纯桌面工具，也不是传统远控软件。

产品定位分为三层：

- 平台层：负责租户、模型、策略、设备与交互配置
- 终端层：负责环境探测、诊断、审批式修复和审计
- 集成层：负责 `Webhook`、`WebSocket`、私有化网关与第三方系统对接

适用场景：

- 个人用户下载后自助诊断和修复
- 技术支持远程协助终端问题排查
- 企业统一管理终端 Agent 与诊断策略
- 私有化环境内的开发/运维辅助系统

## 4. 核心业务流程

图形化流程如下：

```text
PlatformUser
    |
    v
OpenPlatform
    +--> TenantConfig
    +--> ModelProfiles
    +--> PolicyProfiles
    |
    +--> SignedDownloadLink
            |
            v
        TenantAgent
            +--> LocalUI
            |
            +--> LocalAPI
                    +--> ToolRuntime --> ObserveTools --> HostState
                    +--> PolicyEngine --> ApprovalGate --> FixTools --> AuditStore
                    \--> ModelRouter
                            +--> HostedProviders
                            +--> PrivateProviders
                            \--> LocalProviders

ExternalClient --> WebSocketGateway --> OpenPlatform
ExternalClient --> WebhookGateway   --> OpenPlatform
```

主流程：

1. 平台管理员登录网站，选择租户、模型、交互方式和安全策略
2. 平台生成租户专属下载链接
3. 终端用户下载 Agent 并首次启动
4. Agent 通过签名链接完成激活、设备注册和配置拉取
5. 用户通过本地 UI、WebSocket 或外部系统事件触发诊断
6. Agent 先执行只读诊断，再给出结论与修复建议
7. 若需修改系统，必须经过本地审批与策略校验
8. 执行后记录审计事件并反馈结果到本地或平台

## 5. 总体架构

### 5.1 开放平台架构

开放平台需至少包含以下服务：

- 用户与租户管理
- 模型配置中心
- 策略配置中心
- 设备注册服务
- 下载链接签发服务
- 会话网关
- 审计与日志中心
- 私有化部署控制模块

默认技术栈：

- Web 控制台：`Next.js`
- 后端 API 与平台服务：`Go`
- 数据库：`MySQL 8`
- 缓存/队列：`Redis`
- 对象存储：安装包、审计归档、策略快照

### 5.1.1 最终技术决策

本文档最终技术方案固定如下：

- 平台控制台：`Next.js + TypeScript`
- 平台后端：`Go`
- 后端服务边界：`platform-api` + `session-gateway` + `job-runner`
- 桌面客户端界面层：`Electron + React + TypeScript`
- 本地执行内核：`Go agent-core`
- 本地通信：`HTTP + WebSocket`，仅监听 `127.0.0.1`
- 主数据库：`MySQL 8`
- 缓存与短状态：`Redis`
- 对象存储：`MinIO` 兼容接口
- Agent 本地状态存储：`SQLite + Files`
- 首版部署：`Docker Compose`

固定原则：

- 桌面 UI 与执行内核分离，不采用单进程纯前端桌面方案
- `agent-core` 保持独立二进制，不内嵌到 `Electron` 主进程中
- 平台和终端只保留 `Go + TypeScript` 两条主技术线
- 首版不引入额外语言栈，不引入超过 `3` 个后端服务

默认命名：

- `EnvNexus Console`
- `EnvNexus Gateway`
- `EnvNexus Enrollment Service`
- `EnvNexus Audit Center`

数据库选型原则：

- 主业务数据库统一采用 `MySQL 8`
- 核心业务对象采用强结构化表设计
- 工具输入输出、事件载荷、诊断快照等半结构化数据以 JSON 字段作为补充
- 缓存、分布式锁和短期状态继续使用 `Redis`

### 5.2 终端 Agent 架构

终端侧固定拆分为两个主要模块：

- `Agent Core`
  - 工具执行
  - 环境探测
  - 模型调用编排
  - 安全策略引擎
  - 审批与审计
  - 本地 API
- `Desktop Shell`
  - 聊天 UI
  - 授权弹窗
  - 诊断结果展示
  - 变更预览与回滚提示

终端默认技术栈：

- Agent Core：`Go`
- Desktop Shell：`Electron + React + TypeScript`
- 本地通信：`HTTP + WebSocket`，默认只监听 `127.0.0.1`

技术栈统一原则：

- 平台后端 3 个服务统一使用 `Go`
- 终端 Agent Core 统一使用 `Go`
- 前端控制台与桌面 UI 统一使用 `TypeScript` Web 技术栈
- 桌面端采用 `Electron` 承载 Web UI，避免在首版继续比较 `Tauri` 与 `Electron` 路线

平台与终端的对外命名：

- 下载包：`EnvNexus Agent`
- 私有化镜像：`EnvNexus Private`
- 平台后台：`EnvNexus Console`
- CLI：`enxctl`
- 设备守护进程：`enx-agent`

### 5.3 集成与接入架构

交互需抽象为统一会话层，支持：

- 本地直接交互
- `WebSocket` 实时双向对话
- `Webhook` 事件驱动调用
- 私有化环境中的内网网关接入

设计原则：

- 交互协议统一映射到同一套会话和审批模型
- 本地 Agent 不直接暴露公网端口
- 所有远程交互必须经过平台或私有化网关鉴权

### 5.4 服务拆分与职责边界

为避免平台后端在后续演进中变成单体杂糅服务，本文档将服务边界固定如下。

核心后端服务定义：

- `platform-api`
  - 面向控制台前端、设备端和管理端的统一聚合 API
  - 负责租户、用户、角色、模型配置、策略配置、下载链接、设备注册、配置拉取、审计查询、包管理元数据、Webhook 接入
- `session-gateway`
  - 负责 `WebSocket` 会话接入、流式事件转发、实时审批回传
- `job-runner`
  - 负责异步任务执行
  - 负责审计批处理、包构建与清理、Webhook 重试、定时治理任务、通知和后续可扩展后台任务

服务拆分原则：

- 控制面和配置面全部收敛到 `platform-api`
- 实时长连接职责固定放在 `session-gateway`
- 异步与定时任务职责固定放在 `job-runner`
- 首版不再继续把身份、模型、策略、注册、审计、包管理拆成独立微服务
- 所有 Agent 面向平台的协议尽量统一到稳定的注册、拉配、上报、审批接口

### 5.5 部署形态与运行边界

部署层次固定为三种：

- 单体开发模式
  - 适合 MVP 和本地联调
  - 多服务逻辑可在单进程中以模块方式运行
- 标准云部署模式
  - 控制台、`platform-api`、`session-gateway`、`job-runner` 分离部署
  - 适合 SaaS 运营
- 私有化部署模式
  - 按同一服务协议部署到客户环境
  - 支持裁剪不需要的云端运营模块

运行边界：

- `Go` 服务之间通过内部 REST 或 gRPC 通信
- 对 Agent 暴露稳定的 HTTPS + WebSocket 接口
- 审计、策略和注册服务都必须支持独立扩容
- 私有化与公有云共用同一份协议和对象模型，避免能力分叉

### 5.5.1 单机部署 MVP 形态

为了让后续代码可以直接生成并运行，首版默认部署形态固定为：

- 一台 `Linux` 主机承载平台侧全部云端组件
- 一个独立的终端设备运行 `EnvNexus Agent`
- 平台侧使用 `Docker Compose`
- 终端侧运行 `agent-core` + `agent-desktop`

首版平台侧最小进程集合：

- `console-web`
  - `Next.js` 控制台
- `platform-api`
  - 逻辑合并租户、身份、模型、策略、注册、下载链接、审计查询、Webhook 接入和包管理元数据
- `session-gateway`
  - 独立负责 `WebSocket` 会话与流式事件
- `job-runner`
  - 独立负责异步任务和定时任务
- `mysql`
- `redis`
- `minio`

说明：

- `Webhook` 接入能力在首版并入 `platform-api`
- 包管理元数据在首版并入 `platform-api`
- 审计批处理、包构建和清理任务统一进入 `job-runner`
- 私有化部署保留相同 3 服务边界，不额外拆更多微服务

### 5.5.2 单机部署拓扑

图形化拓扑如下：

```text
AdminBrowser
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
agent-core-local-api
    +--> platform-api
    +--> session-gateway
    \--> SQLite + Files
```

默认端口：

- `console-web`: `3000`
- `platform-api`: `8080`
- `session-gateway`: `8081`
- `job-runner`: 无公网端口，仅内部运行
- `mysql`: `3306`
- `redis`: `6379`
- `minio-api`: `9000`
- `minio-console`: `9001`
- `agent-core local api`: `127.0.0.1:17700`

### 5.5.3 MVP 服务合并策略

为了提高代码生成成功率，首版遵循以下合并原则：

- 所有平台配置类和管理类能力统一合并到 `platform-api`
- 长连接事件与会话流转固定放在 `session-gateway`
- 所有异步任务和定时任务统一放在 `job-runner`
- 对象存储先统一通过一个抽象接口访问，默认落 `MinIO`
- 审计查询放在 `platform-api`，审计批处理与归档任务放在 `job-runner`

拆分触发条件：

- `WebSocket` 连接数和流量成为瓶颈时，只扩容 `session-gateway`
- 异步任务量上升时，只扩容 `job-runner`
- 首版之后如非必要，不再把后端拆超过 `3` 个服务

### 5.5.4 运行目录与持久化边界

平台侧数据卷固定如下：

- `./volumes/mysql`
- `./volumes/redis`
- `./volumes/minio`
- `./volumes/platform/logs`
- `./volumes/platform/packages`
- `./volumes/platform/audits`

终端侧本地目录固定如下：

- `config/`
- `data/`
- `logs/`
- `cache/`

任何自动生成代码都必须保证：

- 服务重启后平台配置不丢失
- 设备注册记录不丢失
- 审计记录和审批记录不因进程重启丢失
- Agent 本地身份和未上报审计事件可恢复

### 5.5.5 配置与环境变量规范

所有服务必须支持：

- `.env.example`
- 开发环境 `.env.local`
- 部署环境 `.env`
- 通过环境变量覆盖配置文件

首批必须统一的环境变量：

- `ENX_APP_ENV`
- `ENX_HTTP_PORT`
- `ENX_PUBLIC_BASE_URL`
- `ENX_WS_PUBLIC_URL`
- `ENX_DATABASE_DSN`
- `ENX_REDIS_ADDR`
- `ENX_OBJECT_STORAGE_ENDPOINT`
- `ENX_OBJECT_STORAGE_ACCESS_KEY`
- `ENX_OBJECT_STORAGE_SECRET_KEY`
- `ENX_OBJECT_STORAGE_BUCKET`
- `ENX_JWT_SECRET`
- `ENX_DEVICE_TOKEN_SECRET`
- `ENX_ENROLL_SIGNING_SECRET`
- `ENX_AUDIT_ENCRYPTION_KEY`
- `ENX_LOG_LEVEL`

配置规则：

- 所有密钥必须来源于环境变量或外部注入，不写死在仓库
- 所有服务必须打印配置摘要，但严禁打印明文密钥
- 所有 URL 类配置必须明确区分内部地址和公网访问地址

`.env.example` 示例：

```dotenv
ENX_APP_ENV=local
ENX_HTTP_PORT=8080
ENX_PUBLIC_BASE_URL=http://localhost:8080
ENX_WS_PUBLIC_URL=ws://localhost:8081
ENX_DATABASE_DSN=root:root@tcp(mysql:3306)/envnexus?parseTime=true&charset=utf8mb4
ENX_REDIS_ADDR=redis:6379
ENX_OBJECT_STORAGE_ENDPOINT=minio:9000
ENX_OBJECT_STORAGE_ACCESS_KEY=minioadmin
ENX_OBJECT_STORAGE_SECRET_KEY=minioadmin
ENX_OBJECT_STORAGE_BUCKET=envnexus-local
ENX_JWT_SECRET=change-me-jwt-secret
ENX_DEVICE_TOKEN_SECRET=change-me-device-secret
ENX_ENROLL_SIGNING_SECRET=change-me-enroll-secret
ENX_AUDIT_ENCRYPTION_KEY=change-me-audit-key
ENX_LOG_LEVEL=info
```

`runtime.yaml` 示例：

```yaml
app:
  env: local
  log_level: info

server:
  http_port: 8080
  public_base_url: http://localhost:8080
  ws_public_url: ws://localhost:8081

database:
  dsn: ${ENX_DATABASE_DSN}

redis:
  addr: ${ENX_REDIS_ADDR}

object_storage:
  endpoint: ${ENX_OBJECT_STORAGE_ENDPOINT}
  bucket: ${ENX_OBJECT_STORAGE_BUCKET}

security:
  jwt_secret: ${ENX_JWT_SECRET}
  device_token_secret: ${ENX_DEVICE_TOKEN_SECRET}
  enroll_signing_secret: ${ENX_ENROLL_SIGNING_SECRET}
  audit_encryption_key: ${ENX_AUDIT_ENCRYPTION_KEY}
```

## 6. 模型与密钥管理

平台必须从第一天支持三种模式。

### 6.1 Hosted 模式

- 模型配置和 API Key 由平台托管
- 终端用户下载后无需任何模型配置
- 适合公有云 SaaS 场景

### 6.2 Hybrid 模式

- 平台保存 provider、endpoint、model、策略等非敏感配置
- 密钥由企业环境、本地环境或私有网关注入
- 适合客户希望保留密钥控制权的场景

### 6.3 Private 模式

- 平台可整体私有化部署
- 模型配置与密钥都不经过公有云
- 适合内网、政企或高合规场景

### 6.4 模型配置对象

`ModelProfile` 必须至少包含以下字段：

- provider 类型
- base URL
- model 名称
- 推理参数
- fallback 模型
- 预算限制
- 超时策略
- 可用工具范围
- 密钥来源模式

首版 Provider 范围固定为：

- OpenAI-compatible
- Anthropic
- Gemini
- OpenRouter
- SiliconFlow
- DeepSeek
- Ollama
- 企业私有模型网关

约束：

- 首版必须至少实现 `OpenAI-compatible`、`OpenRouter`、`Ollama` 三种 provider 适配
- 其他 provider 可以先保留接口与配置枚举，但不得影响首版运行闭环
- 所有 provider 适配层必须走统一 `llm/router` 抽象，不允许在业务层直接拼接第三方请求

`ModelProfile` JSON 示例：

```json
{
  "id": "01JMODEL0001ABCDEFGHJKMNPQ",
  "tenant_id": "01JTENANT0001ABCDEFGHJKMNPQ",
  "name": "default-openrouter",
  "provider": "openrouter",
  "base_url": "https://openrouter.ai/api/v1",
  "model_name": "openai/gpt-4.1",
  "params_json": {
    "temperature": 0.2,
    "max_tokens": 2048
  },
  "secret_mode": "hosted",
  "status": "active",
  "version": 1
}
```

`PolicyProfile` JSON 示例：

```json
{
  "id": "01JPOLICY001ABCDEFGHJKMNPQ",
  "tenant_id": "01JTENANT0001ABCDEFGHJKMNPQ",
  "name": "default-safe-policy",
  "policy_json": {
    "default_mode": "read_only",
    "allow_write_tools": true,
    "require_approval_levels": ["L1", "L2", "L3"],
    "blocked_tools": ["shell.exec_any"],
    "allowed_repair_tools": [
      "dns.flush_cache",
      "service.restart",
      "cache.rebuild"
    ]
  },
  "status": "active",
  "version": 1
}
```

`AgentProfile` JSON 示例：

```json
{
  "id": "01JAGENT0001ABCDEFGHJKMNPQ",
  "tenant_id": "01JTENANT0001ABCDEFGHJKMNPQ",
  "name": "windows-standard",
  "model_profile_id": "01JMODEL0001ABCDEFGHJKMNPQ",
  "policy_profile_id": "01JPOLICY001ABCDEFGHJKMNPQ",
  "capabilities_json": {
    "local_ui": true,
    "websocket_session": true,
    "webhook_trigger": false,
    "repair_enabled": true
  },
  "update_channel": "stable",
  "status": "active",
  "version": 1
}
```

## 7. 下载、激活与设备注册

### 7.1 租户专属下载链接

平台生成的下载链接必须具备：

- 租户绑定
- 有效期
- 可撤销
- 签名校验
- 支持按租户或链接策略控制的一次性首次激活能力

不允许直接在链接中包含明文 `API Key` 或其他长期敏感密钥。

### 7.2 首次激活流程

1. 用户点击租户专属下载链接下载 Agent
2. Agent 启动后读取内置或附带的租户引导信息
3. Agent 与平台注册服务完成激活握手
4. 平台下发设备令牌、策略和模型配置
5. Agent 完成本地安全初始化并进入可用状态

### 7.3 激活后身份

首次激活后统一切换为长期设备身份：

- 设备证书
- 长期设备令牌
- 平台侧设备记录

这样可以支持：

- 后续配置同步
- 审计归属
- 远程协助授权
- 设备禁用与撤销

### 7.4 设备生命周期状态机

设备生命周期需要显式状态机，否则设备注册、撤销、重装、迁移会在后续变得混乱。

设备状态固定为：

- `downloaded`
- `bootstrapping`
- `pending_activation`
- `active`
- `policy_outdated`
- `quarantined`
- `revoked`
- `retired`

设备状态流转固定为：

```text
downloaded
    |
    v
bootstrapping
    |
    v
pending_activation
    |
    v
active
  +--> policy_outdated --> active
  +--> quarantined    --> active
  \--> revoked        --> retired
```

关键规则：

- `revoked` 设备不得继续拉取新配置或建立会话
- `quarantined` 设备只允许最小诊断和恢复流程
- 设备重装不应复用原长期身份，必须重新激活或完成迁移确认

### 7.5 安装包与更新模型

为了让项目后续可以直接部署运行，首版必须明确安装产物：

- 平台侧：
  - `docker-compose.yml`
  - `.env.example`
  - 初始化脚本或 migration job
- 终端侧：
  - `Windows` 安装包或压缩包
  - `Linux` 压缩包与 `systemd` 示例配置

安装包中允许包含：

- 平台地址
- 租户引导信息
- 一次性激活 token 或签名激活信息
- 版本号与校验摘要

安装包中禁止包含：

- 长期 `API Key`
- 长期设备凭证
- 明文管理员账号信息

更新策略：

- 首版默认手动更新或控制台提示更新
- 自动更新能力只保留对象模型和接口，不作为首版必须实现
- 每次升级必须支持数据库 migration 与最小回滚说明

### 7.6 租户专属分发包设计

当前方案明确支持“多租户 + 租户专属分发应用”的目标，但实现方式必须固定为：

- 平台不是只生成一个通用下载链接
- 平台要基于租户配置生成 `DownloadPackage`
- 每个 `DownloadPackage` 都绑定唯一 `tenant_id`
- 下载链接必须指向某个具体的租户专属分发包或租户专属引导包

#### 7.6.1 支持的分发模式

首版固定支持两种模式：

1. `bootstrap_package`
   - 使用统一应用二进制
   - 通过租户专属引导配置、签名激活信息、品牌资源覆盖实现“租户专属分发”
   - 这是首版默认模式

2. `branded_package`
   - 为某个租户生成独立安装包产物
   - 可替换应用名、Logo、启动页、默认平台地址、发布描述
   - 这是第二阶段增强模式，但对象模型首版就必须预留

首版约束：

- 首版必须先实现 `bootstrap_package`
- `branded_package` 在首版可以先保留构建协议和字段，不强制实现真实重新打包
- 无论哪种模式，下载链接都必须体现租户隔离与可撤销能力

#### 7.6.2 租户专属分发的生成流程

```text
TenantConfig
    |
    +--> ModelProfile
    +--> PolicyProfile
    +--> AgentProfile
    +--> BrandingAssets
    |
    v
DownloadPackage
    +--> PackageArtifact
    +--> EnrollmentToken
    +--> SignedDownloadLink
```

流程说明：

1. 平台管理员在租户下完成模型、策略、AgentProfile 和品牌资源配置
2. 平台创建一条 `DownloadPackage` 记录
3. `job-runner` 依据包模式生成引导包或租户定制包
4. 平台生成与该包绑定的 `EnrollmentToken`
5. 平台签发 `SignedDownloadLink`
6. 用户下载后首次启动，Agent 使用包内引导信息和激活令牌完成租户绑定

#### 7.6.3 租户专属分发包允许差异化的内容

租户包允许差异化的内容固定为：

- `app_display_name`
- `package_name`
- `logo_asset`
- `splash_asset`
- `default_platform_base_url`
- `default_ws_url`
- `tenant_bootstrap_manifest`
- `release_channel`
- `version_label`

首版禁止差异化的内容：

- 核心执行权限模型
- 审批状态机逻辑
- 核心协议格式
- 高危工具白名单

解释：

- 租户分发包必须支持独立外观和默认接入配置
- 但不能通过租户分发包绕过平台统一安全边界

#### 7.6.4 下载链接与安装包的映射规则

下载链接必须满足：

- 一条链接必须明确指向一个 `DownloadPackage`
- 一条链接必须只属于一个 `tenant_id`
- 一个 `DownloadPackage` 必须支持签发多个临时下载链接
- 一个下载链接必须支持绑定一个或多个使用次数受限的 `EnrollmentToken`
- 链接撤销后不得继续下载新包
- 激活令牌过期或耗尽后，旧包仍可打开，但不得完成新设备激活

固定映射关系：

- `tenant_id -> agent_profile_id -> download_package_id -> signed_download_link`
- `download_package_id -> enrollment_token`

#### 7.6.5 品牌资源与应用名规则

为了支撑“私有应用”体验，租户级品牌资源必须支持：

- 控制台中上传品牌图标
- 设置应用显示名称
- 设置发行说明摘要
- 设置安装包文件名前缀

命名规则固定为：

- Windows 安装包：
  - `envnexus-{tenant_slug}-{channel}-{version}-windows-{arch}.zip`
  - 或 `envnexus-{tenant_slug}-{channel}-{version}-setup.exe`
- Linux 压缩包：
  - `envnexus-{tenant_slug}-{channel}-{version}-linux-{arch}.tar.gz`

如果启用 `branded_package`，则允许文件名前缀替换为租户品牌名的 slug 版本。

#### 7.6.6 包构建与签名规则

首版包构建规则：

- `job-runner` 负责构建或装配分发包
- 所有分发包都必须生成 `checksum`
- 所有公开下载包都必须记录 `build_version`
- 所有下载包都必须记录构建来源模板版本

签名规则：

- Windows 包必须支持代码签名字段预留
- Linux 包必须至少提供校验摘要
- 首版如果无法真实完成平台级代码签名，也必须预留 `sign_status` 和 `sign_metadata`

#### 7.6.7 回滚与版本策略

租户专属分发必须支持：

- 每个租户保留最近若干版本包记录
- 可以把某个租户的默认下载目标回退到旧版 `DownloadPackage`
- 新链接只指向当前默认版本
- 旧链接是否继续有效由租户策略决定

回滚最小规则：

- 不删除历史 `DownloadPackage`
- 不硬删除历史 `EnrollmentToken`
- 回滚只改变“默认分发目标”，不篡改历史审计

#### 7.6.8 首版必须补齐的对象字段

为了真正支持“租户专属分发应用”，首版必须把以下字段写入对象模型。

`download_packages` 必须新增：

- `distribution_mode`
- `package_name`
- `artifact_path`
- `artifact_size`
- `bootstrap_manifest_json`
- `branding_version`
- `build_version`
- `sign_status`
- `sign_metadata_json`
- `published_at`

`enrollment_tokens` 必须新增：

- `download_package_id`
- `channel`
- `issued_by_user_id`

#### 7.6.9 首版实现边界

首版必须做到：

- 多租户下能为不同租户生成不同下载链接
- 下载链接绑定租户与 `AgentProfile`
- 包内能注入租户引导信息
- 首次激活后自动绑定正确租户配置
- 平台能撤销某个租户的下载链接和激活令牌

首版可以暂缓：

- 真正的每租户独立重新打包
- 每租户独立桌面图标和安装器资源编译
- 自动更新通道的租户级差异化策略
- 多平台并行构建集群

#### 7.6.10 包构建供应链安全约束

为了确保“租户专属分发应用”不会变成供应链风险入口，分发包构建安全约束固定如下：

- 所有构建任务必须记录：
  - `tenant_id`
  - `download_package_id`
  - `build_version`
  - `source_template_version`
  - `triggered_by_user_id`
- 所有构建产物必须保留：
  - `checksum`
  - `artifact_path`
  - `build_log_reference`
  - `sign_status`
- 构建模板、品牌资源、引导清单必须带版本号，不允许直接覆盖历史版本
- 品牌资源更新不得隐式影响已发布包，变更后必须生成新的 `DownloadPackage`
- 构建结果必须可审计、可重放、可回滚
- 不允许手工替换租户构建产物而不留下审计记录

首版最小安全要求：

- 构建任务只能由具备发布权限的用户触发
- 构建日志必须可追溯到租户和操作人
- 构建失败必须记录失败原因
- 已发布构建产物必须只读存储

## 8. 安全模型

### 8.1 基本原则

- 默认只读
- 禁止模型拥有任意 Shell
- 工具与权限必须白名单化
- 所有写操作均需审批
- 本地策略优先于云端策略
- 全量记录审批与执行证据

### 8.2 风险分级

- `L0`：只读操作，无需确认
- `L1`：低风险写操作，需要普通确认
- `L2`：中风险修改，需要强确认和变更预览
- `L3`：高危动作，默认禁用，仅实验模式或私有化高级策略下开启

### 8.3 审批模型

审批必须在执行层完成，而不是仅靠提示词约束。

每次变更请求至少要展示：

- 变更目的
- 诊断依据
- 影响范围
- 风险等级
- 预计结果
- 回滚策略
- 执行命令或结构化动作摘要

### 8.4 关键约束

禁止在默认产品形态中直接开放：

- 任意 shell 命令执行
- 下载并执行外部脚本
- 静默修改系统配置
- 云端绕过本地审批
- 无感知远程控制

### 8.5 策略优先级模型

为了避免 Hosted、Hybrid、Private 三种模式下出现权限混乱，策略优先级固定如下：

1. 本地硬限制
2. 本地管理员策略
3. 租户策略
4. 项目或 AgentProfile 策略
5. 会话级临时策略
6. 模型建议

解释：

- 模型永远不能提升权限，只能在既有策略范围内建议动作
- 会话级策略只能收紧权限，不能突破本地硬限制
- 私有化环境允许替换云端策略源，但不改变优先级模型

### 8.6 审批流状态机

审批必须形成可追踪状态机，避免“弹窗确认”变成不可审计的临时行为。

审批状态固定为：

- `drafted`
- `pending_user`
- `approved`
- `denied`
- `expired`
- `executing`
- `succeeded`
- `failed`
- `rolled_back`

```text
drafted
    |
    v
pending_user
  +--> approved --> executing --> succeeded
  |                    |
  |                    \--> failed --> rolled_back
  +--> denied
  \--> expired
```

审批记录关键字段：

- 请求摘要
- 风险等级
- 审批人
- 审批时间
- 策略快照版本
- 执行前环境摘要
- 执行结果
- 回滚结果

### 8.7 首版认证与密钥落地规范

首版实现时必须把以下认证边界固定下来：

- 控制台用户：
  - 使用 `JWT` 或等价 session token
  - 必须区分 access token 与 refresh token 生命周期
- 设备：
  - 首次激活后获得长期 device token
  - device token 必须可撤销、可轮换
- `WebSocket`：
  - 使用短期 session token 建连
  - 建连后所有事件都要绑定 `tenant_id`、`device_id`、`session_id`

密钥落地规则：

- 平台托管密钥默认只存平台侧，不下发终端明文
- `Hybrid` 模式下终端只接收非敏感模型配置，密钥由本地或企业环境注入
- Agent 本地敏感信息必须放系统密钥链或本地加密存储
- 所有签名密钥、token secret、审计加密密钥都必须支持轮换

审计安全规则：

- 审计事件采用 append-only 语义
- 审批结果、执行结果、回滚结果必须形成同一条可追踪链
- 任何高风险动作都必须能追溯到审批单和会话

### 8.8 RBAC 与关键操作权限边界

首批角色固定为：

- `platform_super_admin`
- `tenant_admin`
- `security_auditor`
- `ops_operator`
- `read_only_observer`

首批关键操作权限边界固定如下：

- `platform_super_admin`
  - 管理全局平台配置
  - 查看所有租户概览
  - 处理跨租户平台级故障
- `tenant_admin`
  - 管理当前租户的模型、策略、AgentProfile
  - 生成租户专属下载链接
  - 发布或回滚租户分发包
  - 撤销租户设备
- `security_auditor`
  - 查看当前租户全部审计
  - 查看审批记录
  - 不得修改策略和发包
- `ops_operator`
  - 处理设备运维
  - 发起或批准低风险操作
  - 不得修改核心租户安全策略
- `read_only_observer`
  - 只读查看设备、会话、审计摘要
  - 不得生成下载链接、不得审批、不得发包

硬性约束：

- 生成下载链接、发布分发包、撤销设备必须是显式权限
- 审计查看权限与配置修改权限必须分离
- 高风险审批权限不能默认授予 `ops_operator`
- 平台级角色权限不得自动下沉为租户内发包权限

### 8.9 数据治理、脱敏与审计导出规则

为了让产品后续可以面向企业客户交付，首版必须把数据治理边界固定下来，而不能仅停留在“有审计”和“有隔离”。

数据分级固定如下：

- `P0`：密钥材料与认证凭据
  - 例如：`JWT secret`、设备令牌签名密钥、对象存储密钥、第三方模型密钥
  - 默认不得出现在业务日志、审计导出和前端响应体中
- `P1`：敏感业务数据
  - 例如：用户邮箱、设备标识、主机名、IP、命令输出中的凭据片段、品牌包下载令牌
  - 默认需要脱敏展示，明文查看必须受权限控制
- `P2`：普通业务数据
  - 例如：策略名称、模型名称、构建版本、审批状态、错误码
  - 用于运营和排障，但仍需受租户隔离约束
- `P3`：公开或低敏元数据
  - 例如：产品版本、平台环境、公开帮助链接
  - 只用于公开文档和非敏感界面

日志脱敏规则固定如下：

- 任何日志、Trace、错误快照中不得输出完整 token、secret、密码、签名串
- 邮箱、手机号、设备序列号、主机名、IP 默认按“部分可识别”规则脱敏
- 工具执行结果如包含环境变量、命令参数、证书片段、访问地址，必须先脱敏后写入日志与审计
- 长文本输出必须裁剪长度上限，避免把整段终端输出直接入库
- 前端和桌面端默认只展示脱敏后的错误详情，明文原文只允许具备授权的审计角色导出

审计导出规则固定如下：

- 审计导出必须是显式操作，不允许在列表页隐式批量下载
- 审计导出只允许 `tenant_admin` 和 `security_auditor`
- 导出动作必须记录：
  - `tenant_id`
  - `operator_user_id`
  - 导出时间范围
  - 过滤条件摘要
  - 导出文件摘要值
  - 导出原因
- 审计导出文件必须带生成时间、租户标识和导出摘要，便于后续追溯
- 涉及高风险审批、设备撤销、分发包发布、回滚的审计导出，不得跳过留痕

保留、冻结、删除规则固定如下：

- 审计事件、审批记录、分发包构建记录默认不可物理删除
- 租户进入 `suspended` 后，只允许查询与导出，不允许新增会改变证据链的写操作
- 租户进入 `archived` 后，仅保留查询、归档和合规导出能力
- 用户发起“删除”时，如与审计保留义务冲突，必须优先满足保留义务，再执行逻辑删除或脱敏归档
- 本地 Agent 缓存的敏感数据必须按最小保留原则存储，不得长期保留过期令牌和明文审批上下文

## 9. 工具系统与执行模型

本地执行必须采用结构化工具调用，而不是模型自由拼接命令。

每个工具都必须提供以下元数据：

- `name`
- `platformSupport`
- `environmentSupport`
- `readOnly`
- `riskLevel`
- `requiresApproval`
- `requiredPrivileges`
- `rollbackSupported`
- `estimatedImpact`

### 9.1 只读诊断工具

首批只读工具固定为：

- 网络接口状态
- DNS 配置与连通性
- 代理配置检查
- 路由表读取
- 端口监听检查
- 服务状态读取
- 系统版本与权限状态
- 磁盘/CPU/内存健康信息
- Docker 容器状态
- 指定日志读取
- 环境变量与开发工具版本检查

### 9.2 审批式修复工具

第二阶段才允许开放的修复工具：

- 刷新 DNS 缓存
- 重启指定服务
- 打开/关闭应用层代理
- 修改已知配置字段
- 重建缓存目录
- 重载容器或进程级配置

## 10. 运行环境适配

Agent 启动后应探测：

- 物理机
- 虚拟机
- Docker 容器
- WSL
- root/admin 权限状态
- systemd/service manager 可用性

不同环境只暴露合理能力。例如：

- 容器中禁用整机级操作
- 非管理员权限不展示系统级修复
- 私有化环境支持内网模型地址
- 虚拟机可开放实验性高级工具

## 11. 平台对象模型

首批核心对象固定为：

- `Tenant`
- `User`
- `Role`
- `AgentProfile`
- `ModelProfile`
- `PolicyProfile`
- `DownloadPackage`
- `EnrollmentToken`
- `Device`
- `Session`
- `ToolInvocation`
- `ApprovalRequest`
- `AuditEvent`

这些对象足以支撑：

- 多租户管理
- 模型与策略配置
- 下载链接签发
- 设备生命周期
- 会话与交互协议
- 审批和审计

命名统一规范：

- 设备 ID 前缀：`enx_dev_`
- 会话 ID 前缀：`enx_sess_`
- 审批单号前缀：`enx_appr_`
- 审计事件前缀：`enx_evt_`

### 11.1 对象职责分层

为了防止对象越来越多但语义混乱，本文档固定采用三层对象职责划分：

- 平台配置层
  - `Tenant`
  - `User`
  - `Role`
  - `ModelProfile`
  - `PolicyProfile`
  - `AgentProfile`
- 设备运行层
  - `Device`
  - `Session`
  - `EnrollmentToken`
  - `DownloadPackage`
- 执行与审计层
  - `ToolInvocation`
  - `ApprovalRequest`
  - `AuditEvent`

这样可以让数据库、API 和权限边界更清晰：

- 配置层决定“能做什么”
- 运行层描述“谁在运行”
- 执行层记录“实际做了什么”

### 11.2 会话状态机

会话系统是 `WebSocket`、本地交互、远程协助的共同基础，状态机固定如下。

会话状态固定为：

- `created`
- `attached`
- `diagnosing`
- `awaiting_approval`
- `executing`
- `completed`
- `aborted`
- `expired`

```text
created
    |
    v
attached
  +--> diagnosing --> awaiting_approval --> executing --> completed
  |                      |                    |
  |                      \------------------> aborted
  \-----------------------------------------> expired
```

关键规则：

- 一个会话在任一时刻只能有一个进行中的高风险审批
- 会话结束后，所有审批与工具调用必须可回溯到该会话
- `Webhook` 触发的无界面会话不得执行需要本地确认的动作，除非进入人工接管模式

### 11.3 环境治理闭环定义

为了让产品不只是“有问题再修”，而是真正体现环境治理，治理闭环固定如下：

1. 采集
  - 周期性或事件驱动收集环境状态、配置和健康指标
2. 基线
  - 定义期望状态、版本范围、策略和允许偏差
3. 检测
  - 比较当前状态与期望基线，识别异常与漂移
4. 诊断
  - 使用工具与模型解释异常原因和影响范围
5. 决策
  - 输出建议、修复动作、风险等级和审批需求
6. 执行
  - 在审批后执行受控修复
7. 验证
  - 再次采集并确认问题是否消除
8. 沉淀
  - 将案例、策略、基线调整和审计结果沉淀为治理资产

产品路线中的治理对象固定为三类：

- 运行环境治理
  - 网络、代理、端口、服务、容器、系统服务
- 开发环境治理
  - 语言运行时、依赖管理、环境变量、工具链版本
- 策略治理
  - 权限边界、审批策略、模型使用范围、预算和风险控制

### 11.4 公有云与私有化能力对齐原则

为防止后续私有化版本变成完全不同的产品，对齐规则固定如下。

必须一致的部分：

- 核心对象模型
- Agent 与平台协议
- 审批流状态机
- 审计事件结构
- 策略优先级模型

允许裁剪的部分：

- SaaS 计费与租户运营模块
- 公共下载分发模块
- 公有云托管密钥模式
- 某些第三方集成与市场化功能

私有化版本额外增强项：

- 内网 IdP 对接
- 内网对象存储
- 企业模型网关
- 本地密钥托管或 HSM/KMS
- 离线升级包与离线审计归档

### 11.5 租户生命周期模型

租户生命周期固定为：

- `draft`
- `active`
- `suspended`
- `archived`

生命周期规则：

- `draft`
  - 允许创建基础配置
  - 不允许发包和激活设备
- `active`
  - 允许完整使用产品能力
- `suspended`
  - 禁止新发包
  - 禁止新设备激活
  - 允许审计查询和数据导出
- `archived`
  - 不允许写操作
  - 仅保留审计和归档查询能力

首批租户初始化流程固定如下：

1. 创建租户
2. 创建默认租户管理员
3. 初始化默认 `ModelProfile`
4. 初始化默认 `PolicyProfile`
5. 初始化默认 `AgentProfile`
6. 初始化默认品牌资源占位
7. 进入 `active` 前完成最小配置校验

### 11.6 多租户隔离规则

多租户隔离规则固定如下：

- 所有业务主表必须带 `tenant_id`
- 所有高价值查询默认带 `tenant_id` 过滤
- 所有对象存储路径必须按租户分层，例如：
  - `tenants/{tenant_id}/packages/...`
  - `tenants/{tenant_id}/audits/...`
- 所有缓存 key 必须包含 `tenant_id`
- 所有异步任务 payload 必须包含 `tenant_id`
- 所有审计查询默认只能查看当前租户数据
- 不允许跨租户引用 `Device`、`Session`、`DownloadPackage`、`EnrollmentToken`

首版隔离策略固定为：

- 数据库采用逻辑隔离
- 对象存储采用路径隔离
- 缓存与任务采用 key / payload 隔离
- 审计查询采用权限 + `tenant_id` 双重过滤

保留演进能力：

- 后续允许对高合规租户增强为物理隔离数据库或独立对象存储
- 但对象模型、API 协议和租户主键语义保持不变

### 11.7 数据驻留与责任边界

为了避免 Hosted、Hybrid、Private 三种模式下责任混乱，数据驻留与责任边界固定如下：

- `Hosted`
  - 平台侧负责主数据库、对象存储、缓存、审计归档、备份恢复
  - 租户负责其品牌资源、模型接入配置和业务使用合规性
- `Hybrid`
  - 平台侧负责平台元数据、设备注册、审计聚合和公共控制平面
  - 企业侧负责本地密钥注入、本地模型访问控制和企业内网数据合规
- `Private`
  - 客户负责数据库、对象存储、缓存、审计、备份、密钥与运维
  - 产品方负责交付相同对象模型、协议与升级规则，不接管客户数据责任

驻留规则固定如下：

- 审计归档、分发包产物、策略快照必须与其所属部署模式的存储边界保持一致
- `Private` 模式下不得要求客户把审计明文回传到公有云
- `Hybrid` 模式下终端侧采集的数据默认优先保留在企业边界内，平台仅接收必要摘要和治理结果
- 任意模式下，跨区域复制、跨主体共享、第三方转储都必须有显式配置和审计留痕

## 12. MVP 与最终形态拆分

### 12.1 最终形态蓝图

- 开放平台控制台
- Hosted / Hybrid / Private 三种模式
- 租户专属下载链接
- Agent 自动注册与配置同步
- 本地 UI + `WebSocket` + `Webhook`
- 审批式修复
- 私有化部署
- 全量审计与设备管理

### 12.2 第一阶段 MVP

第一阶段必须先做：

- 开放平台基础版
- `Hosted` 和 `Hybrid` 两种模式
- Windows + Linux Agent
- 本地 UI
- 本地只读诊断工具
- 风险分级与审批框架
- `WebSocket` 会话
- 租户专属下载链接和首次激活

暂缓：

- `Webhook`
- macOS 深度支持
- 完整私有化发行版
- 高危修复动作
- 企业批量分发与 MDM

### 12.2.1 MVP 首发能力清单

为了避免后续一次性生成出不可运行的大项目，首版必须只实现以下最小闭环：

- 平台登录与租户基础管理
- `ModelProfile`、`PolicyProfile`、`AgentProfile` 增删改查
- 下载链接签发
- 设备激活、配置拉取、心跳
- 本地诊断会话
- `WebSocket` 流式事件
- 首批只读工具
- 不超过 `3` 个低风险修复工具
- 审批流和审计上报
- 设备列表与审计列表查询

首批低风险修复工具固定为：

- 刷新 DNS 缓存
- 重启指定服务
- 重建指定缓存目录

### 12.2.2 MVP 非目标清单

首版生成代码时明确不实现：

- `Webhook` 自动触发执行闭环
- 多租户计费
- `macOS` 正式安装包
- 高危系统配置改写
- 任意 Shell 透传
- 多区域部署、高可用主从切换
- 完整对象存储生命周期治理

### 12.2.3 MVP 最小可运行进程集合

首版代码生成完成后，必须至少生成并能跑通以下组件：

- `apps/console-web` (需包含中英双语支持)
- `apps/agent-desktop`
- `apps/agent-core`
- `services/platform-api`
- `services/session-gateway`
- `services/job-runner`
- `deploy/docker/docker-compose.yml`

其中：

- `platform-api` 作为平台侧聚合服务承担大部分业务职责
- `session-gateway` 独立承担 `WebSocket` 与实时会话职责
- `job-runner` 承担所有异步和定时任务职责
- 首版后端服务总数不得超过 `3` 个

### 12.2.4 版本兼容策略

首版版本兼容规则固定如下：

- `console-web` 与 `platform-api` 必须保持同一主版本
- `session-gateway` 与 `platform-api` 必须保持同一主版本和兼容的事件协议版本
- `job-runner` 与 `platform-api` 必须保持同一主版本和兼容的任务模型版本
- `agent-desktop` 与 `agent-core` 必须保持同一主版本
- `agent-core` 最多允许落后平台一个次版本，但不得落后一个主版本

升级原则：

- 平台升级优先保证旧版 Agent 的只读诊断能力不立即失效
- 如果平台变更了注册、配置、事件或审批协议，必须提升协议版本并保留兼容窗口
- 首版不支持跨主版本长期兼容
- 租户专属分发包必须记录构建时所对应的平台协议版本

兼容失败处理：

- Agent 发现协议不兼容时，必须返回结构化错误
- 平台必须提示升级 Agent 或切换兼容分发包

### 12.3 从方案到开工的下一步设计产物

为了从当前总方案进入研发阶段，必须补齐以下文档：

- 平台服务拆分图
- Agent 模块目录与包结构
- 核心数据库 ER 草图
- 注册、拉配、会话、审批四类 API 草案
- 设备、会话、审批三个状态机说明
- 公有云与私有化版本的能力边界说明
- 环境治理基线模型与漂移检测规则草案

### 12.4 代码仓库组织规范

为了后续可以按文档完整生成代码，仓库从第一天起就固定采用单仓多应用结构，并固定目录边界。

固定仓库结构：

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
    go/
      auth/
      config/
      db/
      events/
      logging/
      policy/
      sdk/
      types/
    ts/
      ui-kit/
      api-client/
  deploy/
    docker/
    k8s/
    private/
  docs/
    prd/
    architecture/
    api/
    database/
```

组织原则：

- `apps` 放可独立运行的终端产品
- `services` 放平台后端服务
- `libs/go` 放所有服务和 Agent Core 共享包
- `libs/ts` 放控制台和桌面 UI 共享逻辑
- `deploy` 放 SaaS、私有化和本地开发部署清单

### 12.5 Go 服务与 Agent 模块包结构

为了让代码生成具备稳定输出目标，Go 项目的包层次固定如下。

每个 Go 服务统一结构：

```text
service-name/
  cmd/
    service-name/
      main.go
  internal/
    app/
    handler/
    service/
    repository/
    domain/
    dto/
    middleware/
    transport/
    worker/
  migrations/
  config/
```

### 12.5.1 `platform-api` 服务规格

`platform-api` 是平台侧唯一的聚合业务服务，首版必须承载以下能力：

- 控制台登录与身份校验
- 租户、用户、角色管理
- `ModelProfile`、`PolicyProfile`、`AgentProfile` 管理
- 租户专属下载链接签发
- 设备激活、配置拉取、心跳接入
- 审计查询
- 包管理元数据
- `Webhook` 接入与签名校验

固定目录结构：

```text
services/platform-api/
  cmd/
    platform-api/
      main.go
  internal/
    app/
    handler/
      http/
      agent/
      webhook/
    service/
      auth/
      tenant/
      profile/
      package/
      enrollment/
      device/
      audit/
    repository/
    domain/
    dto/
    middleware/
    transport/
      http/
    worker/
  migrations/
  config/
    default.yaml
```

固定启动入口：

- 二进制入口：`services/platform-api/cmd/platform-api/main.go`
- 本地启动命令：`go run ./services/platform-api/cmd/platform-api`
- 构建产物名：`platform-api`
- 默认监听端口：`8080`

固定配置文件：

- `services/platform-api/config/default.yaml`
- `services/platform-api/.env.example`

固定环境变量前缀：

- `ENX_PLATFORM_API_*`

`default.yaml` 示例：

```yaml
app:
  name: platform-api
  env: local
  log_level: info

server:
  host: 0.0.0.0
  port: 8080
  public_base_url: http://localhost:8080
  read_timeout_seconds: 15
  write_timeout_seconds: 15

database:
  dsn: ${ENX_DATABASE_DSN}
  max_open_conns: 20
  max_idle_conns: 10

redis:
  addr: ${ENX_REDIS_ADDR}

object_storage:
  endpoint: ${ENX_OBJECT_STORAGE_ENDPOINT}
  bucket: ${ENX_OBJECT_STORAGE_BUCKET}

security:
  jwt_secret: ${ENX_JWT_SECRET}
  device_token_secret: ${ENX_DEVICE_TOKEN_SECRET}
  enroll_signing_secret: ${ENX_ENROLL_SIGNING_SECRET}
```

`main.go` 启动职责固定为：

1. 加载配置与环境变量
2. 初始化日志、数据库、`Redis`、对象存储客户端
3. 执行 migration 版本检查
4. 初始化 repository、service、handler、middleware
5. 注册 HTTP 路由
6. 启动 HTTP server
7. 注册优雅退出信号处理

首批 HTTP 接口职责固定为：

- `/api/v1/auth/*`
- `/api/v1/me`
- `/api/v1/tenants/*`
- `/api/v1/tenants/:tenantId/model-profiles`
- `/api/v1/tenants/:tenantId/policy-profiles`
- `/api/v1/tenants/:tenantId/agent-profiles`
- `/api/v1/tenants/:tenantId/download-links`
- `/api/v1/tenants/:tenantId/devices`
- `/api/v1/tenants/:tenantId/audit-events`
- `/agent/v1/enroll`
- `/agent/v1/heartbeat`
- `/agent/v1/config`
- `/agent/v1/device-events`
- `/agent/v1/audit-events`
- `/webhooks/v1/events`
- `/healthz`
- `/readyz`

`platform-api` 不负责：

- 长连接 `WebSocket` 会话保持
- 实时流式事件分发
- 后台异步构建和重试任务执行

### 12.5.2 `session-gateway` 服务规格

`session-gateway` 是平台侧唯一的实时会话网关，首版只负责实时链路，不承载配置和管理类 API。

固定目录结构：

```text
services/session-gateway/
  cmd/
    session-gateway/
      main.go
  internal/
    app/
    handler/
      ws/
    service/
      session/
      approval/
      relay/
    repository/
    domain/
    dto/
    middleware/
    transport/
      ws/
    worker/
  config/
    default.yaml
```

固定启动入口：

- 二进制入口：`services/session-gateway/cmd/session-gateway/main.go`
- 本地启动命令：`go run ./services/session-gateway/cmd/session-gateway`
- 构建产物名：`session-gateway`
- 默认监听端口：`8081`

固定配置文件：

- `services/session-gateway/config/default.yaml`
- `services/session-gateway/.env.example`

固定环境变量前缀：

- `ENX_SESSION_GATEWAY_*`

`default.yaml` 示例：

```yaml
app:
  name: session-gateway
  env: local
  log_level: info

server:
  host: 0.0.0.0
  port: 8081
  public_ws_url: ws://localhost:8081
  handshake_timeout_seconds: 10
  idle_timeout_seconds: 60

platform_api:
  base_url: http://platform-api:8080
  internal_token: ${ENX_SESSION_GATEWAY_INTERNAL_TOKEN}

redis:
  addr: ${ENX_REDIS_ADDR}
```

`main.go` 启动职责固定为：

1. 加载配置与环境变量
2. 初始化日志与 `Redis` 连接
3. 初始化平台内部调用客户端
4. 初始化连接管理器、事件路由器和会话服务
5. 注册 `WebSocket` 路由与健康检查路由
6. 启动网关服务
7. 注册优雅退出和连接清理逻辑

固定职责：

- 校验短期会话 token
- 建立和维持 `WebSocket` 连接
- 转发 `session.created`、`diagnosis.*`、`approval.*`、`tool.*`、`session.completed` 等事件
- 接收客户端上行事件
- 维护有限的连接状态和幂等事件处理

首批接口职责固定为：

- `GET /ws/v1/sessions/:deviceId`
- `GET /healthz`
- `GET /readyz`

`session-gateway` 不负责：

- 用户登录
- 租户配置写入
- 设备激活
- 数据库 migration
- 包构建

### 12.5.3 `job-runner` 服务规格

`job-runner` 是平台侧唯一的异步任务和定时任务执行器，所有后台任务都必须优先落在这里，不再新增第四个后端服务。

固定目录结构：

```text
services/job-runner/
  cmd/
    job-runner/
      main.go
  internal/
    app/
    handler/
    service/
      packagebuild/
      auditflush/
      webhookretry/
      governance/
      cleanup/
    repository/
    domain/
    dto/
    middleware/
    transport/
    worker/
      queue/
      cron/
  config/
    default.yaml
```

固定启动入口：

- 二进制入口：`services/job-runner/cmd/job-runner/main.go`
- 本地启动命令：`go run ./services/job-runner/cmd/job-runner`
- 构建产物名：`job-runner`
- 无公网端口

固定配置文件：

- `services/job-runner/config/default.yaml`
- `services/job-runner/.env.example`

固定环境变量前缀：

- `ENX_JOB_RUNNER_*`

`default.yaml` 示例：

```yaml
app:
  name: job-runner
  env: local
  log_level: info

database:
  dsn: ${ENX_DATABASE_DSN}

redis:
  addr: ${ENX_REDIS_ADDR}

object_storage:
  endpoint: ${ENX_OBJECT_STORAGE_ENDPOINT}
  bucket: ${ENX_OBJECT_STORAGE_BUCKET}

jobs:
  worker_concurrency: 10
  poll_interval_seconds: 5
  enable_package_build: true
  enable_webhook_retry: true
  enable_cleanup: true
  enable_governance_jobs: true
```

`main.go` 启动职责固定为：

1. 加载配置与环境变量
2. 初始化日志、数据库、`Redis`、对象存储客户端
3. 初始化任务仓储、任务分发器和各类 worker
4. 注册周期任务与清理任务
5. 启动任务消费循环
6. 处理重试、失败落盘和优雅退出

固定职责：

- 构建或装配租户专属分发包
- 审计批处理与归档
- `Webhook` 重试与失败补偿
- 清理过期下载链接、过期激活令牌和临时构建产物
- 执行治理类周期任务

队列与任务约束：

- 所有异步任务必须带 `tenant_id`
- 所有任务必须记录 `job_type`
- 所有任务必须可重试
- 所有任务必须记录失败原因和最后一次执行时间

`job-runner` 不负责：

- 对外管理 API
- 实时 `WebSocket` 连接
- 桌面端本地交互

#### 12.5.3.1 任务模型与状态机

首批任务类型固定为：

- `package_build`
- `audit_flush`
- `webhook_retry`
- `link_cleanup`
- `token_cleanup`
- `governance_scan`

任务状态固定为：

- `queued`
- `running`
- `succeeded`
- `failed`
- `retry_scheduled`
- `dead_lettered`
- `cancelled`

任务状态流转：

```text
queued
  |
  v
running
  +--> succeeded
  +--> failed ---------> dead_lettered
  |         |
  |         \---------> retry_scheduled --> queued
  \--> cancelled
```

任务约束：

- 所有任务必须带 `tenant_id`
- 所有任务必须带 `job_id`
- 所有任务必须带 `job_type`
- 所有任务必须带 `created_at` 和 `attempt_count`
- 所有任务必须定义幂等键
- 所有失败任务必须可追踪最后一次错误信息

### 12.5.4 三服务协作规则

三服务之间的调用关系固定如下：

```text
console-web
    |
    v
platform-api
    +--> session-gateway
    \--> job-runner

agent-core
    +--> platform-api
    \--> session-gateway
```

协作规则：

- `platform-api` 是唯一配置写入入口
- `session-gateway` 只消费配置结果和会话令牌，不写主业务配置
- `job-runner` 只执行后台任务，不对外暴露管理接口
- 任意新增平台能力优先归属到这三个服务之一，不得先新增第四个服务

服务依赖初始化顺序固定为：

1. `platform-api`
2. `session-gateway`
3. `job-runner`

代码生成约束：

- 每个服务都必须有独立 `main.go`
- 每个服务都必须有独立 `config/default.yaml`
- 每个服务都必须有独立 `.env.example`
- 每个服务都必须实现 `healthz`
- 对外暴露端口的服务必须实现 `readyz`

`agent-core` 统一采用以下运行时模块边界：

```text
agent-core/
  cmd/
    enx-agent/
  internal/
    bootstrap/
    runtime/
    device/
    enrollment/
    session/
    diagnosis/
    governance/
    policy/
    approval/
    audit/
    tools/
      network/
      system/
      runtime/
      container/
      env/
    llm/
      router/
      providers/
    adapters/
      os/
      vm/
      container/
    api/
    store/
```

模块职责：

- `bootstrap`：首次启动、激活、配置拉取
- `runtime`：主事件循环、任务调度、生命周期管理
- `device`：设备身份、设备状态、心跳与配置版本
- `session`：本地会话、远程会话、状态机
- `diagnosis`：问题分解、证据收集、诊断计划
- `governance`：基线、漂移检测、治理规则
- `policy`：策略求值、权限裁决、动作过滤
- `approval`：审批单生成、等待、超时、执行前确认
- `audit`：本地审计和上报
- `tools/*`：各类结构化工具实现
- `llm/router`：多模型路由与降级

### 12.5.5 `console-web` 模块规格

为了避免后续代码生成时把控制台前端写成“页面堆”，`console-web` 的模块边界固定如下：

固定目录结构：

```text
apps/console-web/
  src/
    app/
    modules/
      auth/
      overview/
      tenants/
      devices/
      sessions/
      audits/
      packages/
      profiles/
      governance/
      settings/
    components/
    lib/
      api/
      auth/
      tenant/
      observability/
      i18n/
    hooks/
    stores/
    types/
  public/
  middleware.ts
  next.config.js
```

页面与路由域固定如下：

- `/login`
- `/overview`
- `/tenants/[tenantId]/devices`
- `/tenants/[tenantId]/sessions`
- `/tenants/[tenantId]/audit-events`
- `/tenants/[tenantId]/download-packages`
- `/tenants/[tenantId]/model-profiles`
- `/tenants/[tenantId]/policy-profiles`
- `/tenants/[tenantId]/agent-profiles`
- `/tenants/[tenantId]/governance`
- `/settings`

前端约束固定如下：

- **国际化 (i18n)**：控制台前端必须全面支持中英双语（`zh` / `en`），所有硬编码文案必须通过 `i18n` 上下文进行管理，支持用户在界面上动态切换语言。
- 控制台前端只通过 `platform-api` 暴露的 HTTP API 读写配置，不直连数据库、`Redis`、对象存储
- 认证、租户上下文、权限快照必须在统一会话层处理，不允许散落在页面组件里重复判断
- 业务模块只能通过统一 API SDK 访问后端，不允许页面内直接拼接请求协议
- 所有列表页必须支持按 `tenant_id`、时间范围、关键状态过滤
- 审计、审批、发布、回滚页面必须展示关联 ID，并作为后端日志与 Trace 的统一对齐入口

固定环境变量前缀：

- `ENX_CONSOLE_WEB_*`

首版必须具备的前端可观测入口：

- 全局错误页
- 请求失败统一提示
- 用户操作事件埋点入口
- 租户级筛选上下文展示
- 诊断链路、审批链路、发布链路的状态页

### 12.5.6 `agent-desktop` 模块规格

桌面端必须明确区分主进程、渲染进程和 `preload`，避免首版就形成不可维护的进程耦合。

固定目录结构：

```text
apps/agent-desktop/
  src/
    main/
      bootstrap/
      windows/
      tray/
      updater/
      local-api/
      observability/
    preload/
    renderer/
      app/
      modules/
        chat/
        diagnosis/
        approvals/
        history/
        settings/
        diagnostics/
      components/
      hooks/
      stores/
      lib/
    shared/
      contracts/
      errors/
      types/
  assets/
```

进程职责固定如下：

- 主进程
  - 负责窗口生命周期、系统托盘、应用菜单、升级入口、日志目录、诊断包导出
- `preload`
  - 负责暴露受限的 IPC 接口
  - 不直接持有业务状态
- 渲染进程
  - 负责聊天 UI、审批弹窗、诊断结果展示、设置页、故障说明页
  - 不得直接访问本地文件系统和系统命令

桌面端约束固定如下：

- `agent-desktop` 不实现诊断与修复核心逻辑，实际执行统一由 `agent-core` 完成
- 主进程不得绕过 `agent-core` 直接执行系统变更
- 渲染进程不得直接调用任意 IPC 通道，必须经 `preload` 白名单暴露
- 品牌资源加载必须按租户分发包元数据生效，不允许运行中无审计地热替换品牌资源
- 升级入口必须先校验当前 `agent-core` 兼容性，再展示升级动作

固定环境变量前缀：

- `ENX_AGENT_DESKTOP_*`

### 12.5.7 `agent-desktop` 与 `agent-core` 本地协议边界

本地桌面 UI 与执行内核必须按“受控本地协议”协作，禁止通过共享内存或任意命令行参数直接耦合。

本地依赖关系固定如下：

1. `agent-desktop` 启动后优先探测本地 `agent-core`
2. 如 `agent-core` 未就绪，则展示启动中或降级说明页
3. 如 `agent-core` 不可用但设备身份存在，则仅开放只读错误说明和诊断包导出
4. 如设备未激活，则进入激活引导页

本地 API 边界固定如下：

- `GET /local/v1/runtime/status`
- `GET /local/v1/device`
- `GET /local/v1/sessions/current`
- `POST /local/v1/chat/messages`
- `GET /local/v1/approvals/:approvalId`
- `POST /local/v1/approvals/:approvalId/confirm`
- `POST /local/v1/approvals/:approvalId/deny`
- `GET /local/v1/audits`
- `POST /local/v1/diagnostics/export`

协议规则固定如下：

- 本地 API 只监听 `localhost`
- 所有响应必须带结构化错误码，不返回裸字符串错误
- `agent-desktop` 与 `agent-core` 必须共享一套 `contracts` 与错误码定义
- 审批、会话、诊断、审计事件必须带 `session_id` 或 `approval_id`
- 本地 UI 的所有状态展示都必须来源于 `agent-core` 的状态机，而不是 UI 自行推断

本地降级规则固定如下：

- `agent-core` 离线时，桌面端必须明确展示“当前不可执行修复”
- 本地策略禁止时，桌面端只能展示不可执行原因，不得隐藏策略拦截事实
- 本地错误快照导出时必须先脱敏，再打包为诊断包
- 诊断包至少包含版本信息、错误码、最近状态流和脱敏后的日志摘要

### 12.5.8 后端开发架构与框架规范

为了确保后续每次后端开发都能保持一致的代码结构和质量，必须严格遵守以下规范：

- **Web 框架**：所有 HTTP 接口必须使用 `github.com/gin-gonic/gin` 框架暴露。
- **架构模式**：必须采用 DDD（领域驱动设计）分层架构。
  - `domain`：定义核心业务实体（Entity）、值对象（Value Object）和仓储接口（Repository Interface）。不允许依赖其他层。
  - `repository`：实现 `domain` 中定义的仓储接口，负责与数据库（如 MySQL/GORM）、缓存（Redis）或第三方服务交互。
  - `service`：应用服务层，负责编排领域对象和仓储，实现具体业务用例。
  - `handler`：表现层，负责解析 HTTP 请求（Gin Context）、参数校验、调用 `service` 并返回标准化 HTTP 响应。
  - `dto`：数据传输对象，用于 `handler` 与 `service` 之间，以及对外 API 的请求/响应载荷定义。
- **依赖注入**：在 `cmd/xxx/main.go` 中手动进行依赖注入（如 `repo -> service -> handler`），不引入复杂的 DI 框架。
- **数据库操作**：默认使用 `gorm.io/gorm` 进行关系型数据库操作。

### 12.6 数据库设计粒度

为了支持后续直接生成数据库模型代码，核心表结构必须细化到“首批字段级别”。

首批必须落地的表：

- `tenants`
  - `id`
  - `name`
  - `slug`
  - `status`
  - `created_at`
- `users`
  - `id`
  - `tenant_id`
  - `email`
  - `display_name`
  - `status`
  - `last_login_at`
- `roles`
  - `id`
  - `tenant_id`
  - `name`
  - `permissions_json`
- `model_profiles`
  - `id`
  - `tenant_id`
  - `name`
  - `provider`
  - `base_url`
  - `model_name`
  - `params_json`
  - `secret_mode`
  - `status`
- `policy_profiles`
  - `id`
  - `tenant_id`
  - `name`
  - `policy_json`
  - `version`
  - `status`
- `agent_profiles`
  - `id`
  - `tenant_id`
  - `name`
  - `model_profile_id`
  - `policy_profile_id`
  - `capabilities_json`
  - `update_channel`
- `download_packages`
  - `id`
  - `tenant_id`
  - `agent_profile_id`
  - `distribution_mode`
  - `platform`
  - `arch`
  - `version`
  - `package_name`
  - `download_url`
  - `artifact_path`
  - `artifact_size`
  - `checksum`
  - `bootstrap_manifest_json`
  - `branding_version`
  - `build_version`
  - `sign_status`
  - `sign_metadata_json`
  - `published_at`
- `enrollment_tokens`
  - `id`
  - `tenant_id`
  - `agent_profile_id`
  - `download_package_id`
  - `token_hash`
  - `channel`
  - `expires_at`
  - `max_uses`
  - `used_count`
  - `issued_by_user_id`
  - `status`
- `devices`
  - `id`
  - `tenant_id`
  - `agent_profile_id`
  - `device_name`
  - `platform`
  - `arch`
  - `environment_type`
  - `agent_version`
  - `status`
  - `last_seen_at`
  - `policy_version`
- `sessions`
  - `id`
  - `tenant_id`
  - `device_id`
  - `transport`
  - `status`
  - `started_at`
  - `ended_at`
  - `initiator_type`
- `tool_invocations`
  - `id`
  - `session_id`
  - `device_id`
  - `tool_name`
  - `risk_level`
  - `input_json`
  - `output_json`
  - `status`
  - `duration_ms`
- `approval_requests`
  - `id`
  - `session_id`
  - `device_id`
  - `requested_action_json`
  - `risk_level`
  - `status`
  - `approver_user_id`
  - `approved_at`
  - `expires_at`
- `audit_events`
  - `id`
  - `tenant_id`
  - `device_id`
  - `session_id`
  - `event_type`
  - `event_payload_json`
  - `created_at`

数据库约束：

- 所有主表使用 `ULID` 或有序 UUID
- 所有配置类表带 `version`
- 所有高价值事件表带 `tenant_id`、`device_id`、`created_at`
- 审计与审批相关记录默认不可硬删除

### 12.6.1 MySQL 8 建模规范

为了让后续代码生成、迁移和索引设计保持统一，MySQL 规范固定如下：

- 字符集：`utf8mb4`
- 排序规则：`utf8mb4_0900_ai_ci`
- 存储引擎：`InnoDB`
- 主键类型：`CHAR(26)`，存储 `ULID`
- 时间字段：统一使用 `DATETIME(3)`，默认 UTC
- JSON 字段：统一使用 `JSON`
- 布尔值：统一使用 `TINYINT(1)`
- 状态字段：统一使用 `VARCHAR(32)` 或 `VARCHAR(48)`，不直接用数据库枚举
- 版本字段：统一使用 `INT UNSIGNED`
- 软删除字段：统一使用 `deleted_at DATETIME(3) NULL`

建模原则：

- 关键关系对象显式结构化，不用 JSON 替代关系建模
- 变化频繁、结构不稳定的数据才进入 JSON 字段
- 高频筛选字段必须有独立列，不能只存在 JSON 中
- 所有列表页默认基于 `(tenant_id, created_at)` 或 `(tenant_id, updated_at)` 检索

### 12.6.2 外键与删除策略

外键策略采用“关键关系保留外键，审计事件减少强外键”的混合模式。

保留外键的表：

- `users.tenant_id -> tenants.id`
- `roles.tenant_id -> tenants.id`
- `model_profiles.tenant_id -> tenants.id`
- `policy_profiles.tenant_id -> tenants.id`
- `agent_profiles.tenant_id -> tenants.id`
- `devices.tenant_id -> tenants.id`
- `sessions.device_id -> devices.id`
- `approval_requests.session_id -> sessions.id`
- `tool_invocations.session_id -> sessions.id`

删除策略：

- 租户、设备、会话、审批、审计默认不物理删除
- 配置对象优先使用 `status=archived` 或 `deleted_at`
- 审计类表避免深层级级联删除，保留历史可追溯性

### 12.6.3 核心表关系草图

核心关系可按下面理解：

```text
Tenant
  +--> User
  +--> Role
  +--> ModelProfile
  +--> PolicyProfile
  +--> AgentProfile --> Device --> Session
  |                        |          +--> ToolInvocation
  |                        |          +--> ApprovalRequest
  |                        |          \--> AuditEvent
  |                        |
  |                        \--> Device --> AuditEvent
  |
  \--> AuditEvent
```

### 12.6.4 表级详细设计规范

#### `tenants`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `name VARCHAR(128) NOT NULL`
- `slug VARCHAR(64) NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `plan_code VARCHAR(32) NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`

固定索引：

- `UNIQUE KEY uk_tenants_slug (slug)`
- `KEY idx_tenants_status_created (status, created_at)`

#### `users`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `email VARCHAR(191) NOT NULL`
- `display_name VARCHAR(128) NOT NULL`
- `password_hash VARCHAR(255) NULL`
- `status VARCHAR(32) NOT NULL`
- `last_login_at DATETIME(3) NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`
- `deleted_at DATETIME(3) NULL`

固定索引：

- `UNIQUE KEY uk_users_tenant_email (tenant_id, email)`
- `KEY idx_users_tenant_status (tenant_id, status)`

#### `roles`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `name VARCHAR(64) NOT NULL`
- `permissions_json JSON NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`

固定索引：

- `UNIQUE KEY uk_roles_tenant_name (tenant_id, name)`

#### `model_profiles`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `name VARCHAR(128) NOT NULL`
- `provider VARCHAR(64) NOT NULL`
- `base_url VARCHAR(255) NOT NULL`
- `model_name VARCHAR(128) NOT NULL`
- `params_json JSON NOT NULL`
- `secret_mode VARCHAR(32) NOT NULL`
- `fallback_model_profile_id CHAR(26) NULL`
- `status VARCHAR(32) NOT NULL`
- `version INT UNSIGNED NOT NULL DEFAULT 1`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`
- `deleted_at DATETIME(3) NULL`

固定索引：

- `UNIQUE KEY uk_model_profiles_tenant_name (tenant_id, name)`
- `KEY idx_model_profiles_tenant_status (tenant_id, status)`

#### `policy_profiles`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `name VARCHAR(128) NOT NULL`
- `policy_json JSON NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `version INT UNSIGNED NOT NULL DEFAULT 1`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`
- `deleted_at DATETIME(3) NULL`

固定索引：

- `UNIQUE KEY uk_policy_profiles_tenant_name (tenant_id, name)`
- `KEY idx_policy_profiles_tenant_status (tenant_id, status)`

#### `agent_profiles`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `name VARCHAR(128) NOT NULL`
- `model_profile_id CHAR(26) NOT NULL`
- `policy_profile_id CHAR(26) NOT NULL`
- `capabilities_json JSON NOT NULL`
- `update_channel VARCHAR(32) NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `version INT UNSIGNED NOT NULL DEFAULT 1`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`
- `deleted_at DATETIME(3) NULL`

固定索引：

- `UNIQUE KEY uk_agent_profiles_tenant_name (tenant_id, name)`
- `KEY idx_agent_profiles_tenant_status (tenant_id, status)`
- `KEY idx_agent_profiles_model_policy (model_profile_id, policy_profile_id)`

#### `download_packages`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `agent_profile_id CHAR(26) NOT NULL`
- `distribution_mode VARCHAR(32) NOT NULL`
- `platform VARCHAR(32) NOT NULL`
- `arch VARCHAR(32) NOT NULL`
- `version VARCHAR(32) NOT NULL`
- `package_name VARCHAR(255) NOT NULL`
- `download_url VARCHAR(1024) NOT NULL`
- `artifact_path VARCHAR(1024) NOT NULL`
- `artifact_size BIGINT UNSIGNED NOT NULL DEFAULT 0`
- `checksum VARCHAR(128) NOT NULL`
- `bootstrap_manifest_json JSON NULL`
- `branding_version INT UNSIGNED NOT NULL DEFAULT 1`
- `build_version VARCHAR(64) NOT NULL`
- `sign_status VARCHAR(32) NOT NULL`
- `sign_metadata_json JSON NULL`
- `status VARCHAR(32) NOT NULL`
- `published_at DATETIME(3) NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`

固定索引：

- `UNIQUE KEY uk_download_packages_profile_platform_arch_version (agent_profile_id, distribution_mode, platform, arch, version)`
- `KEY idx_download_packages_tenant_created (tenant_id, created_at)`
- `KEY idx_download_packages_tenant_status_published (tenant_id, status, published_at)`

#### `enrollment_tokens`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `agent_profile_id CHAR(26) NOT NULL`
- `download_package_id CHAR(26) NOT NULL`
- `token_hash CHAR(64) NOT NULL`
- `channel VARCHAR(32) NOT NULL`
- `expires_at DATETIME(3) NOT NULL`
- `max_uses INT UNSIGNED NOT NULL DEFAULT 1`
- `used_count INT UNSIGNED NOT NULL DEFAULT 0`
- `issued_by_user_id CHAR(26) NULL`
- `status VARCHAR(32) NOT NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`

固定索引：

- `UNIQUE KEY uk_enrollment_tokens_hash (token_hash)`
- `KEY idx_enrollment_tokens_tenant_status_expiry (tenant_id, status, expires_at)`
- `KEY idx_enrollment_tokens_package_status (download_package_id, status)`

#### `devices`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `agent_profile_id CHAR(26) NOT NULL`
- `device_name VARCHAR(128) NOT NULL`
- `hostname VARCHAR(191) NULL`
- `platform VARCHAR(32) NOT NULL`
- `arch VARCHAR(32) NOT NULL`
- `environment_type VARCHAR(32) NOT NULL`
- `agent_version VARCHAR(32) NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `policy_version INT UNSIGNED NOT NULL DEFAULT 1`
- `last_seen_at DATETIME(3) NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`
- `deleted_at DATETIME(3) NULL`

固定索引：

- `KEY idx_devices_tenant_status_last_seen (tenant_id, status, last_seen_at)`
- `KEY idx_devices_tenant_profile (tenant_id, agent_profile_id)`
- `KEY idx_devices_hostname (hostname)`

#### `sessions`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `device_id CHAR(26) NOT NULL`
- `transport VARCHAR(32) NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `initiator_type VARCHAR(32) NOT NULL`
- `started_at DATETIME(3) NOT NULL`
- `ended_at DATETIME(3) NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`

固定索引：

- `KEY idx_sessions_device_started (device_id, started_at)`
- `KEY idx_sessions_tenant_status_started (tenant_id, status, started_at)`

#### `tool_invocations`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `session_id CHAR(26) NOT NULL`
- `device_id CHAR(26) NOT NULL`
- `tool_name VARCHAR(128) NOT NULL`
- `risk_level VARCHAR(16) NOT NULL`
- `input_json JSON NOT NULL`
- `output_json JSON NULL`
- `status VARCHAR(32) NOT NULL`
- `duration_ms INT UNSIGNED NULL`
- `started_at DATETIME(3) NOT NULL`
- `finished_at DATETIME(3) NULL`
- `created_at DATETIME(3) NOT NULL`

固定索引：

- `KEY idx_tool_invocations_session_started (session_id, started_at)`
- `KEY idx_tool_invocations_device_tool (device_id, tool_name)`
- `KEY idx_tool_invocations_status_created (status, created_at)`

#### `approval_requests`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `session_id CHAR(26) NOT NULL`
- `device_id CHAR(26) NOT NULL`
- `requested_action_json JSON NOT NULL`
- `risk_level VARCHAR(16) NOT NULL`
- `status VARCHAR(32) NOT NULL`
- `approver_user_id CHAR(26) NULL`
- `approved_at DATETIME(3) NULL`
- `expires_at DATETIME(3) NULL`
- `executed_at DATETIME(3) NULL`
- `created_at DATETIME(3) NOT NULL`
- `updated_at DATETIME(3) NOT NULL`

固定索引：

- `KEY idx_approval_requests_session_status (session_id, status)`
- `KEY idx_approval_requests_device_created (device_id, created_at)`
- `KEY idx_approval_requests_approver_status (approver_user_id, status)`

#### `audit_events`

固定字段：

- `id CHAR(26) PRIMARY KEY`
- `tenant_id CHAR(26) NOT NULL`
- `device_id CHAR(26) NULL`
- `session_id CHAR(26) NULL`
- `event_type VARCHAR(64) NOT NULL`
- `event_payload_json JSON NOT NULL`
- `created_at DATETIME(3) NOT NULL`

固定索引：

- `KEY idx_audit_events_tenant_created (tenant_id, created_at)`
- `KEY idx_audit_events_device_created (device_id, created_at)`
- `KEY idx_audit_events_session_created (session_id, created_at)`
- `KEY idx_audit_events_type_created (event_type, created_at)`

### 12.6.5 JSON 字段使用边界

为了避免 `MySQL` 下 JSON 字段被滥用，JSON 使用边界固定如下。

可以使用 JSON 的场景：

- 模型参数
- 策略原始表达式
- 工具输入与输出
- 审批动作摘要
- 审计事件 payload
- 设备环境快照

不建议只放 JSON 的场景：

- 需要唯一约束的字段
- 高频过滤条件
- 租户归属、状态、版本、时间戳
- 关键外键关系

如果某个 JSON 内字段后续需要频繁查询，应升级为普通列，或增加生成列加索引。

`Device` JSON 示例：

```json
{
  "id": "01JDEVICE001ABCDEFGHJKMNPQ",
  "tenant_id": "01JTENANT0001ABCDEFGHJKMNPQ",
  "agent_profile_id": "01JAGENT0001ABCDEFGHJKMNPQ",
  "device_name": "DEV-001",
  "hostname": "dev-host",
  "platform": "windows",
  "arch": "amd64",
  "environment_type": "physical",
  "agent_version": "0.1.0",
  "status": "active",
  "policy_version": 1,
  "last_seen_at": "2026-03-21T12:00:00Z"
}
```

`ApprovalRequest` JSON 示例：

```json
{
  "id": "01JAPPR0001ABCDEFGHJKMNPQ",
  "session_id": "01JSESSION01ABCDEFGHJKMNPQ",
  "device_id": "01JDEVICE001ABCDEFGHJKMNPQ",
  "requested_action_json": {
    "tool_name": "service.restart",
    "target": "Dnscache",
    "reason": "dns resolver unhealthy"
  },
  "risk_level": "L1",
  "status": "pending_user",
  "approver_user_id": null,
  "approved_at": null,
  "expires_at": "2026-03-21T12:10:00Z"
}
```

### 12.6.6 首批迁移顺序

首批 migration 固定按依赖顺序生成：

1. `tenants`
2. `users`
3. `roles`
4. `model_profiles`
5. `policy_profiles`
6. `agent_profiles`
7. `download_packages`
8. `enrollment_tokens`
9. `devices`
10. `sessions`
11. `tool_invocations`
12. `approval_requests`
13. `audit_events`

理由：

- 先建租户和配置类基础表
- 再建注册和设备生命周期表
- 最后建会话、工具、审批和审计执行表

### 12.6.7 第二阶段扩展表

为了避免第一版一次性建太多表，以下表固定为第二阶段扩展：

- `role_bindings`
- `device_heartbeats`
- `policy_snapshots`
- `governance_baselines`
- `governance_drifts`
- `session_messages`
- `webhook_subscriptions`
- `webhook_deliveries`
- `package_channels`
- `device_labels`

这些表等核心主流程稳定后再引入，能显著降低第一版 schema 复杂度。

### 12.7 首批 API 设计边界

为了方便后续自动生成后端路由、DTO 和 SDK，首批 API 分组固定如下。

控制台 API：

- `POST /api/v1/auth/login`
- `GET /api/v1/me`
- `GET /api/v1/tenants/:tenantId`
- `GET /api/v1/tenants/:tenantId/model-profiles`
- `POST /api/v1/tenants/:tenantId/model-profiles`
- `GET /api/v1/tenants/:tenantId/policy-profiles`
- `POST /api/v1/tenants/:tenantId/agent-profiles`
- `POST /api/v1/tenants/:tenantId/download-links`
- `GET /api/v1/tenants/:tenantId/devices`
- `GET /api/v1/tenants/:tenantId/audit-events`

注册与配置 API：

- `POST /agent/v1/enroll`
- `POST /agent/v1/heartbeat`
- `GET /agent/v1/config`
- `POST /agent/v1/device-events`
- `POST /agent/v1/audit-events`

会话与审批 API：

- `GET /ws/v1/sessions/:deviceId`
- `POST /api/v1/sessions`
- `POST /api/v1/sessions/:sessionId/approve`
- `POST /api/v1/sessions/:sessionId/deny`
- `POST /api/v1/sessions/:sessionId/abort`

Webhook API：

- `POST /webhooks/v1/events`
- `GET /api/v1/webhooks`
- `POST /api/v1/webhooks/test`

API 设计原则：

- 控制台 API 面向用户和管理端
- `/agent/v1/*` 只面向设备端
- `ws` 通道只传会话事件，不承载复杂管理查询
- DTO 与内部 domain model 分离，便于后续演进

### 12.7.1 API 统一规范

为了后续可以稳定生成后端 handler、前端 SDK 和 Agent Client，API 统一风格固定如下。

统一规则：

- 所有 HTTP API 前缀统一为 `/api/v1`、`/agent/v1`、`/webhooks/v1`
- 内容类型统一为 `application/json`
- 时间统一使用 RFC3339 UTC 字符串
- ID 统一使用 `ULID`
- 分页统一使用游标或 `page_size + next_cursor`
- 列表接口默认返回 `items` 与 `page` 对象

响应包络：

```json
{
  "request_id": "01HXXXX",
  "data": {},
  "error": null
}
```

错误响应：

```json
{
  "request_id": "01HXXXX",
  "data": null,
  "error": {
    "code": "approval_expired",
    "message": "approval request has expired",
    "details": {}
  }
}
```

### 12.7.2 认证与鉴权模型

认证凭证固定为三类：

- 控制台用户凭证
  - 用户登录获取 access token
  - 用于控制台和管理 API
- 设备凭证
  - 设备激活后获得 device token 或 mTLS 证书
  - 用于 `/agent/v1/*`
- Webhook 凭证
  - 每个 webhook endpoint 独立签名密钥
  - 用于来源校验和幂等

鉴权策略：

- `/api/v1/*`：JWT 或 session token
- `/agent/v1/*`：设备 token 或 mTLS
- `/webhooks/v1/*`：HMAC 签名
- `ws`：通过短期会话 token 建连

### 12.7.3 控制台 API 详细设计

#### `POST /api/v1/auth/login`

用途：

- 用户登录控制台

请求体：

```json
{
  "email": "admin@example.com",
  "password": "******"
}
```

响应体：

```json
{
  "request_id": "01HXXXX",
  "data": {
    "access_token": "jwt-or-session-token",
    "expires_in": 3600,
    "user": {
      "id": "01HUSER",
      "tenant_id": "01HTENANT",
      "email": "admin@example.com",
      "display_name": "Admin"
    }
  },
  "error": null
}
```

#### `GET /api/v1/me`

用途：

- 获取当前登录用户、租户和角色摘要

返回字段：

- `user`
- `tenant`
- `roles`
- `permissions`

#### `GET /api/v1/tenants/:tenantId`

用途：

- 获取租户详情

返回字段：

- 基础资料
- 套餐与状态
- 设备统计
- 配置统计

#### `GET /api/v1/tenants/:tenantId/model-profiles`

用途：

- 分页查询模型配置

筛选参数：

- `status`
- `provider`
- `page_size`
- `next_cursor`

#### `POST /api/v1/tenants/:tenantId/model-profiles`

用途：

- 新建模型配置

请求体：

```json
{
  "name": "default-openai",
  "provider": "openai_compatible",
  "base_url": "https://api.example.com/v1",
  "model_name": "gpt-4.1",
  "params": {
    "temperature": 0.2,
    "max_tokens": 2048
  },
  "secret_mode": "hosted"
}
```

#### `GET /api/v1/tenants/:tenantId/policy-profiles`

用途：

- 查询策略配置列表

返回字段：

- `policy_json`
- `version`
- `status`
- `updated_at`

#### `POST /api/v1/tenants/:tenantId/agent-profiles`

用途：

- 创建 AgentProfile

请求体：

```json
{
  "name": "default-agent",
  "model_profile_id": "01HMODEL",
  "policy_profile_id": "01HPOLICY",
  "capabilities": {
    "diagnose_network": true,
    "diagnose_runtime": true,
    "allow_repair": false
  },
  "update_channel": "stable"
}
```

#### `POST /api/v1/tenants/:tenantId/download-links`

用途：

- 生成租户专属下载链接或激活包

请求体：

```json
{
  "agent_profile_id": "01HAGENT",
  "platform": "windows",
  "arch": "amd64",
  "expires_in_minutes": 120
}
```

返回字段：

- `download_url`
- `package_id`
- `enrollment_token_preview`
- `expires_at`

#### `GET /api/v1/tenants/:tenantId/devices`

用途：

- 查询设备列表

筛选参数：

- `status`
- `platform`
- `environment_type`
- `agent_profile_id`

#### `GET /api/v1/tenants/:tenantId/audit-events`

用途：

- 查询审计事件

筛选参数：

- `device_id`
- `session_id`
- `event_type`
- `start_at`
- `end_at`

### 12.7.4 Agent API 详细设计

#### `POST /agent/v1/enroll`

用途：

- 设备首次激活

请求体：

```json
{
  "enrollment_token": "one-time-token",
  "device": {
    "device_name": "DEV-001",
    "hostname": "dev-host",
    "platform": "windows",
    "arch": "amd64",
    "environment_type": "physical"
  },
  "agent": {
    "version": "0.1.0"
  }
}
```

响应体：

```json
{
  "request_id": "01HXXXX",
  "data": {
    "device_id": "01HDEVICE",
    "device_token": "signed-device-token",
    "config_version": 1,
    "agent_profile": {},
    "model_profile": {},
    "policy_profile": {}
  },
  "error": null
}
```

#### `POST /agent/v1/heartbeat`

用途：

- 设备保活、状态汇报、版本上报

请求体：

```json
{
  "device_id": "01HDEVICE",
  "status": "active",
  "agent_version": "0.1.0",
  "policy_version": 1,
  "stats": {
    "cpu_percent": 10.2,
    "memory_mb": 128
  }
}
```

#### `GET /agent/v1/config`

用途：

- 拉取最新配置

查询参数：

- `device_id`
- `current_config_version`

返回字段：

- `has_update`
- `config_version`
- `agent_profile`
- `model_profile`
- `policy_profile`

#### `POST /agent/v1/device-events`

用途：

- 上报设备状态变化和治理事件

payload：

- `event_type`
- `event_payload`
- `occurred_at`

#### `POST /agent/v1/audit-events`

用途：

- 批量或单条上报审计事件

请求体：

```json
{
  "events": [
    {
      "event_type": "approval.requested",
      "session_id": "01HSESSION",
      "event_payload": {}
    }
  ]
}
```

### 12.7.5 会话与审批 API 详细设计

#### `POST /api/v1/sessions`

用途：

- 创建会话

请求体：

```json
{
  "device_id": "01HDEVICE",
  "transport": "web",
  "initiator_type": "user",
  "initial_message": "network is broken"
}
```

返回字段：

- `session_id`
- `status`
- `ws_token`

#### `POST /api/v1/sessions/:sessionId/approve`

用途：

- 批准审批单

请求体：

```json
{
  "approval_request_id": "01HAPPR",
  "comment": "approved by admin"
}
```

#### `POST /api/v1/sessions/:sessionId/deny`

用途：

- 拒绝审批单

请求体：

```json
{
  "approval_request_id": "01HAPPR",
  "reason": "high risk"
}
```

#### `POST /api/v1/sessions/:sessionId/abort`

用途：

- 中止会话

请求体：

```json
{
  "reason": "user aborted"
}
```

审批接口规则：

- 一个会话同一时间只允许一个 `pending_user` 的高风险审批
- 已过期审批不得批准
- 已执行审批不得重复提交批准或拒绝

### 12.7.6 WebSocket 协议草案

连接地址：

- `GET /ws/v1/sessions/:deviceId?token=...`

连接后事件统一采用事件信封：

```json
{
  "event_id": "01HEVENT",
  "event_type": "session.created",
  "tenant_id": "01HTENANT",
  "device_id": "01HDEVICE",
  "session_id": "01HSESSION",
  "timestamp": "2026-03-21T10:00:00Z",
  "payload": {}
}
```

首批双向事件：

- 服务端下发
  - `session.created`
  - `diagnosis.started`
  - `diagnosis.completed`
  - `approval.requested`
  - `approval.expired`
  - `tool.started`
  - `tool.completed`
  - `session.completed`
- 客户端上行
  - `session.input`
  - `approval.submit`
  - `session.abort`
  - `heartbeat.ping`

协议规则：

- 所有事件必须带 `event_id`
- 所有高价值事件需保证幂等处理
- 连接断开后可根据 `session_id` 做有限重连恢复

### 12.7.7 Webhook 协议草案

#### `POST /webhooks/v1/events`

用途：

- 接收外部系统推送的治理事件或诊断触发事件

请求头：

- `X-ENX-Signature`
- `X-ENX-Timestamp`
- `X-ENX-Event-Id`

请求体：

```json
{
  "source": "monitoring-system",
  "event_type": "runtime.alert",
  "tenant_id": "01HTENANT",
  "device_selector": {
    "device_id": "01HDEVICE"
  },
  "payload": {
    "message": "service unreachable"
  }
}
```

规则：

- Webhook 只负责触发会话或诊断任务，不直接绕过本地审批
- 必须支持重试和幂等
- 必须记录原始签名校验结果

### 12.7.8 错误码集合

稳定错误码集合定义如下，供前后端与 Agent 共用。

首批错误码：

- `unauthorized`
- `forbidden`
- `tenant_not_found`
- `device_not_found`
- `session_not_found`
- `approval_not_found`
- `approval_expired`
- `approval_invalid_state`
- `policy_violation`
- `device_revoked`
- `invalid_enrollment_token`
- `config_version_conflict`
- `rate_limited`
- `internal_error`

### 12.7.9 API 生成约束

为了后续可以直接按文档生成代码，以下约束固定生效：

- HTTP handler 只负责参数解析、鉴权、响应映射
- 业务规则全部下沉到 service 层
- repository 层不得感知 HTTP 对象
- WebSocket 事件结构必须复用统一事件信封
- Agent 与控制台禁止复用完全相同的 DTO，避免边界混乱
- 所有状态流转接口必须校验当前状态是否合法

### 12.7.10 首版健康检查与 smoke test API

为了确保生成后的项目可以直接部署运行，首版必须补齐以下接口：

- `GET /healthz`
  - 进程存活检查
- `GET /readyz`
  - 依赖项就绪检查
- `GET /api/v1/bootstrap`
  - 控制台初始化摘要，可用于前端启动

`readyz` 至少要检查：

- 数据库连接
- `Redis` 连接
- 对象存储连接
- migration 版本是否已执行

首版 smoke test 固定按以下顺序执行：

1. `docker compose up -d`
2. 健康检查全部通过
3. 执行数据库 migration
4. 创建默认租户与管理员
5. 控制台成功登录
6. 成功创建 `ModelProfile`、`PolicyProfile`、`AgentProfile`
7. 成功生成下载链接
8. Agent 成功激活
9. Agent 成功建立 `WebSocket` 会话
10. 完成一次只读诊断
11. 完成一次审批式低风险修复
12. 审计列表中可查到完整事件链

### 12.8 事件模型与协议草案

后续如果要完整生成项目代码，必须尽早固定事件类型，不然 WebSocket、Webhook、审计和 Agent Runtime 会反复返工。

统一事件信封：

- `event_id`
- `event_type`
- `tenant_id`
- `device_id`
- `session_id`
- `timestamp`
- `payload`

首批事件类型：

- `device.enrolled`
- `device.heartbeat`
- `session.created`
- `session.status_changed`
- `diagnosis.started`
- `diagnosis.completed`
- `approval.requested`
- `approval.approved`
- `approval.denied`
- `tool.invoked`
- `tool.completed`
- `tool.failed`
- `governance.drift_detected`
- `governance.baseline_updated`
- `audit.recorded`

### 12.9 代码生成顺序

为了降低一次性生成全项目代码的失败率，代码生成固定按依赖顺序推进，而不是一口气全生成。

固定顺序：

1. 生成共享类型与配置规范
2. 生成数据库 schema、迁移和 repository 接口
3. 生成身份、租户、配置、注册四类后端服务骨架
4. 生成 Agent Core 的启动、配置拉取、设备心跳和本地存储
5. 生成会话系统与 WebSocket 网关
6. 生成策略引擎与审批流
7. 生成首批只读工具
8. 生成控制台前端和桌面 UI
9. 生成审计和治理闭环相关任务
10. 最后再生成低风险修复工具和私有化支持

原因：

- 先把对象模型、协议和持久化固定下来
- 再生成运行时和交互逻辑
- 最后开放高风险执行模块，避免早期代码结构失控

### 12.10 生成代码时必须坚持的约束

后续如果按本方案直接生成项目代码，必须遵守以下约束：

- 不允许跳过数据库迁移设计直接硬编码表结构
- 不允许让模型层直接依赖 HTTP handler
- 不允许把审批逻辑写在前端或提示词中
- 不允许把任意 Shell 暴露成通用工具
- 不允许私有化版本重新定义协议
- 不允许控制台 API、Agent API、WebSocket 事件混用对象格式

只要这几条被坚持，后续就可以把方案稳定转化成完整项目代码。

### 12.11 Agent Core 详细设计

为了让后续能够按文档生成 `agent-core` 的完整代码，需要把本地端的运行时模型、模块边界、执行链路和本地存储进一步固定。

#### 12.11.1 Agent Core 目标

`agent-core` 是整个产品最关键的执行组件，负责：

- 首次激活与配置拉取
- 设备身份与心跳
- 会话管理
- 诊断计划执行
- 工具调用与结果归集
- 策略求值
- 审批等待与确认
- 审计记录与上报
- 治理基线与漂移检测

它不负责：

- 富交互页面渲染
- 后台管理逻辑
- 多租户控制台聚合查询
- 云侧复杂报表

#### 12.11.2 Agent Core 进程结构

`agent-core` 固定采用单进程、多组件运行时实现，避免第一版引入本地多进程复杂度。

推荐运行时结构：

```text
Bootstrap
    |
    v
ConfigManager
    +--> DeviceManager --> HeartbeatWorker
    +--> SessionManager --> DiagnosisEngine
    |                         +--> ToolRuntime   --> AuditPipeline
    |                         \--> PolicyEngine --> ApprovalEngine --> LocalAPI
    |                                                   |
    |                                                   \--> AuditPipeline
    |
    \--> LocalStore
            +--> ConfigManager
            +--> SessionManager
            \--> AuditPipeline

LlmRouter ---------> DiagnosisEngine
GovernanceEngine --> AuditPipeline
DeviceManager -----> AuditPipeline
```

#### 12.11.3 启动阶段状态流

启动过程固定拆成明确状态，避免后续初始化逻辑散落在各个模块中。

启动状态：

- `boot.init`
- `boot.load_local_config`
- `boot.check_device_identity`
- `boot.enroll_if_needed`
- `boot.pull_remote_config`
- `boot.start_local_api`
- `boot.start_workers`
- `boot.ready`

启动规则：

- 如果不存在本地设备身份，则进入激活流程
- 如果存在设备身份但配置版本落后，则优先拉取远程配置
- 如果平台不可达，则在本地降级模式启动，只开放安全的只读能力
- 启动失败必须带可恢复错误码，便于桌面端展示

#### 12.11.4 模块边界与接口规范

##### `bootstrap`

职责：

- 加载本地配置文件
- 校验运行目录
- 初始化日志、数据库、缓存
- 协调首次激活与配置同步

固定接口：

- `Run(ctx context.Context) error`
- `LoadLocalConfig() (*LocalConfig, error)`
- `EnsureDeviceIdentity() (*DeviceIdentity, error)`

##### `config`

职责：

- 管理本地配置快照
- 拉取和缓存远程配置
- 比较配置版本

固定接口：

- `GetCurrentConfig() *RuntimeConfig`
- `PullRemoteConfig(ctx context.Context) (*RuntimeConfig, error)`
- `ApplyConfig(ctx context.Context, cfg *RuntimeConfig) error`

##### `device`

职责：

- 管理设备身份
- 维护设备状态
- 周期性发送心跳

固定接口：

- `Enroll(ctx context.Context, token string, req EnrollRequest) (*EnrollResult, error)`
- `Heartbeat(ctx context.Context) error`
- `CurrentDevice() *DeviceState`

##### `session`

职责：

- 创建和管理会话
- 驱动会话状态机
- 跟踪会话消息、审批、执行结果

固定接口：

- `CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error)`
- `AttachTransport(ctx context.Context, sessionID string, transport Transport) error`
- `AbortSession(ctx context.Context, sessionID string, reason string) error`

##### `diagnosis`

职责：

- 解析用户问题
- 构建诊断计划
- 分阶段调度工具
- 汇总证据并生成诊断结论

固定接口：

- `Plan(ctx context.Context, s *Session, input string) (*DiagnosisPlan, error)`
- `Execute(ctx context.Context, plan *DiagnosisPlan) (*DiagnosisResult, error)`

##### `policy`

职责：

- 判断某个工具是否可用
- 判断动作是否需要审批
- 评估当前会话风险等级

固定接口：

- `EvaluateTool(ctx context.Context, req ToolEvalRequest) (*ToolDecision, error)`
- `EvaluateAction(ctx context.Context, req ActionEvalRequest) (*ActionDecision, error)`

##### `approval`

职责：

- 生成审批请求
- 等待用户确认
- 驱动审批状态机

固定接口：

- `RequestApproval(ctx context.Context, req ApprovalDraft) (*ApprovalRequest, error)`
- `WaitDecision(ctx context.Context, approvalID string) (*ApprovalDecision, error)`
- `ApplyDecision(ctx context.Context, decision ApprovalDecision) error`

##### `tools`

职责：

- 统一注册工具
- 执行工具
- 收集标准化结果

固定接口：

- `ListAvailable(ctx context.Context, scope ToolScope) ([]ToolSpec, error)`
- `Invoke(ctx context.Context, call ToolCall) (*ToolResult, error)`

##### `audit`

职责：

- 本地写入审计记录
- 批量上报平台
- 在离线模式下暂存事件

固定接口：

- `Record(ctx context.Context, event AuditEvent) error`
- `Flush(ctx context.Context) error`

##### `governance`

职责：

- 采集环境基线
- 比较漂移
- 输出治理结果

固定接口：

- `CaptureBaseline(ctx context.Context) (*BaselineSnapshot, error)`
- `DetectDrift(ctx context.Context, baseline *BaselineSnapshot) ([]DriftFinding, error)`

#### 12.11.5 本地存储设计

Agent Core 必须使用本地持久化层，避免所有运行状态都依赖云端。

本地存储内容固定为：

- 本地配置快照
- 设备身份
- 会话摘要
- 审批单缓存
- 未上报审计事件
- 治理基线快照

本地存储方式固定为：

- 本地轻量数据库：`SQLite`
- 文件型配置：`YAML` 或 `JSON`
- 敏感信息：系统密钥链或本地加密存储

本地目录结构固定为：

```text
envnexus-agent/
  config/
    runtime.yaml
    device.json
  data/
    agent.db
  logs/
    agent.log
  cache/
    audits/
    snapshots/
```

本地存储原则：

- 配置和身份不能只存在内存中
- 审计上报失败时必须可恢复
- 会话恢复只保留必要摘要，不缓存无限量聊天历史

#### 12.11.6 工具运行时规范

工具系统是 Agent Core 的高风险区域，必须形成统一约束。

每个工具必须定义：

- `name`
- `display_name`
- `category`
- `platform_support`
- `environment_support`
- `read_only`
- `risk_level`
- `required_privileges`
- `timeout_seconds`
- `input_schema`
- `output_schema`

工具调用统一结果：

```json
{
  "tool_name": "read_network_config",
  "status": "succeeded",
  "summary": "network adapters listed",
  "output": {},
  "error": null,
  "duration_ms": 120
}
```

执行规则：

- 工具执行前必须先过策略引擎
- 高风险工具执行前必须绑定审批单
- 工具超时必须可中断
- 工具结果必须先落本地审计，再决定是否上报

#### 12.11.7 诊断引擎详细链路

诊断引擎固定为 5 个步骤，便于后续代码生成时形成稳定 orchestrator。

步骤：

1. `IntentParse`
  - 解析问题类型、环境范围、风险偏好
2. `EvidencePlan`
  - 选择要调用的只读工具集合
3. `EvidenceCollect`
  - 并行或串行执行工具
4. `Reasoning`
  - 汇总证据，生成诊断结论和修复建议
5. `ActionDraft`
  - 若需要修改，则生成候选动作与审批单

输出结构：

- `problem_type`
- `confidence`
- `findings`
- `recommended_actions`
- `approval_required`
- `next_step`

#### 12.11.8 审批执行链路

审批不是单独功能，而是 Agent Core 执行链中的核心拦截层。

执行链路：

```text
UserInput
    |
    v
DiagnosisPlan
    |
    v
ActionDraft
    |
    v
PolicyCheck
    |
    v
ApprovalDraft
    |
    v
WaitDecision
    |
    v
ExecuteAction
    |
    v
VerifyResult
    |
    v
AuditRecord
```

关键规则：

- 没有审批通过的动作不得进入执行器
- 审批过期后必须重新生成新审批单
- 执行器只能接收结构化动作，不直接接收 LLM 原始文本

#### 12.11.9 治理引擎详细设计

为了体现“环境治理”而不只是“聊天修复”，治理引擎固定在 Agent Core 内常驻。

治理引擎职责：

- 周期采集环境快照
- 对比基线
- 识别配置漂移
- 生成治理建议
- 触发低风险诊断会话

首批治理能力：

- 网络代理基线
- DNS 基线
- 端口监听基线
- 运行时版本基线
- 容器服务状态基线

调度方式：

- 启动后首次采集
- 固定周期采集
- 关键事件触发采集
- 用户手动触发重新基线

#### 12.11.10 Agent Core 任务调度模型

为了避免后续代码生成后 goroutine 失控，任务分类固定如下。

任务类型：

- 前台交互任务
  - 会话消息处理
  - 审批等待
- 后台常驻任务
  - 心跳
  - 配置同步
  - 审计 flush
  - 基线采集
  - 漂移检测
- 临时执行任务
  - 工具调用
  - 修复动作

调度原则：

- 每类任务都要有独立超时控制
- 会话内任务必须带 `session_id`
- 高风险任务必须可取消
- 后台治理任务不得阻塞前台诊断会话

#### 12.11.11 Agent Core 错误模型

后续代码生成时，错误分类固定如下：

- `bootstrap_error`
- `config_error`
- `device_error`
- `session_error`
- `policy_error`
- `approval_error`
- `tool_error`
- `governance_error`
- `transport_error`

错误对象字段：

- `code`
- `message`
- `retryable`
- `temporary`
- `cause`
- `metadata`

这样有利于：

- 前端展示
- 审计记录
- 平台汇总
- 自动重试判断

#### 12.11.12 Agent Core 代码生成边界

为了让后续可以稳定生成 `agent-core` 代码，以下边界固定生效：

- `runtime` 只能依赖模块接口，不直接依赖具体工具实现
- `diagnosis` 不直接操作数据库，只通过 store 接口读写
- `tools` 不直接做审批判断，审批统一在 policy + approval 层
- `llm/router` 不直接执行工具，只输出计划与结构化建议
- `audit` 不依赖前端 UI
- `governance` 与 `session` 可以共享工具层，但不能共享状态机

### 12.12 前后端与 Agent 代码生成顺序更新

基于当前细化程度，代码生成顺序固定如下：

1. 共享类型、错误码、配置协议
2. MySQL schema、迁移、repository 接口
3. `platform-api`、`session-gateway`、`job-runner` 三个后端服务骨架
4. Agent Core 的 bootstrap、config、device、store
5. 会话状态机和 WebSocket 网关
6. 诊断引擎、策略引擎、审批引擎
7. 首批只读工具与审计上报
8. 治理引擎与基线采集
9. 控制台前端与桌面 UI
10. 低风险修复工具、Webhook、私有化增强

### 12.13 部署、运维与验收规范

为了保证后续生成出来的不是“代码堆”，而是“可部署项目”，首版必须额外生成以下资产：

- `docker-compose.yml`
- `.env.example`
- `Makefile` 或等价启动脚本
- migration 执行脚本
- seed 数据脚本
- 健康检查脚本
- smoke test 脚本
- 平台部署说明
- Agent 本地运行说明

### 12.13.0 非功能目标

首版非功能目标固定如下：

- 单实例平台支持：
  - `1000` 台已注册设备
  - `200` 台同时在线设备
  - `50` 个同时活跃会话
- 审计数据默认保留：
  - 在线查询 `180` 天
  - 归档保留 `1` 年
- 分发包构建目标：
  - 常规引导包在 `5` 分钟内完成
- 恢复目标：
  - `RTO <= 4` 小时
  - `RPO <= 15` 分钟

这些目标用于约束首版架构，不等价于无限扩展能力承诺。

#### 12.13.1 Docker Compose 约束

`docker-compose.yml` 必须至少包含：

- `console-web`
- `platform-api`
- `session-gateway`
- `job-runner`
- `mysql`
- `redis`
- `minio`

必须定义：

- 网络
- 数据卷
- depends_on
- healthcheck
- restart 策略
- 时区与日志输出策略

`docker-compose.yml` 结构示例：

```yaml
version: "3.9"

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: envnexus
    volumes:
      - ./volumes/mysql:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]

  redis:
    image: redis:7
    volumes:
      - ./volumes/redis:/data

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - ./volumes/minio:/data

  platform-api:
    build:
      context: ../..
      dockerfile: deploy/docker/platform-api.Dockerfile
    env_file:
      - .env
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_started
      minio:
        condition: service_started
    ports:
      - "8080:8080"

  session-gateway:
    build:
      context: ../..
      dockerfile: deploy/docker/session-gateway.Dockerfile
    env_file:
      - .env
    depends_on:
      - platform-api
    ports:
      - "8081:8081"

  job-runner:
    build:
      context: ../..
      dockerfile: deploy/docker/job-runner.Dockerfile
    env_file:
      - .env
    depends_on:
      - platform-api
      - redis
      - mysql

  console-web:
    build:
      context: ../..
      dockerfile: deploy/docker/console-web.Dockerfile
    env_file:
      - .env
    depends_on:
      - platform-api
    ports:
      - "3000:3000"
```

约束：

- 所有业务服务必须通过 `env_file` 或环境变量注入配置
- 所有容器日志必须输出到标准输出，不依赖容器内文件查看
- 所有数据目录必须映射到宿主机卷
- 所有服务镜像必须能够在仓库根目录通过固定 `Dockerfile` 路径构建

#### 12.13.2 启动顺序约束

默认启动顺序：

1. `mysql`
2. `redis`
3. `minio`
4. migration job
5. `platform-api`
6. `session-gateway`
7. `job-runner`
8. `console-web`

规则：

- migration 未完成时，业务服务不得进入 ready
- `console-web` 不得在 `platform-api` 未 ready 时宣称可用
- `session-gateway` 未拿到平台配置前不得接受业务会话
- `job-runner` 未拿到数据库和 `Redis` 连接前不得开始消费任务

#### 12.13.3 日志与观测规范

首版日志必须统一为结构化日志，至少包含：

- `timestamp`
- `level`
- `service`
- `request_id`
- `tenant_id`
- `device_id`
- `session_id`
- `trace_id`
- `approval_id`
- `job_id`
- `error_code`
- `message`

首版指标和观测最少覆盖：

- HTTP 请求数和错误率
- `WebSocket` 在线连接数
- 设备激活成功率
- 审批通过率
- 工具执行成功率
- 审计上报失败率

功能可观测性必须按以下分层固定：

- 日志
  - 面向排障，记录结构化上下文与错误细节
- Metrics
  - 面向容量、稳定性、SLO 与告警
- Trace
  - 面向跨服务链路定位
- Audit
  - 面向合规、审批、回滚与责任追踪
- 业务事件
  - 面向产品运营，例如下载、激活、发布、回滚、导出
- 用户操作事件
  - 面向界面行为分析，例如点击发布、确认审批、导出诊断包

首版核心业务链路必须具备端到端观测：

1. 控制台登录
2. 租户初始化
3. 下载链接签发
4. 分发包构建
5. 设备首次激活
6. 设备心跳与配置拉取
7. `WebSocket` 会话建立与结束
8. 审批生成、确认、拒绝、超时
9. 工具执行与回滚
10. 审计上报与归档

统一关联规则：

- 单次 HTTP 请求必须生成 `request_id`
- 跨服务链路必须传递 `trace_id`
- 审批、任务、会话、设备、租户必须分别具备稳定主标识
- 任一错误日志都应至少能关联到 `tenant_id`、`service`、`error_code`
- 发布、回滚、导出操作必须可关联到操作人和构建对象

首版 SLI / SLO 下限固定如下：

- 设备激活成功率：`>= 99%`
- 配置拉取成功率：`>= 99%`
- 审计入库成功率：`>= 99.9%`
- 分发包构建成功率：`>= 95%`
- 关键审批链路成功率：`>= 99%`
- 平台核心 API 可用性：`>= 99.5%`

首版告警下限固定如下：

- `5` 分钟内设备激活失败率持续高于阈值
- `WebSocket` 连接数异常骤降
- 审计事件积压超过阈值
- 分发包构建连续失败
- 回滚失败或回滚后错误率未恢复
- `job-runner` 队列持续堆积
- `platform-api`、`session-gateway`、`agent-core` 出现高频同类错误码

本地端可观测性规则固定如下：

- `agent-core` 必须记录启动状态流、配置版本、心跳结果、审批状态变化、工具执行摘要
- `agent-desktop` 必须记录窗口启动、UI 崩溃、关键页面错误、诊断包导出动作
- 本地日志查看入口默认展示脱敏摘要，不展示敏感明文
- 本地诊断包必须支持一键导出，并可被平台侧或运维侧用于快速排障
- 诊断包默认不包含密钥、明文 token 和未脱敏命令输出

#### 12.13.4 备份与恢复下限

首版必须给出最小备份恢复策略：

- `MySQL` 全量备份
- `MinIO` 包与审计归档备份
- 平台配置导出能力
- 设备撤销名单恢复能力

恢复目标：

- 平台重建后可恢复租户、策略、模型和审计查询能力
- 设备重新连接后能够识别撤销状态和最新配置版本
- 分发包元数据与构建记录必须可恢复
- 租户默认分发目标和激活令牌状态必须可恢复

#### 12.13.5 发布治理与灰度规则

为了让首版不仅“能发版”，而且“可运营地发版”，发布治理规则固定如下：

release channel 固定为：

- `dev`
  - 仅用于开发与联调
- `beta`
  - 用于内测租户或试点设备组
- `stable`
  - 用于默认生产发布
- `lts`
  - 用于私有化客户或稳定维护窗口

灰度发布规则固定如下：

- 默认分发目标切换必须以 `DownloadPackage` 为单位进行
- 灰度范围必须支持按租户、设备组、`AgentProfile`、平台版本窗口控制
- 任意灰度发布都必须保留上一稳定版本作为回滚目标
- 未完成灰度观察窗口前，不得自动清理上一版本构建产物

回滚触发条件至少包括：

- 激活成功率明显下降
- 审计上报失败率持续升高
- 构建后客户端启动失败率超阈值
- 高优先级错误码在短时间内集中出现
- 用户侧关键链路异常导致无法诊断或无法审批

失败处理规则固定如下：

- 构建失败必须支持重试、取消或标记人工处理
- 发布失败必须保留失败原因、构建对象、触发人和最近一次变更摘要
- 回滚只切换默认分发目标，不覆盖历史审计和历史构建记录
- 任何手工干预发布目标的动作都必须进入审计链

三端发布顺序固定如下：

1. 平台后端
2. `console-web`
3. `agent-core`
4. `agent-desktop`

兼容闸门固定如下：

- 未通过兼容性检查不得把新版本设为默认分发目标
- `agent-desktop` 升级不得领先其兼容的 `agent-core` 主版本
- 平台升级后如检测到旧客户端落出兼容窗口，必须提示切换兼容包或安排升级

#### 12.13.6 发布验收清单

每次版本发布前至少要通过：

- 单元测试
- repository / service 层基础测试
- 关键状态机测试
- API 契约测试
- `WebSocket` 事件顺序测试
- `Docker Compose` 启动 smoke test
- 一个 demo Agent 激活与诊断回归测试
- 发布链路 dashboard 必须可查看
- 核心告警规则已启用
- 关键链路必须具备 Trace 或等价关联能力

## 13. 分阶段实施路线

### Phase 1：开放平台 MVP

目标：

- 多租户控制台
- 模型与策略配置
- 租户专属下载链接
- 设备首次激活
- Windows/Linux Agent 下载即用

验收标准：

- 单机 `Docker Compose` 可启动
- 控制台可登录并完成基础配置
- 下载链接可生成且可撤销
- Agent 可激活并出现在设备列表中
- 激活链路基础日志、指标和错误码必须可观测

### Phase 2：只读诊断与本地交互

目标：

- 本地聊天 UI
- 只读诊断工具集
- `WebSocket` 实时会话
- 审计日志

验收标准：

- 至少 `5` 个只读工具可稳定运行
- 诊断链路可输出结构化 findings
- 会话事件可通过 `WebSocket` 完整流转
- 审计事件必须可在平台检索
- 本地诊断日志和诊断包导出必须可用

### Phase 3：审批式修复

目标：

- 增加低风险修复工具
- 批准前展示变更预览
- 支持回滚点

验收标准：

- 至少 `3` 个低风险修复工具可用
- 所有修复动作都必须经过审批状态机
- 执行失败能给出结构化错误和回滚结果
- 审计中可关联审批单、执行记录和会话
- 审批与回滚链路具备端到端关联 ID

### Phase 4：Webhook 与企业接入

目标：

- 支持外部系统事件驱动
- 与工单/监控系统联动
- 增强租户策略和权限管理

验收标准：

- `Webhook` 签名校验和幂等处理可用
- 外部事件只能触发诊断，不得绕过本地审批
- 必须与至少一种外部系统完成 demo 联调
- 关键链路告警与业务事件可接入外部系统

### Phase 5：Private 模式与私有化发行

目标：

- 平台私有化部署
- 内网模型接入
- 私有密钥管理
- 更强的企业设备治理能力

验收标准：

- 私有化版本沿用相同对象模型与协议
- 不依赖公有云即可完成激活、配置与审计闭环
- 支持企业内网模型与密钥注入
- 私有化模式下必须具备本地可观测与审计归档闭环

## 14. 成功指标

首版成功指标固定如下：

产品链路指标：

- 下载到激活成功率
- 首次诊断成功率
- 问题平均定位时间
- 审批通过率
- 修复后问题解决率
- 误操作率
- 回滚成功率
- 设备在线率

发布治理指标：

- 分发包构建成功率
- 灰度发布成功率
- 发布后 `24` 小时关键错误率
- 回滚触发后恢复时间

可观测性指标：

- 关键链路日志覆盖率
- 关键链路 Trace 关联率
- 审计入库延迟
- 告警误报率与漏报率
- 诊断包导出可用率

经营与成本指标：

- 模型调用成本
- 租户续费或留存率
- 审计导出响应时长
- 私有化客户升级成功率

## 15. 最大风险与应对

### 风险

- 模型给出错误修复建议
- Hosted 模式下密钥托管风险
- 云端策略与本地安全策略冲突
- 下载链接泄露导致越权激活
- 跨平台执行结果不一致
- 私有化版本与公有云版本能力分叉

### 应对

- 执行层强审批
- 高危工具默认关闭
- 配置变更前显示 diff
- 敏感数据加密与脱敏
- 下载链接设置短有效期和撤销能力
- 本地策略优先级高于云端
- 工具适配层按平台能力收敛
- 公有云和私有化共享同一核心协议与对象模型

## 16. 命名建议

首选名称：`EnvNexus`

命名理由：

- `Env` 直接表达环境、系统环境与运行环境
- `Nexus` 强调中枢、连接与治理中心
- 比偏“守护”的命名更适合环境治理型平台
- 适合开放平台、终端 Agent 和私有化产品线统一命名

备选名称：

- `AegisOne`
- `EnvGuardian`
- `HostSage`
- `Orion Shield`

## 17. 最终建议

该产品的最终形态应理解为：

- 平台负责配置、分发、设备身份与多模型策略
- Agent 负责本地探测、审批式执行与审计
- 交互协议统一支持本地、`WebSocket`、`Webhook` 和私有化网关

如果要既追求最终形态，又保证能落地，建议路线是：

- 先搭出平台、下载链接和终端激活闭环
- 再做只读诊断与实时对话
- 然后逐步开放审批式修复
- 最后再做完整私有化和高级企业能力

最终统一建议：

- 商业品牌名：`EnvNexus`
- 中文品牌名：`环枢`
- 英文缩写：`ENX`
- 终端产品：`EnvNexus Agent`
- 平台产品：`EnvNexus Platform`
- 私有化版本：`EnvNexus Private`

这是从“好想法”走向“可交付产品”最稳妥的路径。
