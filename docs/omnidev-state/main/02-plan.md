---
total_tasks: 62
parallel_groups: 26
critical_path: [T1 → T3 → T5 → T7 → T8 → T13 → T14 → T15 → T18 → T20 → T22 → T24 → T25 → T30 → T33 → T35 → T38 → T40 → T43 → T46 → T48 → T51 → T54 → T56]
frontend_impact: yes
principle: 在现有功能基础上增量增强，不重写、不破坏已有功能
---

# Phase 2: EnvNexus 全功能增量开发计划

> **基于**: `01-blueprint.md` v2 — 17 项 Gap 分析、6 个里程碑
> **核心原则**: 新增优先于修改；接口兼容；行为兼容；路由追加；回归验证

---

## 现有功能基线（不可破坏）

| 模块 | 现有能力 | 增强方向 |
|------|---------|---------|
| `agent/loop.go` | Chat Loop: LLM 迭代调用工具、SSE 事件流、逐工具审批 | 新增"修复计划"分支，与现有 Chat 并存 |
| `diagnosis/engine.go` | 4 步管线: 意图分类 → 工具映射 → 并行采集 → LLM 推理 | 插入复杂度评估，Execute 改为分层采集 |
| `policy/engine.go` | Check/Resolve 审批流、Platform 同步 | 新增 CheckPlan 方法 |
| `governance/engine.go` | 基线采集、漂移检测 | 新增 WatchlistManager |
| `tools/tool.go` | Tool 接口、Registry、34 个工具 | 不改接口，新增工具实现 |
| `api/server.go` | 12+ API 端点 | 新增端点，不修改现有签名 |
| Desktop `index.html` | 5 页面、Chat UI、审批卡片 | 新增页面和 SSE 事件 |
| platform-api | 全管理 REST API + Agent API | 新增端点，不修改现有 |
| console-web | 全管理后台 20+ 页面 | 新增页面 |

---

## Milestone 1: 修复计划引擎 [P0] (G1+G2+G3+G4)

> 核心差异化能力。在现有 Chat Loop 基础上新增"修复计划"能力。

### Group 1 (parallel — no prerequisites)

- [x] **T1** [backend] 新增 `remediation/types.go` — 核心数据结构 · outputs: `apps/agent-core/internal/remediation/types.go`
- [x] **T2** [backend] 新增 `remediation/dag.go` — DAG 构建与拓扑排序 · depends: T1

### Group 2 (parallel — after Group 1)

- [x] **T3** [backend] 新增 `remediation/planner.go` — LLM 生成修复计划 · depends: T1, T2
- [x] **T4** [backend] 新增 `remediation/snapshot.go` — 状态快照管理 · depends: T1

### Group 3 (parallel — after Group 2)

- [x] **T5** [backend] 新增 `remediation/executor.go` — DAG 执行器 · depends: T2, T3, T4
- [x] **T6** [backend] 扩展 Policy Engine — 新增 `CheckPlan` · depends: T1

### Group 4 (parallel — after Group 3)

- [x] **T7** [backend] 新增修复计划 API 端点 · depends: T5, T6
- [x] **T8** [backend] Agent Loop 集成修复计划 · depends: T5, T7

### Group 5 (frontend — after Group 4)

- [x] **T9** [frontend] Desktop 修复计划 SSE 事件处理 · depends: T7, T8
- [x] **T10** [frontend] Desktop IPC 扩展 — 修复计划 · depends: T7

### Group 6 (test — after Group 5)

- [x] **T11** [test] 修复计划引擎单元测试 · depends: T3, T5
- [x] **T12** [test] 回归测试 — Chat 和诊断不受影响 · depends: T8

---

## Milestone 2: 智能诊断升级 [P1] (G6)

### Group 7 (parallel — after M1)

- [x] **T13** [backend] 复杂度评估器 · depends: T8
- [x] **T14** [backend] 分层证据收集 · depends: T13

### Group 8 (parallel — after Group 7)

- [x] **T15** [backend] 迭代推理 · depends: T14
- [x] **T16** [backend] 诊断→修复计划自动衔接 · depends: T15, T3

### Group 9 (test — after Group 8)

- [x] **T17** [test] 诊断增强回归测试 · depends: T13, T15

---

## Milestone 3: Watchlist 主动巡检 [P1] (G8)

### Group 10 (parallel — after M2)

- [x] **T18** [backend] `governance/watchlist/types.go` + `store.go` — 数据结构 & 存储 · depends: T15
- [x] **T19** [backend] 条件评估引擎 · depends: T18

### Group 11 (parallel — after Group 10)

- [x] **T20** [backend] LLM 拆解器（自然语言 → WatchItems） · depends: T18, T19
- [x] **T21** [backend] 巡检调度器 · depends: T18, T19

### Group 12 (parallel — after Group 11)

- [x] **T22** [backend] 内置规则包 · depends: T18, T21
- [x] **T23** [backend] 告警 → 修复建议闭环 · depends: T21, T3
- [x] **T24** [backend] Governance Engine 集成 Watchlist · depends: T21, T22

### Group 13 (parallel — after Group 12)

- [x] **T25** [backend] Watchlist API 端点 · depends: T20, T21, T23, T24
- [x] **T26** [backend] Bootstrap 集成 Watchlist · depends: T24

### Group 14 (frontend — after Group 13)

- [x] **T27** [frontend] Desktop "我的关注"页面 · depends: T25
- [x] **T28** [frontend] Desktop 自然语言添加关注点 · depends: T25, T27
- [x] **T29** [frontend] Desktop 告警通知 & 健康看板 · depends: T25, T27
- [x] **T30** [frontend] Desktop IPC 扩展 — Watchlist · depends: T25

### Group 15 (test — after Group 14)

- [x] **T31** [test] Watchlist 单元测试 · depends: T19, T21
- [x] **T32** [test] Governance 回归测试 · depends: T24

---

## Milestone 4: 远程文件取证 [P1] (G7)

### Group 16 (parallel — after M3)

- [x] **T33** [backend] agent-core `file_download` 工具 · depends: T25
- [x] **T34** [backend] agent-core 文件访问 API · depends: T33

### Group 17 (parallel — after Group 16)

- [x] **T35** [backend] platform-api 文件访问域模型 & 服务 · depends: T33
- [x] **T36** [backend] platform-api 文件访问 HTTP handler · depends: T35
- [x] **T37** [backend] session-gateway 文件访问事件转发 · depends: T35

### Group 18 (frontend — after Group 17)

- [x] **T38** [frontend] console-web 文件浏览器页面 · depends: T36
- [x] **T39** [frontend] console-web 文件访问审计集成 · depends: T36

### Group 19 (test — after Group 18)

- [x] **T40** [test] 文件取证端到端测试 · depends: T34, T36, T38

---

## Milestone 5: 多模态 + 批量干预 [P2] (G9+G10)

### Group 20 (parallel — after M4)

- [x] **T41** [backend] LLM Router 多模态消息 · depends: T40
- [x] **T42** [backend] Provider Vision 适配 · depends: T41
- [x] **T43** [backend] platform-api 设备组域模型 · depends: T40

### Group 21 (parallel — after Group 20)

- [x] **T44** [frontend] Desktop 截图上传（Ctrl+V 粘贴 + 拖拽 + base64 ContentPart） · depends: T42
- [x] **T45** [backend] platform-api 设备组 HTTP handler · depends: T43
- [x] **T46** [backend] command task 批量下发扩展（batch_size 分批 + 成功率门槛 + 批次延迟） · depends: T43

### Group 22 (frontend — after Group 21)

- [x] **T47** [frontend] console-web 设备组管理页面 · depends: T45
- [x] **T48** [frontend] console-web 批量任务页面增强 · depends: T46

### Group 23 (test — after Group 22)

- [x] **T49** [test] 多模态测试 · depends: T42
- [x] **T50** [test] 批量下发测试 · depends: T46

---

## Milestone 6: 平台增强 [P2] (G14+G15+G16)

### Group 24 (parallel — after M5)

- [x] **T51** [backend] platform-api 健康评分聚合 · depends: T48
- [x] **T52** [backend] platform-api 治理规则管理 · depends: T48
- [x] **T53** [backend] agent-core 接收平台规则（platform_sync.go + 心跳拉取） · depends: T52, T26

### Group 25 (parallel — after Group 24)

- [x] **T54** [backend] platform-api 细粒度工具权限 · depends: T51
- [x] **T55** [frontend] console-web 健康态势仪表盘 · depends: T51
- [x] **T56** [frontend] console-web 治理规则管理页面 · depends: T52
- [x] **T57** [frontend] console-web 策略配置增强（工具白/黑名单 + 路径访问控制） · depends: T54

### Group 26 (test — after Group 25)

- [x] **T58** [test] 健康评分聚合测试 · depends: T51
- [x] **T59** [test] 规则下发端到端测试 · depends: T53
- [x] **T60** [test] 工具权限测试 · depends: T54

---

## 数据库迁移任务

- [x] **T61** [infra] M4 数据库迁移 — file_access_requests 表 · outputs: `services/platform-api/migrations/000016_file_access.up.sql`
- [x] **T62** [infra] M5+M6 数据库迁移 — device_groups + governance_rules 表 · outputs: `services/platform-api/migrations/000017_device_groups_batch.up.sql`, `services/platform-api/migrations/000018_governance_rules_tool_perms.up.sql`

---

## 增量改造原则清单

| # | 原则 | 说明 |
|---|------|------|
| 1 | **新增优先于修改** | 优先创建新文件/新方法，最小化对现有文件的修改 |
| 2 | **接口兼容** | 不改变现有 `Tool`、`Provider`、`Engine` 接口的已有方法签名 |
| 3 | **行为兼容** | 不设置新功能时，所有现有行为保持不变 |
| 4 | **路由追加** | 新 API 端点追加到路由组，不修改现有端点 |
| 5 | **SSE 扩展** | 新事件类型追加到现有 switch，不修改现有事件 payload |
| 6 | **UI 追加** | 新页面/组件追加到现有结构，不重构现有 UI |
| 7 | **回归验证** | 每个里程碑包含回归测试任务 |
| 8 | **DDD 一致** | platform-api 新增功能遵循 domain/service/repository/handler 四层 |

---

## 里程碑交付物

| 里程碑 | 任务数 | 核心交付 | 用户可感知的变化 |
|--------|-------|---------|----------------|
| **M1** | T1-T12 (12) | 修复计划引擎 + 计划审批 UI | 诊断后看到完整修复方案，一键审批，自动回滚 |
| **M2** | T13-T17 (5) | 智能诊断升级 | 复杂问题更精准，自动衔接修复计划 |
| **M3** | T18-T32 (15) | Watchlist + 主动发现 | 自然语言定义关注点，持续巡检，健康评分 |
| **M4** | T33-T40 (8) | 远程文件取证 | 管理员远程浏览/预览/下载终端文件 |
| **M5** | T41-T50 (10) | 多模态 + 批量干预 | 截图诊断，设备组批量下发 |
| **M6** | T51-T62 (12) | 平台增强 | 健康仪表盘，规则下发，工具权限 |

---

## 关键路径

```
T1(数据结构) → T3(Planner) → T5(Executor) → T7(API) → T8(Loop集成)
→ T13(复杂度) → T14(分层证据) → T15(迭代推理)
→ T18(Watchlist数据) → T20(LLM拆解) → T22(内置规则) → T24(Governance集成) → T25(API)
→ T30(Desktop IPC) → T33(file_download) → T35(Platform文件访问) → T38(Console文件浏览器)
→ T40(测试) → T43(设备组) → T46(批量下发) → T48(Console批量页面)
→ T51(健康聚合) → T54(工具权限) → T56(Console规则管理)
```