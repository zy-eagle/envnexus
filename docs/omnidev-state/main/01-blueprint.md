# Phase 1 Blueprint: EnvNexus 全功能增量路线图

> **基准**: 产品白皮书 v4.0 (`docs/product-manual.md`)
> **方法**: 对比产品愿景中的每项能力与当前代码实现，识别 Gap，输出增量开发蓝图
> **原则**: 在现有功能基础上增量增强，不重写、不破坏已有功能

---

## 1. 功能 Gap 全景矩阵

### 1.1 已实现功能基线

| 组件 | 已实现能力 | 完成度 |
|------|-----------|--------|
| **platform-api** | 多租户管理、用户/RBAC、设备管理、会话管理、审计事件、模型/策略/Agent配置、客户端分发（含构建流水线）、环境治理（基线+漂移）、远程命令任务（含NL生成+审批策略+风险评估）、Webhook、IM通知、License、飞书集成 | 85% |
| **session-gateway** | WebSocket 会话、命令转发、Redis pub/sub、健康检查 | 90% |
| **job-runner** | Token/Link/Session清理、审计归档、审批过期、安装包构建、治理扫描 | 90% |
| **agent-core** | Chat Loop（LLM迭代工具调用）、诊断引擎（4步管线）、34个结构化工具、策略引擎（逐工具审批）、治理引擎（基线+漂移）、多Provider LLM路由、OTA自更新、本地SQLite、设备激活/注册 | 70% |
| **agent-desktop** | 5页面（仪表盘/诊断对话/待审批/历史会话/设置）、Chat UI、逐工具审批卡片、系统托盘、加密配置、OTA更新 | 70% |
| **console-web** | 全管理后台（租户/用户/角色/设备/会话/审计/配置/分发/治理/命令任务/审批/策略）、i18n | 85% |
| **deploy** | 智能部署脚本、Docker Compose、交叉编译、Electron打包、MinIO上传 | 95% |

### 1.2 产品白皮书 vs 已实现 — Gap 分析

| # | 产品能力（白皮书） | 当前状态 | Gap 描述 | 优先级 |
|---|-------------------|---------|---------|--------|
| **G1** | 自然语言转受控执行（核心场景一） | 🟡 部分 | 有NL→命令生成（command task），但缺少**执行计划（DAG）**、**计划级审批**、**有序执行+回滚** | P0 |
| **G2** | 工具优先的执行漏斗 | 🟡 部分 | Chat Loop中LLM自行调用工具，但**无执行计划概念**、无降级漏斗（工具→Shell）、无预算控制 | P0 |
| **G3** | 分层审批（L0-L3） | 🟡 部分 | 策略引擎支持逐工具审批，但**无计划级审批**、无L0-L3分层矩阵、无步骤级二次确认 | P0 |
| **G4** | 自动回滚 | 🔴 缺失 | 无状态快照、无回滚策略生成、无逆序恢复 | P0 |
| **G5** | 确切命令的受控下发（核心场景二） | 🟢 已实现 | command task + NL生成 + 审批策略 + 风险评估 + WebSocket下发 | — |
| **G6** | 智能诊断与计划式修复（场景三） | 🟡 部分 | 诊断引擎存在但是4步固定管线，缺少**复杂度评估**、**分层证据收集**、**迭代推理**、**诊断→修复计划自动衔接** | P1 |
| **G7** | 远程文件系统查看与下载（场景五） | 🟡 部分 | agent-core有`dir_list`/`file_info`/`file_tail`工具，但**无文件下载能力**、**无文件审批流程**、**无平台端文件浏览UI** | P1 |
| **G8** | 本地环境Watchlist（场景四） | 🔴 缺失 | 治理引擎仅检测hostname/网卡/env变化，**无自然语言→WatchItem拆解**、**无巡检调度器**、**无内置规则包**、**无健康评分** | P1 |
| **G9** | 海量终端批量干预（场景六） | 🟡 部分 | command task支持单设备下发，但**无设备组选择**、**无批量下发**、**无灰度执行**、**无全局执行大盘** | P2 |
| **G10** | 多模态输入（截图诊断） | 🔴 缺失 | LLM Router的Message.Content是string，**不支持图像**、**无Vision Provider适配**、**Desktop无截图上传** | P2 |
| **G11** | 防失联灰度自更新（OTA） | 🟢 已实现 | agent-core有check-update/download/apply，Desktop有electron-updater，平台有版本管理 | — |
| **G12** | 全链路审计 | 🟢 已实现 | 诊断/工具调用/审批/命令执行全流程审计事件 | — |
| **G13** | 配置加密 | 🟢 已实现 | AES-256-GCM + DPAPI/Keychain | — |
| **G14** | 健康态势仪表盘（Platform） | 🟡 部分 | console-web有overview页面但仅显示设备数/会话数，**无健康评分聚合**、**无异常趋势**、**无修复状态** | P2 |
| **G15** | 策略配置中心（工具权限细粒度） | 🟡 部分 | 策略配置有只读/完整模式，但**无工具级白名单/黑名单**、**无路径级文件访问控制** | P2 |
| **G16** | 规则与基线管理（平台下发） | 🟡 部分 | 治理引擎有基线采集，但**无平台端规则配置UI**、**无规则下发到终端** | P2 |
| **G17** | 学习型规则（从修复中提取） | 🔴 缺失 | 无历史修复模式提取、无自动建议监控 | P3 |

### 1.3 Gap 优先级分层

```
P0 (核心差异化 — 必须优先)
├── G1: 执行计划引擎 (DAG)
├── G2: 工具优先执行漏斗
├── G3: 分层审批 (L0-L3)
└── G4: 自动回滚

P1 (产品完整性 — 紧随其后)
├── G6: 智能诊断升级
├── G7: 远程文件取证
└── G8: Watchlist 主动巡检

P2 (规模化 & 体验)
├── G9: 批量终端干预
├── G10: 多模态截图诊断
├── G14: 健康态势仪表盘
├── G15: 细粒度工具权限
└── G16: 平台规则下发

P3 (智能化演进)
└── G17: 学习型规则
```

---

## 2. 增量架构设计

### 2.1 修复计划引擎 (G1+G2+G3+G4)

这是产品核心差异化能力，将当前的"逐工具审批"升级为"计划级审批+有序执行+自动回滚"。

```mermaid
graph TB
    subgraph "输入层"
        A1[用户自然语言]
        A2[管理员确切命令]
        A3[诊断引擎输出]
        A4[Watchlist 告警]
    end

    subgraph "计划生成层 (新增 remediation/)"
        B1[执行漏斗<br/>工具匹配 → Shell降级]
        B2[LLM 计划生成器<br/>结构化JSON输出]
        B3[引擎校验<br/>工具存在性/风险等级/DAG无环]
        B4[回滚策略注入<br/>基于工具元数据]
    end

    subgraph "审批层 (扩展 policy/)"
        C1[L0: 自动通过]
        C2[L1: 计划级审批]
        C3[L2: 计划+执行前确认]
        C4[L3: 逐步审批]
    end

    subgraph "执行层 (新增 remediation/executor)"
        D1[DAG 拓扑排序]
        D2[步骤前置检查]
        D3[状态快照]
        D4[工具执行]
        D5[执行后验证]
        D6[失败→逆序回滚]
    end

    A1 --> B1
    A2 --> B1
    A3 --> B1
    A4 --> B1
    B1 --> B2
    B2 --> B3
    B3 --> B4
    B4 --> C1
    C1 --> D1
    C2 --> D1
    C3 --> D1
    C4 --> D1
    D1 --> D2
    D2 --> D3
    D3 --> D4
    D4 --> D5
    D5 -->|失败| D6
    D5 -->|成功| D1
```

**核心数据结构**:

```go
type RemediationPlan struct {
    PlanID      string            `json:"plan_id"`
    Summary     string            `json:"summary"`
    RiskLevel   string            `json:"risk_level"`   // 整体最高风险
    Steps       []RemediationStep `json:"steps"`
    Verification *ToolCheck       `json:"verification"` // 最终验证
}

type RemediationStep struct {
    StepID      int                    `json:"step_id"`
    Description string                 `json:"description"`
    ToolName    string                 `json:"tool_name"`
    Params      map[string]interface{} `json:"params"`
    RiskLevel   string                 `json:"risk_level"`
    DependsOn   []int                  `json:"depends_on"`
    Rollback    *RollbackAction        `json:"rollback"`
    Verify      *ToolCheck             `json:"verify"`
    Timeout     time.Duration          `json:"timeout"`
}
```

**与现有代码的集成点**:
- `agent/loop.go`: 新增可选的 `RemediationPlanner`，检测到修复建议时生成计划
- `policy/engine.go`: 新增 `CheckPlan` 方法，保留现有 `Check` 不变
- `api/server.go`: 新增 `/local/v1/plan/*` 端点
- Desktop: 新增 `plan_generated` 等 SSE 事件处理

### 2.2 智能诊断升级 (G6)

在现有4步管线基础上增强，不改变外部接口。

```
现有管线:  意图分类 → 工具映射 → 并行采集 → LLM推理
                ↓           ↓           ↓          ↓
增强后:    意图分类 → 复杂度评估 → 分层采集 → 迭代推理
           (保留)    (新增)      (增强)     (增强)
```

- **复杂度评估器**: Simple/Moderate/Complex/Critical，决定工具预算和推理深度
- **分层证据收集**: 第一层基础工具 → 根据结果决定第二层深入工具
- **迭代推理**: 置信度 < 阈值时请求补充证据，最多 N 轮
- **诊断→修复衔接**: `DiagnosisResult.NeedsRemediation` 自动触发计划生成

### 2.3 远程文件取证 (G7)

```mermaid
sequenceDiagram
    participant Admin as 管理员(Console)
    participant Platform as Platform API
    participant GW as Session Gateway
    participant Agent as Agent Core

    Admin->>Platform: POST /file-access/request {deviceId, path, action}
    Platform->>Platform: 创建审批请求
    Platform-->>Admin: 审批请求已创建

    Note over Agent: 终端用户收到审批通知
    Agent->>Platform: POST /file-access/approve
    Platform->>GW: 转发文件访问指令
    GW->>Agent: WS: file_access command

    alt 目录浏览
        Agent-->>GW: 目录结构 JSON
        GW-->>Platform: 转发结果
        Platform-->>Admin: 展示目录树
    else 文件预览
        Agent-->>GW: 文件内容 (文本)
        GW-->>Platform: 转发结果
        Platform-->>Admin: 展示文件内容
    else 文件下载
        Agent->>Agent: 读取文件 → 上传到 MinIO
        Agent-->>GW: 下载 URL
        GW-->>Platform: 转发 URL
        Platform-->>Admin: 提供下载链接
    end

    Platform->>Platform: 记录审计事件
```

**新增组件**:
- agent-core: `file_download` 工具 + MinIO 上传能力
- platform-api: 文件访问请求/审批 API + 审计
- console-web: 文件浏览器 UI（目录树 + 文件预览 + 下载）

### 2.4 Watchlist 主动巡检 (G8)

```
四层规则来源:
┌─────────────────────────────────────────────┐
│ 1. 用户自然语言 → LLM拆解 → 确认 → WatchItem │
│ 2. 内置规则包 (NET/SEC/PERF/DEP/SVC/CERT)   │
│ 3. 平台下发规则 (管理员配置)                   │
│ 4. 学习型规则 (修复后自动提取) [P3]            │
└─────────────────────────────────────────────┘
          ↓ 统一注册
┌─────────────────────────────────────────────┐
│         WatchItem 调度器                      │
│  按各项 Interval 调度 → 工具执行 → 条件评估    │
│  → 状态更新 → 告警生成 → 修复建议              │
└─────────────────────────────────────────────┘
```

**新增 package**: `governance/watchlist/`
- `types.go`: WatchItem, WatchCondition, WatchAlert
- `store.go`: SQLite CRUD
- `evaluator.go`: 条件评估引擎
- `decomposer.go`: LLM 自然语言拆解器
- `scheduler.go`: 巡检调度器
- `builtin_rules.go`: 9条内置规则
- `alerter.go`: 告警→修复建议闭环

### 2.5 批量终端干预 (G9)

在现有 command task 基础上扩展：

- **设备组**: 新增 `device_group` 域模型，支持按标签/部门/平台分组
- **批量下发**: command task 支持 `target_type: group`，下发到设备组
- **灰度执行**: 分批次执行（第一批 N 台 → 确认成功 → 下一批）
- **执行大盘**: console-web 新增批量任务进度页面

### 2.6 多模态截图诊断 (G10)

- `router.Message.Content` 从 `string` 改为 `interface{}`（兼容 string 和 `[]ContentPart`）
- Provider 新增 `SupportsVision() bool`
- OpenAI/Anthropic/Gemini 适配多模态
- Desktop Chat 支持粘贴/拖拽图片

---

## 3. 里程碑路线图

```mermaid
gantt
    title EnvNexus 增量开发路线图
    dateFormat  YYYY-MM-DD
    axisFormat  %m/%d

    section M1: 修复计划引擎 [P0]
    数据结构 & DAG              :m1a, 2026-04-11, 2d
    LLM 计划生成器              :m1b, after m1a, 3d
    状态快照 & 回滚             :m1c, after m1a, 2d
    DAG 执行器                  :m1d, after m1b, 3d
    分层审批扩展                :m1e, after m1a, 2d
    计划 API 端点               :m1f, after m1d, 2d
    Agent Loop 集成             :m1g, after m1f, 2d
    Desktop 计划审批 UI         :m1h, after m1g, 3d
    M1 测试                     :m1t, after m1h, 2d

    section M2: 智能诊断升级 [P1]
    复杂度评估器                :m2a, after m1t, 2d
    分层证据收集                :m2b, after m2a, 3d
    迭代推理                    :m2c, after m2b, 2d
    诊断→计划衔接               :m2d, after m2c, 2d
    M2 测试                     :m2t, after m2d, 2d

    section M3: Watchlist 主动巡检 [P1]
    数据结构 & 存储              :m3a, after m2t, 2d
    条件评估 & 调度器            :m3b, after m3a, 3d
    LLM 拆解器                  :m3c, after m3b, 2d
    内置规则包                   :m3d, after m3b, 2d
    告警→修复闭环               :m3e, after m3c, 2d
    Governance 集成             :m3f, after m3d, 2d
    API 端点                    :m3g, after m3f, 2d
    Desktop 关注页面 & 健康看板  :m3h, after m3g, 3d
    M3 测试                     :m3t, after m3h, 2d

    section M4: 远程文件取证 [P1]
    file_download 工具           :m4a, after m3t, 2d
    Platform 文件访问 API        :m4b, after m4a, 3d
    Console 文件浏览器 UI        :m4c, after m4b, 3d
    M4 测试                     :m4t, after m4c, 2d

    section M5: 多模态 + 批量干预 [P2]
    LLM 多模态消息               :m5a, after m4t, 2d
    Provider Vision 适配         :m5b, after m5a, 2d
    Desktop 截图上传             :m5c, after m5b, 2d
    设备组 & 批量下发            :m5d, after m4t, 3d
    灰度执行 & 执行大盘          :m5e, after m5d, 3d
    M5 测试                     :m5t, after m5e, 2d

    section M6: 平台增强 [P2]
    健康态势仪表盘               :m6a, after m5t, 3d
    平台规则下发                 :m6b, after m6a, 3d
    细粒度工具权限               :m6c, after m6b, 2d
    M6 测试                     :m6t, after m6c, 2d
```

---

## 4. 关键设计决策

### D1: 修复计划由谁生成？
**决策**: LLM 生成 + 引擎校验。LLM 输出结构化 JSON，引擎校验工具存在性、强制注册表风险等级、DAG 无环、注入回滚策略。

### D2: 计划审批 vs 逐步审批？
**决策**: 两层并存，策略驱动。L0 自动通过、L1 计划级、L2 计划+确认、L3 逐步。保留现有逐工具审批作为 Chat Loop 的默认行为。

### D3: 执行漏斗降级策略？
**决策**: 工具优先。计划生成时优先匹配注册工具；无匹配工具时降级为 `shell_exec`（白名单），自动提升风险等级至 L3。

### D4: 文件下载的传输方式？
**决策**: Agent 读取文件 → 上传到 MinIO（presigned URL）→ 管理员通过 URL 下载。避免大文件通过 WebSocket 传输。

### D5: Watchlist 规则的优先级？
**决策**: 平台下发 > 内置规则 > 用户自定义。冲突时高优先级覆盖低优先级的同类规则。

### D6: 批量下发的防爆机制？
**决策**: 分批次 + 成功率门槛。每批次执行完成后检查成功率，低于阈值（默认 90%）自动暂停后续批次。

---

## 5. 安全约束（不可妥协）

1. 修复计划中的所有工具必须在注册表中 — LLM 不能发明工具
2. 风险等级以注册表为准 — LLM 不能降低风险等级
3. Shell 命令强制 L3 审批 — 执行漏斗降级时自动提升
4. 回滚动作也需要审批 — 回滚不是无条件执行
5. 文件下载受路径白名单控制 — 不允许访问 `/etc/shadow` 等敏感路径
6. 批量下发必须经过审批策略 — 不允许绕过审批直接群发
7. 主动发现只触发通知，不自动执行修复 — 除非用户明确启用

---

## 6. 与之前蓝图的变化说明

| 维度 | 之前蓝图 (v1) | 本次蓝图 (v2) | 变化原因 |
|------|-------------|-------------|---------|
| 范围 | 仅 agent-core 智能治理引擎 | 全产品 Gap 分析（含文件取证、批量干预、平台增强） | 产品白皮书 v4.0 新增了文件取证、批量干预等核心场景 |
| 里程碑 | 4 个 (M1-M4) | 6 个 (M1-M6) | 新增文件取证(M4)、批量干预(M5)、平台增强(M6) |
| 远程命令 | 未涉及 | 已实现，标记为完成 | M8-M12 已在 2026-04-03 完成 |
| 修复计划引擎 | M1 | M1（保持） | 核心优先级不变 |
| 智能诊断 | M2 | M2（保持） | 核心优先级不变 |
| Watchlist | M3 | M3（保持） | 核心优先级不变 |
| 多模态 | M4 | M5（降级） | 文件取证优先级更高，多模态推迟 |
| 文件取证 | 未涉及 | M4（新增） | 产品白皮书核心场景五 |
| 批量干预 | 未涉及 | M5（新增） | 产品白皮书核心场景六 |
| 平台增强 | 部分在 M4 | M6（独立） | 健康仪表盘、规则下发、工具权限独立为一个里程碑 |
