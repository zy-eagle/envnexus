---
id: dual-mode-activation
title: 双模式安装包激活系统（自动激活 + 设备码绑定）
status: confirmed
complexity: L
created_at: 2026-03-28
supersedes: v1 auto-activation, v2 device-code-only
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
Agent 采集硬件指纹 → 生成设备码
    ↓
Agent 调用 POST /agent/v1/activate
  { activation_key, device_code, hardware_hash, device_info }
    ↓
服务端验证: activation_key 有效 + 未超限额 + 设备未绑定
    ↓
自动创建绑定记录 → Agent 进入正常工作模式
```

### 模式 B：设备码绑定

```
管理员创建安装包 → 选择"设备码绑定" → 设置最大设备数
    ↓
用户下载安装 → Agent 首次启动
    ↓
Agent 采集硬件指纹 → 生成设备码
    ↓
Agent 调用 POST /agent/v1/device-code 上报设备码
    ↓
界面显示: "您的设备码: ENX-A3F8-K9D2-M7X1，请联系管理员绑定"
    ↓
管理员在控制台输入设备码 → 绑定到安装包
    ↓
Agent 轮询激活状态 → 检测到已绑定 → 进入正常工作模式
```

## 3. 数据模型

### 3.1 修改 `download_packages` 表

```sql
ALTER TABLE download_packages
  ADD COLUMN activation_mode VARCHAR(16) NOT NULL DEFAULT 'auto'
    COMMENT 'auto=自动激活, manual=设备码绑定' AFTER sign_metadata_json,
  ADD COLUMN activation_key VARCHAR(64) NOT NULL DEFAULT ''
    COMMENT '激活码（仅 auto 模式使用）' AFTER activation_mode,
  ADD COLUMN max_devices INT NOT NULL DEFAULT 1
    COMMENT '最大可绑定设备数' AFTER activation_key,
  ADD COLUMN bound_count INT NOT NULL DEFAULT 0
    COMMENT '已绑定设备数' AFTER max_devices;
```

### 3.2 新增 `device_bindings` 表

```sql
CREATE TABLE device_bindings (
  id            VARCHAR(26) NOT NULL PRIMARY KEY,
  tenant_id     VARCHAR(26) NOT NULL,
  package_id    VARCHAR(26) NOT NULL,
  device_code   VARCHAR(20) NOT NULL COMMENT 'ENX-XXXX-XXXX-XXXX',
  hardware_hash VARCHAR(64) NOT NULL COMMENT 'SHA-256 硬件指纹',
  device_info   JSON        COMMENT '{ os, hostname, cpu_model, ... }',
  status        VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/revoked',
  bound_at      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  bound_by      VARCHAR(26) NOT NULL DEFAULT '' COMMENT '管理员ID（manual模式）或 system（auto模式）',
  revoked_at    TIMESTAMP   NULL,
  created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY uk_device_code (device_code),
  INDEX idx_package_status (package_id, status),
  INDEX idx_tenant (tenant_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (package_id) REFERENCES download_packages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3.3 新增 `pending_devices` 表

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

> `pending_devices` 存储已上报但尚未绑定的设备码，供管理员在绑定时校验设备码是否真实存在。

## 4. 设备码生成算法

```
硬件采集（跨平台）:
  Windows: WMI 查询 CPU ProcessorId, Win32_BaseBoard SerialNumber, 物理网卡 MAC
  Linux:   /sys/class/dmi/id/product_uuid, /sys/class/net/*/address, dmidecode
  macOS:   system_profiler, IOPlatformSerialNumber

hardware_hash = SHA-256(cpu_id + "|" + sorted_mac_addrs + "|" + board_serial)

device_code = "ENX-" + Base32Encode(hardware_hash[0:7.5bytes])[:12]
            → 格式: ENX-XXXX-XXXX-XXXX (每组4位 Base32 字符)
```

**特性**：同一台电脑始终生成相同设备码，不同电脑不同。

## 5. API 设计

### 5.1 Agent 端 API（无需 JWT）

| 方法 | 路径 | 说明 | 使用模式 |
|------|------|------|---------|
| POST | `/agent/v1/device-code` | 上报硬件指纹，获取设备码 | A + B |
| POST | `/agent/v1/activate` | 自动激活（提交 activation_key + device_code） | A |
| GET | `/agent/v1/activation-status?device_code=XXX` | 查询激活状态 | A + B |

**POST /agent/v1/device-code**
```json
// 请求
{ "hardware_hash": "sha256...", "device_info": { "os": "windows", "hostname": "PC-001", "cpu_model": "i7-13700" } }
// 响应
{ "device_code": "ENX-A3F8-K9D2-M7X1" }
```

**POST /agent/v1/activate**（仅 auto 模式）
```json
// 请求
{ "activation_key": "ENX-ACT-XXXXXXXX", "device_code": "ENX-A3F8-K9D2-M7X1", "hardware_hash": "sha256..." }
// 响应 (成功)
{ "activated": true, "package_id": "...", "tenant_id": "..." }
// 响应 (失败)
{ "activated": false, "error": "max_devices_reached" }
```

**GET /agent/v1/activation-status**
```json
// 响应
{ "activated": true, "package_id": "...", "tenant_id": "...", "activation_mode": "auto" }
// 或
{ "activated": false }
```

### 5.2 管理员 API（需 JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/tenants/:tid/download-packages/:pid/bind` | 绑定设备码到安装包 |
| DELETE | `/api/v1/tenants/:tid/download-packages/:pid/bindings/:bid` | 解绑设备 |
| GET | `/api/v1/tenants/:tid/download-packages/:pid/bindings` | 查看绑定列表 |
| PUT | `/api/v1/tenants/:tid/download-packages/:pid/max-devices` | 修改最大设备数 |

## 6. Agent 客户端行为

### 6.1 统一启动流程

```
Agent 启动
    ↓
采集硬件指纹 → 计算 hardware_hash → 生成 device_code
    ↓
调用 POST /agent/v1/device-code（上报，幂等）
    ↓
读取内嵌配置中的 activation_mode
    ↓
┌─ auto 模式 ──────────────────────────┐
│ 读取内嵌 activation_key              │
│ 调用 POST /agent/v1/activate         │
│ 成功 → 保存激活状态 → 正常启动        │
│ 失败 → 显示错误信息                   │
└──────────────────────────────────────┘
┌─ manual 模式 ────────────────────────┐
│ 显示设备码 + "请联系管理员绑定"       │
│ 每 30 秒轮询 activation-status       │
│ 已激活 → 保存激活状态 → 正常启动      │
│ 用户可点击 [重试] 立即查询            │
└──────────────────────────────────────┘
```

### 6.2 Agent Desktop 激活等待页

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

## 7. 控制台 UI

### 7.1 创建安装包增强

新增字段：
- **激活模式**：单选（自动激活 / 设备码绑定），默认"自动激活"
- **最大绑定设备数**：数字输入框，默认 1

### 7.2 安装包列表页增强

| 列 | 说明 |
|----|------|
| 激活模式 | 显示"自动"或"手动"标签 |
| 已绑定/上限 | 如 `3/10` |
| 激活码 | 仅 auto 模式显示，可复制（manual 模式显示 `—`） |
| 操作 | 增加"绑定设备"按钮（仅 manual 模式）和"查看设备"按钮 |

### 7.3 绑定设备弹窗（manual 模式）

管理员输入设备码 → 系统校验（格式 + 存在于 pending_devices + 未被其他包绑定 + 未超限额）→ 绑定成功

### 7.4 已绑定设备列表

| 设备码 | 主机名 | 操作系统 | 绑定方式 | 绑定时间 | 操作 |
|--------|--------|---------|---------|---------|------|
| ENX-A3F8-K9D2-M7X1 | PC-001 | Windows 11 | 自动 | 2026-03-28 | [解绑] |
| ENX-B7C1-P4E9-N2K5 | MAC-002 | macOS 15 | 手动 | 2026-03-28 | [解绑] |

## 8. 安装包构建增强

Package Build Worker 注入到二进制的配置增加字段：

```json
{
  "platform_url": "https://...",
  "ws_url": "wss://...",
  "activation_mode": "auto",
  "activation_key": "ENX-ACT-XXXXXXXX"
}
```

- `auto` 模式：注入 `activation_mode` + `activation_key`
- `manual` 模式：注入 `activation_mode`（无 activation_key）

## 9. 边界与异常处理

| 场景 | 处理方式 |
|------|---------|
| 同一台电脑重复请求设备码 | 幂等，返回相同设备码 |
| auto 模式同一台电脑重复激活 | 幂等，返回已激活 |
| 设备码已被其他安装包绑定 | 拒绝，提示"该设备已绑定到其他安装包" |
| 绑定数已达上限 | 拒绝，提示"已达最大绑定设备数" |
| 管理员解绑设备 | bound_count -= 1，设备下次启动需重新激活 |
| 管理员调大 max_devices | 立即生效 |
| 管理员调小 max_devices | 不能小于当前 bound_count |
| Agent 无网络（auto 模式） | 重试 3 次后显示错误 |
| Agent 无网络（manual 模式） | 设备码本地生成可显示，提示网络异常 |
| 硬件变更 | 设备码改变，需管理员解绑旧码 |
| 安装包状态非 ready | 不允许激活/绑定 |

## 10. 涉及的文件变更

### 后端 (platform-api)
| 文件 | 变更 |
|------|------|
| `migrations/000006_device_binding.up.sql` | 新增 device_bindings + pending_devices 表，download_packages 新字段 |
| `domain/device_binding.go` | 新增 DeviceBinding + PendingDevice 实体 |
| `domain/package.go` | 增加 ActivationMode, ActivationKey, MaxDevices, BoundCount |
| `dto/device_binding.go` | 新增绑定相关 DTO |
| `dto/package.go` | 增加激活相关请求/响应字段 |
| `repository/mysql_device_binding_repo.go` | 新增仓库 |
| `service/device_binding/device_binding_service.go` | 新增服务 |
| `handler/http/package_handler.go` | 增加绑定管理路由 |
| `handler/agent/activate_handler.go` | 新增 Agent 激活端点 |
| `cmd/platform-api/main.go` | 注册新服务和路由 |

### Job Runner
| 文件 | 变更 |
|------|------|
| `worker/package_build.go` | 注入 activation_mode + activation_key |

### Agent Core
| 文件 | 变更 |
|------|------|
| `internal/hwinfo/fingerprint.go` | 新增硬件指纹采集（跨平台） |
| `internal/activation/activation.go` | 新增激活模块 |
| `internal/config/config.go` | 解析 activation_mode, activation_key |
| `internal/bootstrap/bootstrap.go` | 启动时先检查激活 |

### Agent Desktop
| 文件 | 变更 |
|------|------|
| `src/renderer/` | 新增激活等待页面 |

### 控制台前端 (console-web)
| 文件 | 变更 |
|------|------|
| `download-packages/page.tsx` | 激活模式选择、绑定弹窗、设备列表 |
| `dictionary.ts` | 新增 i18n 标签 |
