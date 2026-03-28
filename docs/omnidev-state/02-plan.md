---
id: dual-mode-activation-plan
blueprint: dual-mode-activation
status: in_progress
created_at: 2026-03-28
---

# 开发计划：双模式安装包激活系统

## 阶段一：数据库与后端基础（优先级最高）

- [ ] **1.1** 创建数据库迁移脚本 `000006_device_binding.up.sql`
  - download_packages 新增 4 个字段
  - 新建 device_bindings 表
  - 新建 device_components 表
  - 新建 pending_devices 表
  - 新建 activation_audit_logs 表

- [ ] **1.2** 新增 Domain 实体
  - `domain/device_binding.go` — DeviceBinding, PendingDevice, DeviceComponent, ActivationAuditLog
  - `domain/package.go` — 增加 ActivationMode, ActivationKeyHash, MaxDevices, BoundCount 字段

- [ ] **1.3** 新增 DTO
  - `dto/device_binding.go` — 绑定/激活/审计相关请求和响应
  - `dto/package.go` — 增加激活相关字段到 Create/Update/Response

- [ ] **1.4** 新增 Repository
  - `repository/mysql_device_binding_repo.go` — CRUD + 组件容错匹配逻辑

- [ ] **1.5** 新增 Service
  - `service/device_binding/device_binding_service.go`
    - RegisterDevice（上报设备码）
    - ActivateDevice（auto 模式自动激活）
    - GetActivationStatus（查询激活状态）
    - BindDevice（管理员手动绑定）
    - UnbindDevice（管理员解绑）
    - ProcessHeartbeat（心跳校验）
    - ListBindings / ListAuditLogs

- [ ] **1.6** 修改 Package Service
  - CreatePackage 时生成 activation_key，存储哈希
  - 响应中一次性返回明文 activation_key
  - ListPackages / toResponse 增加新字段

## 阶段二：Agent 端 API（后端路由）

- [ ] **2.1** 新增 `handler/agent/activate_handler.go`
  - POST `/agent/v1/device-code` — 上报设备码
  - POST `/agent/v1/activate` — 自动激活
  - GET `/agent/v1/activation-status` — 查询状态

- [ ] **2.2** 增强 `handler/http/package_handler.go`
  - POST `/:pid/bind` — 管理员绑定设备码
  - DELETE `/:pid/bindings/:bid` — 管理员解绑
  - GET `/:pid/bindings` — 查看绑定列表
  - PUT `/:pid/max-devices` — 修改最大设备数
  - GET `/activation-logs` — 审计日志

- [ ] **2.3** 注册路由 `cmd/platform-api/main.go`
  - 注入 DeviceBindingRepo → DeviceBindingService → ActivateHandler
  - 注册 Agent 端公开路由 + 管理员受保护路由

- [ ] **2.4** Rate Limiting 中间件
  - 对 Agent 端 3 个 API 添加 IP 级限流

## 阶段三：安装包构建增强

- [ ] **3.1** 修改 `worker/package_build.go`
  - 读取 package 的 activation_mode
  - auto 模式：从数据库取明文 activation_key（需新增一次性读取机制或构建时传入）注入到二进制
  - manual 模式：仅注入 activation_mode
  - 注入 activation_mode 字段到 JSON payload

## 阶段四：Agent Core 激活模块

- [ ] **4.1** 新增 `internal/hwinfo/fingerprint.go`
  - 跨平台硬件采集（Windows WMI / Linux sysfs / macOS system_profiler）
  - 5 组件独立哈希 + 主指纹计算
  - 设备码生成算法（ENX-XXXX-XXXX-XXXX）

- [ ] **4.2** 新增 `internal/activation/activation.go`
  - RegisterDeviceCode — 上报设备码
  - AutoActivate — 自动激活流程
  - CheckActivationStatus — 轮询状态
  - LoadLocalActivation / SaveLocalActivation — 本地文件读写
  - ValidateLocalFingerprint — 本地容错校验

- [ ] **4.3** 修改 `internal/config/config.go`
  - 解析内嵌配置中的 activation_mode, activation_key

- [ ] **4.4** 修改 `internal/bootstrap/bootstrap.go`
  - 启动时先执行激活检查
  - 未激活 → 阻塞进入激活流程
  - 已激活 → 正常启动

- [ ] **4.5** 心跳集成
  - 在现有 WebSocket lifecycle 连接中增加 activation_heartbeat 消息
  - 处理 revoked 响应 → 清除本地激活

## 阶段五：控制台前端

- [ ] **5.1** 修改创建安装包表单
  - 新增"激活模式"单选（自动激活 / 设备码绑定）
  - 新增"最大绑定设备数"输入框
  - auto 模式创建成功后弹窗显示激活码（一次性）

- [ ] **5.2** 增强安装包列表页
  - 新增列：激活模式标签、已绑定/上限进度条
  - 操作按钮：绑定设备（manual）、查看设备、审计日志

- [ ] **5.3** 新增绑定设备弹窗
  - 输入设备码 → 校验 → 绑定

- [ ] **5.4** 新增已绑定设备列表
  - 展示设备码、主机名、OS、绑定方式、最后心跳、状态
  - 解绑操作

- [ ] **5.5** 新增激活审计日志页面
  - 时间、操作类型、设备码、操作者、详情

- [ ] **5.6** i18n 国际化
  - dictionary.ts 新增所有激活相关中英文标签

## 阶段六：Session Gateway 心跳处理

- [ ] **6.1** WebSocket 消息处理增加 `activation_heartbeat` 类型
  - 调用 DeviceBindingService.ProcessHeartbeat
  - 返回 ack 消息

## 任务依赖关系

```
阶段一 (1.1→1.2→1.3→1.4→1.5→1.6)
    ↓
阶段二 (2.1→2.2→2.3→2.4)  ←── 依赖阶段一
    ↓
阶段三 (3.1)               ←── 依赖阶段一（读取 activation_key）
    ↓
阶段四 (4.1→4.2→4.3→4.4→4.5) ←── 依赖阶段二（API 就绪）
    ↓
阶段五 (5.1→5.2→5.3→5.4→5.5→5.6) ←── 依赖阶段二（API 就绪）
    ↓
阶段六 (6.1)               ←── 依赖阶段一 + 阶段四
```

阶段四和阶段五可以**并行开发**。
