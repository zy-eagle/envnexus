---
id: dual-mode-activation
title: 双模式安装包激活系统（自动激活 + 设备码绑定）
status: confirmed
complexity: L
version: 4
created_at: 2026-03-28
supersedes: v1 auto-activation, v2 device-code-only, v3 dual-mode-basic
improvements: 硬件指纹容错、激活心跳、激活码安全存储
---

# 蓝图：双模式安装包激活系统

## 1. 需求概述

安装包支持两种激活模式，管理员在创建安装包时选择：

| 模式 | 名称 | 行为 | 适用场景 |
|------|------|------|---------|
| **A** | 自动激活 | Agent 安装后自动激活，激活码内嵌在安装包中 | 快速部署、信任内网 |
| **B** | 设备码绑定 | Agent 显示设备码，管理员在控制台手动绑定 | 严格管控、资产审计 |

两种模式共享同一套数据模型，区别仅在于**谁触发绑定动作**。

## 2. 完整用户流程

### 模式 A：自动激活

```
管理员创建安装包 → 选择"自动激活" → 设置最大设备数
    ↓
用户下载安装 → Agent 首次启动
    ↓
Agent 采集硬件组件 → 生成设备码 + 组件指纹列表
    ↓
Agent 调用 POST /agent/v1/activate
  { activation_key, device_code, components[], device_info }
    ↓
服务端验证: activation_key 哈希匹配 + 未超限额 + 设备未绑定
    ↓
自动创建绑定记录 → Agent 进入正常工作模式
    ↓
Agent 定期心跳校验激活状态（复用 WebSocket）
```

### 模式 B：设备码绑定

```
管理员创建安装包 → 选择"设备码绑定" → 设置最大设备数
    ↓
用户下载安装 → Agent 首次启动
    ↓
Agent 采集硬件组件 → 生成设备码 + 组件指纹列表
    ↓
Agent 调用 POST /agent/v1/device-code 上报设备码
    ↓
界面显示: "您的设备码: ENX-A3F8-K9D2-M7X1，请联系管理员绑定"
    ↓
管理员在控制台输入设备码 → 绑定到安装包
    ↓
Agent 轮询激活状态 → 检测到已绑定 → 进入正常工作模式
    ↓
Agent 定期心跳校验激活状态（复用 WebSocket）
```

## 3. 数据模型

### 3.1 修改 `download_packages` 表

```sql
ALTER TABLE download_packages
  ADD COLUMN activation_mode VARCHAR(16) NOT NULL DEFAULT 'auto'
    COMMENT 'auto=自动激活, manual=设备码绑定' AFTER sign_metadata_json,
  ADD COLUMN activation_key_hash VARCHAR(64) NOT NULL DEFAULT ''
    COMMENT '激活码的 SHA-256 哈希（仅 auto 模式）' AFTER activation_mode,
  ADD COLUMN max_devices INT NOT NULL DEFAULT 1
    COMMENT '最大可绑定设备数' AFTER activation_key_hash,
  ADD COLUMN bound_count INT NOT NULL DEFAULT 0
    COMMENT '已绑定设备数' AFTER max_devices;
```

> **安全改进**：数据库只存储 `activation_key` 的 SHA-256 哈希，明文仅在创建时返回一次。

### 3.2 新增 `device_bindings` 表

```sql
CREATE TABLE device_bindings (
  id            VARCHAR(26) NOT NULL PRIMARY KEY,
  tenant_id     VARCHAR(26) NOT NULL,
  package_id    VARCHAR(26) NOT NULL,
  device_code   VARCHAR(20) NOT NULL COMMENT 'ENX-XXXX-XXXX-XXXX',
  hardware_hash VARCHAR(64) NOT NULL COMMENT 'SHA-256 主指纹（向后兼容）',
  device_info   JSON        COMMENT '{ os, hostname, cpu_model, ... }',
  status        VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/revoked',
  bound_at      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  bound_by      VARCHAR(26) NOT NULL DEFAULT '' COMMENT '管理员ID 或 system',
  revoked_at    TIMESTAMP   NULL,
  last_heartbeat TIMESTAMP  NULL COMMENT '最后一次心跳时间',
  created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY uk_device_code (device_code),
  INDEX idx_package_status (package_id, status),
  INDEX idx_tenant (tenant_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (package_id) REFERENCES download_packages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3.3 新增 `device_components` 表（硬件指纹容错）

```sql
CREATE TABLE device_components (
  id            VARCHAR(26) NOT NULL PRIMARY KEY,
  device_code   VARCHAR(20) NOT NULL,
  component_type VARCHAR(16) NOT NULL COMMENT 'cpu/board/mac/disk/gpu',
  component_hash VARCHAR(64) NOT NULL COMMENT '单组件 SHA-256',
  created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,

  INDEX idx_device (device_code),
  INDEX idx_component (component_type, component_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

> **容错机制**：每个硬件组件独立存储哈希，校验时按比例匹配（如 5 个中 3 个匹配即通过），换网卡或升级显卡不会导致激活失效。

### 3.4 新增 `pending_devices` 表

```sql
CREATE TABLE pending_devices (
  id            VARCHAR(26) NOT NULL PRIMARY KEY,
  device_code   VARCHAR(20) NOT NULL,
  hardware_hash VARCHAR(64) NOT NULL,
  device_info   JSON,
  platform_url  VARCHAR(255) NOT NULL DEFAULT '',
  created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY uk_device_code (device_code),
  INDEX idx_hardware (hardware_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3.5 新增 `activation_audit_logs` 表（审计日志）

```sql
CREATE TABLE activation_audit_logs (
  id          VARCHAR(26) NOT NULL PRIMARY KEY,
  tenant_id   VARCHAR(26) NOT NULL,
  package_id  VARCHAR(26) NOT NULL,
  device_code VARCHAR(20) NOT NULL DEFAULT '',
  action      VARCHAR(32) NOT NULL COMMENT 'activate/bind/unbind/revoke/heartbeat_fail',
  actor       VARCHAR(26) NOT NULL DEFAULT '' COMMENT '操作者（管理员ID 或 system）',
  detail      JSON        COMMENT '附加信息',
  created_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,

  INDEX idx_tenant_time (tenant_id, created_at),
  INDEX idx_package (package_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 4. 硬件指纹与设备码

### 4.1 采集的硬件组件（5 项）

| 组件 | 标识 | Windows 采集方式 | Linux 采集方式 | macOS 采集方式 |
|------|------|-----------------|---------------|---------------|
| CPU | `cpu` | WMI `Win32_Processor.ProcessorId` | `/proc/cpuinfo` | `sysctl machdep.cpu` |
| 主板 | `board` | WMI `Win32_BaseBoard.SerialNumber` | `/sys/class/dmi/id/board_serial` | `IOPlatformSerialNumber` |
| 网卡 MAC | `mac` | WMI `Win32_NetworkAdapter` (物理) | `/sys/class/net/*/address` | `networksetup` |
| 硬盘 | `disk` | WMI `Win32_DiskDrive.SerialNumber` | `/dev/disk/by-id/` | `diskutil info` |
| 显卡 | `gpu` | WMI `Win32_VideoController.PNPDeviceID` | `lspci` | `system_profiler SPDisplaysDataType` |

### 4.2 设备码生成

```
对每个组件分别计算哈希:
  cpu_hash   = SHA-256("cpu:" + cpu_id)
  board_hash = SHA-256("board:" + board_serial)
  mac_hash   = SHA-256("mac:" + sorted_physical_macs)
  disk_hash  = SHA-256("disk:" + primary_disk_serial)
  gpu_hash   = SHA-256("gpu:" + gpu_device_id)

主指纹（向后兼容）:
  hardware_hash = SHA-256(cpu_hash + board_hash + mac_hash + disk_hash + gpu_hash)

设备码:
  device_code = "ENX-" + Base32(hardware_hash[0:7.5bytes])[:12]
              → 格式: ENX-XXXX-XXXX-XXXX
```

### 4.3 指纹容错匹配

当设备再次上报时，服务端按以下策略匹配：

```
已存储组件: [cpu_hash, board_hash, mac_hash, disk_hash, gpu_hash]
新上报组件: [cpu_hash, board_hash, mac_hash', disk_hash, gpu_hash]
                                       ↑ 换了网卡

匹配结果: 5 个中有 4 个匹配
阈值配置: match_threshold = 3（默认，可在系统设置中调整）
判定: 4 >= 3 → 视为同一台设备 ✓

→ 更新 mac_hash 为新值（漂移更新）
→ 重新计算 device_code（如果变化则更新 pending_devices）
```

**容错规则**：
- 默认阈值 3/5（5 个组件中至少 3 个匹配）
- CPU 和主板为**核心组件**，至少一个必须匹配
- 匹配通过后自动更新变化的组件哈希（漂移更新）
- 管理员可在系统设置中调整阈值

## 5. 激活码安全

### 5.1 存储策略

```
创建安装包时:
  1. 生成随机激活码明文: activation_key = "ENX-ACT-" + random_base32(16)
  2. 计算哈希: activation_key_hash = SHA-256(activation_key)
  3. 数据库只存储 activation_key_hash
  4. API 响应中返回明文 activation_key（仅此一次）
  5. 控制台显示明文并提示"请立即保存，此激活码不会再次显示"

Agent 激活时:
  1. Agent 提交 activation_key 明文
  2. 服务端计算 SHA-256(提交的 key)
  3. 与数据库中的 activation_key_hash 比对
  4. 匹配 → 激活成功
```

### 5.2 激活码注入安装包

Package Build Worker 构建时将**明文** activation_key 注入二进制：

```json
{
  "platform_url": "https://...",
  "ws_url": "wss://...",
  "activation_mode": "auto",
  "activation_key": "ENX-ACT-XXXXXXXXXXXXXXXX"
}
```

> 明文注入是必要的，因为 Agent 需要用明文向服务端验证。安装包本身应通过签名和分发渠道保护。

## 6. 激活心跳机制

### 6.1 设计

```
Agent 正常运行中（已激活）
    ↓
每 1 小时通过 WebSocket 发送心跳（复用现有连接）
  消息: { type: "activation_heartbeat", device_code: "ENX-...", hardware_hash: "..." }
    ↓
服务端处理:
  1. 更新 device_bindings.last_heartbeat = NOW()
  2. 检查 binding.status 是否仍为 active
  3. 检查硬件指纹是否仍然匹配（容错匹配）
    ↓
┌─ 正常 ─────────────────────────────┐
│ 返回 { status: "ok" }             │
│ Agent 继续正常工作                  │
└────────────────────────────────────┘
┌─ 已被解绑 ─────────────────────────┐
│ 返回 { status: "revoked" }        │
│ Agent 回到"等待激活"状态            │
│ 清除本地 activation.json           │
│ 记录审计日志 heartbeat_fail        │
└────────────────────────────────────┘
┌─ 硬件不匹配 ───────────────────────┐
│ 容错匹配通过 → 漂移更新组件        │
│ 容错匹配失败 → 返回 revoked        │
└────────────────────────────────────┘
```

### 6.2 心跳参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| 心跳间隔 | 1 小时 | Agent 发送间隔 |
| 宽限期 | 24 小时 | 心跳中断后 Agent 仍可工作的时间 |
| 离线容忍 | 72 小时 | 超过此时间未心跳，服务端标记为 `offline` |

### 6.3 断网处理

```
Agent 发送心跳失败（网络异常）
    ↓
本地记录上次成功心跳时间
    ↓
距上次成功心跳 < 24 小时 → 正常工作（宽限期内）
距上次成功心跳 >= 24 小时 → 显示警告"网络连接中断，请尽快恢复"
距上次成功心跳 >= 72 小时 → Agent 进入受限模式（仅只读诊断）
```

## 7. API 设计

### 7.1 Agent 端 API（无需 JWT）

| 方法 | 路径 | 说明 | 使用模式 |
|------|------|------|---------|
| POST | `/agent/v1/device-code` | 上报硬件组件，获取设备码 | A + B |
| POST | `/agent/v1/activate` | 自动激活 | A |
| GET | `/agent/v1/activation-status?device_code=XXX` | 查询激活状态 | A + B |

**POST /agent/v1/device-code**
```json
// 请求
{
  "components": [
    { "type": "cpu",   "hash": "sha256..." },
    { "type": "board", "hash": "sha256..." },
    { "type": "mac",   "hash": "sha256..." },
    { "type": "disk",  "hash": "sha256..." },
    { "type": "gpu",   "hash": "sha256..." }
  ],
  "device_info": { "os": "windows", "hostname": "PC-001", "cpu_model": "i7-13700" }
}
// 响应
{ "device_code": "ENX-A3F8-K9D2-M7X1" }
```

**POST /agent/v1/activate**（仅 auto 模式）
```json
// 请求
{
  "activation_key": "ENX-ACT-XXXXXXXXXXXXXXXX",
  "device_code": "ENX-A3F8-K9D2-M7X1",
  "components": [ ... ]
}
// 响应 (成功)
{ "activated": true, "package_id": "...", "tenant_id": "..." }
// 响应 (失败)
{ "activated": false, "error": "max_devices_reached" | "invalid_key" | "already_bound" }
```

**GET /agent/v1/activation-status**
```json
{ "activated": true, "package_id": "...", "tenant_id": "...", "activation_mode": "auto" }
```

**WebSocket 心跳**（复用现有连接）
```json
// Agent → Server
{ "type": "activation_heartbeat", "device_code": "ENX-...", "components": [ ... ] }
// Server → Agent
{ "type": "activation_heartbeat_ack", "status": "ok" | "revoked" }
```

### 7.2 管理员 API（需 JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/tenants/:tid/download-packages/:pid/bind` | 绑定设备码 |
| DELETE | `/api/v1/tenants/:tid/download-packages/:pid/bindings/:bid` | 解绑设备 |
| GET | `/api/v1/tenants/:tid/download-packages/:pid/bindings` | 查看绑定列表 |
| PUT | `/api/v1/tenants/:tid/download-packages/:pid/max-devices` | 修改最大设备数 |
| GET | `/api/v1/tenants/:tid/activation-logs` | 查看激活审计日志 |

## 8. Agent 客户端行为

### 8.1 统一启动流程

```
Agent 启动
    ↓
采集 5 项硬件组件 → 计算各组件哈希 + 主指纹
    ↓
检查本地 ~/.envnexus/activation.json
    ↓
┌─ 已有本地激活记录 ───────────────────┐
│ 重新计算硬件指纹                     │
│ 与保存的指纹做容错匹配（本地 3/5）   │
│ 匹配 → 正常启动 + 启动心跳          │
│ 不匹配 → 清除记录 → 走首次激活流程   │
└──────────────────────────────────────┘
┌─ 无本地激活记录（首次） ─────────────┐
│ 调用 POST /agent/v1/device-code     │
│ 读取内嵌 activation_mode            │
│                                     │
│ auto → 调用 activate API → 成功则   │
│        保存 activation.json         │
│                                     │
│ manual → 显示设备码等待页           │
│          轮询 activation-status     │
│          成功则保存 activation.json  │
└──────────────────────────────────────┘
    ↓
正常工作模式 → 每 1 小时发送心跳
    ↓
心跳返回 revoked → 清除本地激活 → 回到等待激活
```

### 8.2 本地激活文件 `~/.envnexus/activation.json`

```json
{
  "device_code": "ENX-A3F8-K9D2-M7X1",
  "package_id": "01HYZ...",
  "tenant_id": "01HYX...",
  "activation_mode": "auto",
  "activated_at": "2026-03-28T10:00:00Z",
  "last_heartbeat": "2026-03-28T15:00:00Z",
  "components": {
    "cpu": "sha256...",
    "board": "sha256...",
    "mac": "sha256...",
    "disk": "sha256...",
    "gpu": "sha256..."
  }
}
```

### 8.3 Agent Desktop 激活等待页

```
┌──────────────────────────────────────────┐
│            等待激活                        │
│                                          │
│  您的设备码:                              │
│  ┌────────────────────────────┐          │
│  │  ENX-A3F8-K9D2-M7X1       │  [复制]  │
│  └────────────────────────────┘          │
│                                          │
│  请将此设备码提供给管理员，               │
│  在控制台完成绑定后即可使用。             │
│                                          │
│  正在等待激活...  ●                       │
│                                          │
│              [重试激活]                    │
└──────────────────────────────────────────┘
```

## 9. 控制台 UI

### 9.1 创建安装包增强

新增字段：
- **激活模式**：单选（自动激活 / 设备码绑定），默认"自动激活"
- **最大绑定设备数**：数字输入框，默认 1

创建成功后（auto 模式）：
- 弹窗显示激活码明文 + 警告"请立即保存，此激活码不会再次显示"
- 提供复制按钮

### 9.2 安装包列表页增强

| 列 | 说明 |
|----|------|
| 激活模式 | 标签：`自动` (蓝色) / `手动` (橙色) |
| 已绑定/上限 | 如 `3/10`，进度条样式 |
| 操作 | "绑定设备"（仅 manual）、"查看设备"、"审计日志" |

### 9.3 绑定设备弹窗（manual 模式）

管理员输入设备码 → 系统校验 → 绑定成功

### 9.4 已绑定设备列表

| 设备码 | 主机名 | 操作系统 | 绑定方式 | 最后心跳 | 状态 | 操作 |
|--------|--------|---------|---------|---------|------|------|
| ENX-A3F8-... | PC-001 | Win 11 | 自动 | 5 分钟前 | 在线 | [解绑] |
| ENX-B7C1-... | MAC-002 | macOS 15 | 手动 | 2 小时前 | 在线 | [解绑] |
| ENX-C9D3-... | PC-003 | Win 10 | 自动 | 3 天前 | 离线 | [解绑] |

### 9.5 激活审计日志

| 时间 | 操作 | 设备码 | 操作者 | 详情 |
|------|------|--------|--------|------|
| 03-28 10:00 | 自动激活 | ENX-A3F8-... | system | 安装包 xxx |
| 03-28 11:30 | 手动绑定 | ENX-B7C1-... | admin@co.com | 安装包 xxx |
| 03-28 14:00 | 解绑设备 | ENX-C9D3-... | admin@co.com | 原因：设备报废 |

## 10. 安装包构建增强

Package Build Worker 注入配置：

```json
// auto 模式
{
  "platform_url": "https://...",
  "ws_url": "wss://...",
  "activation_mode": "auto",
  "activation_key": "ENX-ACT-XXXXXXXXXXXXXXXX"
}

// manual 模式
{
  "platform_url": "https://...",
  "ws_url": "wss://...",
  "activation_mode": "manual"
}
```

## 11. Rate Limiting（防暴力破解）

| API | 限流策略 |
|-----|---------|
| POST /agent/v1/device-code | 同一 IP 每分钟 10 次 |
| POST /agent/v1/activate | 同一 IP 每分钟 5 次；同一 activation_key 每分钟 10 次 |
| GET /agent/v1/activation-status | 同一 device_code 每分钟 6 次 |

失败响应采用指数退避提示（1s, 2s, 4s, 8s...）。

## 12. 边界与异常处理

| 场景 | 处理方式 |
|------|---------|
| 同一台电脑重复请求设备码 | 幂等，返回相同设备码 |
| auto 模式同一台电脑重复激活 | 幂等，返回已激活 |
| 设备码已被其他安装包绑定 | 拒绝，提示"该设备已绑定到其他安装包" |
| 绑定数已达上限 | 拒绝，提示"已达最大绑定设备数" |
| 管理员解绑设备 | bound_count -= 1，下次心跳通知 Agent 失效 |
| 管理员调大 max_devices | 立即生效 |
| 管理员调小 max_devices | 不能小于当前 bound_count |
| Agent 无网络（auto 模式） | 重试 3 次后显示错误 |
| Agent 无网络（manual 模式） | 设备码本地生成可显示，提示网络异常 |
| 硬件小幅变更（换网卡） | 容错匹配通过 → 漂移更新组件哈希 |
| 硬件大幅变更（换主板+CPU） | 容错匹配失败 → 需管理员解绑旧码 |
| 安装包状态非 ready | 不允许激活/绑定 |
| 心跳超过 24h 未收到 | Agent 显示警告 |
| 心跳超过 72h 未收到 | Agent 进入受限模式（仅只读） |
| activation_key 泄露 | 管理员可在控制台重新生成（旧 key 失效） |

## 13. 涉及的文件变更

### 后端 (platform-api)
| 文件 | 变更 |
|------|------|
| `migrations/000006_device_binding.up.sql` | 新增 4 张表 + download_packages 新字段 |
| `domain/device_binding.go` | DeviceBinding, PendingDevice, DeviceComponent, ActivationAuditLog |
| `domain/package.go` | 增加 ActivationMode, ActivationKeyHash, MaxDevices, BoundCount |
| `dto/device_binding.go` | 绑定/激活/审计相关 DTO |
| `dto/package.go` | 增加激活相关请求/响应字段 |
| `repository/mysql_device_binding_repo.go` | 新增仓库（含组件容错匹配逻辑） |
| `service/device_binding/device_binding_service.go` | 新增服务 |
| `handler/http/package_handler.go` | 增加绑定管理 + 审计日志路由 |
| `handler/agent/activate_handler.go` | 新增 Agent 激活端点 |
| `cmd/platform-api/main.go` | 注册新服务和路由 |

### Session Gateway（心跳）
| 文件 | 变更 |
|------|------|
| WebSocket 消息处理 | 增加 `activation_heartbeat` 消息类型 |

### Job Runner
| 文件 | 变更 |
|------|------|
| `worker/package_build.go` | 注入 activation_mode + activation_key |

### Agent Core
| 文件 | 变更 |
|------|------|
| `internal/hwinfo/fingerprint.go` | 硬件指纹采集（5 组件，跨平台） |
| `internal/activation/activation.go` | 激活模块（设备码、激活、心跳、本地存储） |
| `internal/config/config.go` | 解析 activation_mode, activation_key |
| `internal/bootstrap/bootstrap.go` | 启动时先检查激活 |

### Agent Desktop
| 文件 | 变更 |
|------|------|
| `src/renderer/` | 激活等待页面 |

### 控制台前端 (console-web)
| 文件 | 变更 |
|------|------|
| `download-packages/page.tsx` | 激活模式、绑定弹窗、设备列表、审计日志 |
| `dictionary.ts` | 新增 i18n 标签 |
