# EnvNexus 产品需求文档 (PRD)

> **文档版本**：v2.0 (详细版)
> **产品名称**：环枢 (EnvNexus)
> **文档状态**：已冻结（对应 MVP 及 Phase 1-6 核心需求）

---

## 1. 引言

### 1.1 编写目的
本文档旨在全面、详细地定义 EnvNexus 平台的功能规格、业务规则、状态机、异常处理及非功能需求。本文档是研发编码、QA 编写测试用例、UI/UX 设计的唯一标准依据。

### 1.2 目标用户与角色定义 (RBAC)

系统采用严格的基于角色的访问控制（RBAC），具体角色及权限边界如下：

| 角色名称 | 适用对象 | 权限边界说明 |
| :--- | :--- | :--- |
| **System Admin** | 平台运维方 | 管理全局配置、License 分发、跨租户监控。不可查看租户内具体业务数据。 |
| **Tenant Owner** | 企业 IT 负责人 | 拥有本租户最高权限。可管理子用户、配置策略 (Policy)、配置模型 (Model)、生成下载链接、查看全局审计。 |
| **Tenant Admin** | IT 部门管理层 | 同 Owner，但不可删除租户或转移 Owner 权限。 |
| **Operator** | IT 支持专家 | 可查看设备列表、发起远程诊断会话、下发修复脚本。**不可**修改全局安全策略和模型配置。 |
| **Auditor** | 安全/合规专员 | 仅拥有只读权限，可查看和导出审计日志 (Audit Events)、查看策略快照。 |
| **End User** | 终端员工 | 无 Web 控制台权限。仅在本地 Agent Desktop 交互，拥有对自己设备的**最终审批权**。 |

---

## 2. 用户故事 (User Stories)

*   **US-01 [分发]**：作为 Tenant Admin，我希望能够选择操作系统并一键生成带有本企业配置的安装包下载链接，以便员工下载后无需配置即可自动接入。
*   **US-02 [诊断]**：作为 End User，我希望在遇到“Docker 无法启动”时，能在桌面端直接提问，Agent 能自动收集本地日志并告诉我原因。
*   **US-03 [修复]**：作为 Operator，我希望在远程协助时，系统能拦截高危命令（如 `rm -rf`）并强制要求终端用户点击同意，以免引发安全纠纷。
*   **US-04 [审计]**：作为 Auditor，我希望能在月末导出一份包含所有终端变更记录的报表，并且报表中的敏感信息（如员工电脑用户名）已被自动脱敏。

---

## 3. 详细功能需求规格

### 3.1 租户与身份管理模块
#### 3.1.1 租户 (Tenant)
*   **字段要求**：租户名称（必填，最长 64 字符）、状态（Active/Suspended）、配额限制（最大设备数）。
*   **业务规则**：
    *   租户被挂起 (Suspended) 时，该租户下所有设备的 WebSocket 连接必须被强制断开，且拒绝新的登录和注册。

#### 3.1.2 认证与鉴权 (Auth)
*   **认证方式**：采用 JWT (JSON Web Token)。
*   **令牌体系**：
    *   `Access Token`：有效期 2 小时，用于常规 API 请求。
    *   `Refresh Token`：有效期 7 天，用于换取新的 Access Token。
    *   `Device Token`：设备专属，有效期 30 天，支持服务端强制吊销 (Revoke) 和轮换 (Rotate)。

### 3.2 策略与模型配置模块 (Profiles)
#### 3.2.1 Model Profile (模型配置)
*   **支持的 Provider**：OpenAI, Anthropic, DeepSeek, Gemini, OpenRouter, Ollama (本地)。
*   **业务规则**：必须支持配置 `BaseURL` 和 `API Key`，以便企业接入内网代理或私有化部署的大模型。API Key 在数据库中必须加密存储。

#### 3.2.2 Policy Profile (安全策略)
*   **策略格式**：JSON 格式的规则引擎配置。
*   **拦截维度**：
    *   `allowed_tools`：允许调用的工具白名单（如仅允许 `network.ping`, `system.info`）。
    *   `blocked_paths`：禁止修改的文件路径（如 `C:\Windows\System32\*`, `/etc/*`）。
    *   `require_approval`：强制要求审批的动作级别（如 `High`, `Medium`）。

### 3.3 客户端分发与激活模块
#### 3.3.1 零编译安装包生成 (EOF Injection)
*   **前置条件**：系统 (MinIO) 中已存在通用的 Base Package。
*   **生成流程**：
    1.  用户在前端选择 Platform (Windows/Linux/macOS) 和 Arch (amd64/arm64)。
    2.  后端生成一个 `EnrollmentToken`（设置最大使用次数和过期时间）。
    3.  Job Runner 异步读取 Base Package，将 `{"tenant_id":"xxx", "token":"xxx", "api_url":"xxx"}` 序列化后追加到文件末尾（EOF）。
    4.  生成预签名下载链接 (Presigned URL，有效期 1 小时) 返回给前端。

#### 3.3.2 设备静默激活 (Enrollment)
*   **激活流程**：Agent 首次启动时，自解析 EOF 配置，向 `/agent/v1/enroll` 发起注册。
*   **状态流转**：注册成功后，设备状态变为 `Active`，并获取到专属 `Device Token`，随后建立 WebSocket 长连接。

### 3.4 诊断与修复闭环模块 (核心)
#### 3.4.1 会话管理 (Session)
*   **会话生命周期**：`Created` -> `Active` -> `Closed`。
*   **心跳机制**：Agent 每 30 秒发送一次 Ping，网关 90 秒未收到则标记设备为 `Offline`。

#### 3.4.2 审批状态机 (Approval Workflow)
任何非只读 (Non-ReadOnly) 的工具调用，必须触发审批流。
*   **状态定义**：
    *   `Pending_User`：已推送到终端，等待用户点击。
    *   `Approved`：用户已点击同意。
    *   `Denied`：用户已点击拒绝。
    *   `Expired`：超过 10 分钟用户未操作，系统自动标记为过期。
    *   `Executing`：Agent 正在执行。
    *   `Completed` / `Failed`：执行结束。
*   **约束**：状态流转必须是单向的，不可逆。执行结果必须附带 stdout/stderr 输出。

### 3.5 审计与合规模块 (Audit)
#### 3.5.1 审计日志记录
*   **触发条件**：登录登出、配置修改、会话创建、审批动作、工具执行结果。
*   **数据结构**：必须包含 `EventID`, `TenantID`, `DeviceID`, `UserID`, `Action`, `RiskLevel`, `Timestamp`, `Payload`。
*   **落盘机制**：Agent 产生的审计日志先存入本地 SQLite，每 15 秒批量上报云端。云端 Job Runner 每小时将 MySQL 中的审计数据打包为 JSONL 归档到 MinIO。

#### 3.5.2 脱敏导出 (PII Redaction)
*   **业务规则**：导出审计日志时，必须支持勾选“脱敏”。开启后，Payload 中的 MAC 地址、内网 IP、Windows 用户名等字段需被替换为 `[REDACTED]`。

---

## 4. 异常处理与边界条件

### 4.1 网络异常降级 (Offline Mode)
*   **场景**：Agent 所在设备断网或无法连接云端 Platform。
*   **系统行为**：
    1. Agent Desktop 顶部显示醒目的“离线模式 (Offline)”横幅。
    2. 禁用所有需要云端 LLM 的自动分析功能。
    3. 仅开放本地只读工具（如收集系统信息、导出日志包）。
    4. 允许用户一键导出加密的 `.zip` 诊断包，以便通过 U 盘拷贝给 IT 人员。

### 4.2 存储组件故障回退 (Storage Fallback)
*   **场景**：云端 MinIO 对象存储服务宕机。
*   **系统行为**：Job Runner 的 `audit_flush` 任务在检测到 MinIO 不可用时，自动触发 Fallback 机制，将审计归档文件写入本地宿主机的磁盘目录（如 `/var/lib/envnexus/audit_fallback/`），并在日志中触发 `Critical` 告警。

### 4.3 审批并发冲突
*   **场景**：同一个会话中，连续下发了两个冲突的修复脚本。
*   **系统行为**：Agent Core 必须实现任务队列锁。前一个审批未完结（未 Approved/Denied/Expired）前，新的高危请求必须排队或直接被拒绝（返回 `ErrBusy`）。

---

## 5. 非功能需求 (NFRs)

### 5.1 性能与容量指标
*   **连接数**：单台 Session Gateway 节点需支撑至少 **5,000** 个并发 WebSocket 连接。
*   **分发打包**：基于 EOF 注入的安装包生成时间必须 **< 500 毫秒**。
*   **API 响应**：95% 的常规 HTTP API 响应时间必须 **< 200 毫秒**。
*   **审计吞吐**：系统需支持每秒至少 **500 条** 审计日志的写入不丢失。

### 5.2 安全基线
*   **密码存储**：用户密码必须使用 `bcrypt` 算法加盐哈希存储。
*   **防爆破**：连续 5 次登录失败，账号锁定 15 分钟。
*   **CORS**：严格限制跨域请求，必须通过 `ENX_CORS_ALLOWED_ORIGINS` 环境变量配置白名单。
*   **依赖安全**：所有第三方库（Go modules, npm packages）必须通过已知漏洞扫描 (CVE Scan)。

### 5.3 数据保留策略 (Retention Policy)
*   **在线审计数据 (MySQL)**：保留 180 天。
*   **归档审计数据 (MinIO)**：保留 3 年。
*   **过期下载链接**：过期后 24 小时由 Job Runner 自动硬删除。

---

## 6. UI/UX 规范与约束

### 6.1 Agent Desktop (终端)
*   **静默原则**：软件启动后默认最小化到系统托盘，不弹窗、不抢焦点。
*   **审批强提醒**：当收到高危审批请求时，窗口必须置顶闪烁。审批按钮必须有 **3 秒的倒计时防误触** 机制。
*   **国际化**：必须支持跟随操作系统语言自动切换（中/英）。

### 6.2 Console Web (云端)
*   **破坏性操作**：删除租户、吊销设备等操作，必须弹出二次确认框，并要求用户手动输入租户名称或设备名称以防误删。
*   **响应式**：管理后台需适配 1080p 及以上分辨率，不强制要求适配移动端手机屏幕。