---
id: device-code-activation
title: 安装包设备码绑定激活系统
status: draft
complexity: L
created_at: 2026-03-28
supersedes: activation-system (v1 auto-activation)
---

# 蓝图：安装包设备码绑定激活系统

## 1. 需求概述

安装包下载安装后，Agent 在本地生成一个**设备码（Device Code）**，用户将设备码提供给管理员，管理员在控制台将设备码绑定到对应的安装包。绑定后 Agent 才能正常工作。

**核心原则**：
- 设备码由硬件指纹生成，**唯一确定一台电脑**
- 管理员在控制台**主动绑定**设备码（非自动激活）
- 每个安装包有**最大可绑定设备数**（管理员可配置）
- 类似 Cursor 的激活体验

## 2. 完整用户流程

```
┌─────────────────────────────────────────────────────────────┐
│                     管理员操作（控制台）                       │
├─────────────────────────────────────────────────────────────┤
│ 1. 创建安装包 → 设置最大绑定设备数（如 10）                    │
│ 2. 安装包构建完成 → 下载链接可用                              │
│ 3. 将下载链接发给用户                                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                     用户操作（本地电脑）                       │
├─────────────────────────────────────────────────────────────┤
│ 4. 下载并安装 Agent                                          │
│ 5. 首次启动 → 屏幕显示设备码（如 ENX-A3F8-K9D2-M7X1）        │
│ 6. 将设备码告知管理员（截图/复制/口述）                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   管理员操作（控制台绑定）                     │
├─────────────────────────────────────────────────────────────┤
│ 7. 进入安装包详情 → 点击"绑定设备"                            │
│ 8. 输入用户提供的设备码 → 确认绑定                            │
│ 9. 系统校验：设备码格式 + 未被其他包绑定 + 未超限额             │
│ 10. 绑定成功 → 已绑定数 +1                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                     用户侧自动生效                            │
├─────────────────────────────────────────────────────────────┤
│ 11. Agent 定期轮询激活状态（或用户点击"重试激活"）             │
│ 12. 服务端返回"已激活" → Agent 进入正常工作模式               │
│ 13. 本地保存激活状态，后续启动不再轮询                        │
└─────────────────────────────────────────────────────────────┘
```

## 3. 数据模型

### 3.1 修改 `download_packages` 表

```sql
ALTER TABLE download_packages
  ADD COLUMN max_devices INT NOT NULL DEFAULT 1 AFTER sign_metadata_json,
  ADD COLUMN bound_count INT NOT NULL DEFAULT 0 AFTER max_devices;
```

### 3.2 新增 `device_bindings` 表

```sql
CREATE TABLE device_bindings (
  id            VARCHAR(26) NOT NULL PRIMARY KEY,
  tenant_id     VARCHAR(26) NOT NULL,
  package_id    VARCHAR(26) NOT NULL,
  device_code   VARCHAR(20) NOT NULL,
  hardware_hash VARCHAR(64) NOT NULL,
  device_info   JSON,
  status        VARCHAR(16) NOT NULL DEFAULT 'active',
  bound_at      TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  bound_by      VARCHAR(26) NOT NULL DEFAULT '',
  revoked_at    TIMESTAMP   NULL,
  created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY uk_device_code (device_code),
  INDEX idx_package (package_id, status),
  INDEX idx_tenant (tenant_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  FOREIGN KEY (package_id) REFERENCES download_packages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3.3 字段说明

| 字段 | 说明 |
|------|------|
| `device_code` | 设备码，格式 `ENX-XXXX-XXXX-XXXX`（12 位 Base32，全局唯一） |
| `hardware_hash` | SHA-256(CPU_ID + MAC地址列表 + 主板序列号)，不可逆 |
| `device_info` | JSON，包含 OS、hostname、CPU 型号等辅助信息（仅展示用） |
| `status` | `active`（已绑定）/ `revoked`（已解绑） |
| `bound_by` | 执行绑定操作的管理员 ID |
| `max_devices` | 安装包最大可绑定设备数 |
| `bound_count` | 当前已绑定设备数（含 active 状态的） |

## 4. 设备码生成算法

```
硬件采集:
  cpu_id     = WMI/dmidecode 获取 CPU ProcessorId
  mac_addrs  = 所有物理网卡 MAC 地址排序后拼接
  board_sn   = 主板序列号

hardware_hash = SHA-256(cpu_id + "|" + mac_addrs + "|" + board_sn)

device_code = "ENX-" + Base32(hardware_hash[0:7.5])
            = "ENX-XXXX-XXXX-XXXX"  (12位 Base32，分3组，每组4位)
```

**特性**：
- 同一台电脑无论安装多少次，生成的设备码始终相同
- 不同电脑的设备码不同（硬件指纹不同）
- 设备码短小易读，方便口述和手动输入

## 5. API 设计

### 5.1 Agent 端 API（无需认证）

```
POST /agent/v1/device-code
  请求: { hardware_hash: "sha256...", device_info: { os, hostname, cpu_model } }
  响应: { device_code: "ENX-A3F8-K9D2-M7X1" }
  说明: Agent 启动时上报硬件指纹，服务端生成并返回设备码（幂等）

GET /agent/v1/activation-status?device_code=ENX-A3F8-K9D2-M7X1
  响应: { activated: true/false, package_id: "...", tenant_id: "..." }
  说明: Agent 轮询自己的激活状态
```

### 5.2 管理员 API（需认证）

```
POST /api/v1/tenants/:tenantId/download-packages/:packageId/bind
  请求: { device_code: "ENX-A3F8-K9D2-M7X1" }
  响应: { binding_id: "...", device_code: "...", status: "active" }
  说明: 管理员将设备码绑定到安装包

DELETE /api/v1/tenants/:tenantId/download-packages/:packageId/bindings/:bindingId
  说明: 管理员解绑设备（释放名额）

GET /api/v1/tenants/:tenantId/download-packages/:packageId/bindings
  响应: { items: [{ device_code, device_info, status, bound_at }] }
  说明: 查看安装包的所有绑定设备

PUT /api/v1/tenants/:tenantId/download-packages/:packageId/max-devices
  请求: { max_devices: 20 }
  说明: 修改最大可绑定设备数
```

## 6. Agent 客户端行为

### 6.1 首次启动（未激活）

```
1. 采集硬件信息 → 计算 hardware_hash
2. 调用 POST /agent/v1/device-code 获取设备码
3. 在 Agent Desktop 界面显示:
   ┌──────────────────────────────────────┐
   │        等待激活                       │
   │                                      │
   │  您的设备码:                          │
   │  ┌──────────────────────────┐        │
   │  │  ENX-A3F8-K9D2-M7X1     │ [复制]  │
   │  └──────────────────────────┘        │
   │                                      │
   │  请将此设备码提供给管理员进行绑定      │
   │                                      │
   │           [重试激活]                  │
   └──────────────────────────────────────┘
4. 每 30 秒自动轮询 GET /agent/v1/activation-status
5. 激活成功 → 保存到 ~/.envnexus/activation.json → 进入正常模式
```

### 6.2 后续启动（已激活）

```
1. 读取 ~/.envnexus/activation.json
2. 重新计算 hardware_hash，与保存的对比
3. 匹配 → 正常启动
4. 不匹配（硬件变更） → 回到"等待激活"状态
```

## 7. 控制台 UI

### 7.1 安装包列表页增强

表格新增列：
- **已绑定/上限**：显示 `3/10` 格式
- **操作**：增加"绑定设备"按钮

### 7.2 绑定设备弹窗

```
┌──────────────────────────────────────┐
│  绑定设备到安装包                     │
│                                      │
│  安装包: envnexus-agent-win-amd64    │
│  已绑定: 3/10                        │
│                                      │
│  设备码:                             │
│  ┌──────────────────────────┐        │
│  │  ENX-                    │        │
│  └──────────────────────────┘        │
│                                      │
│         [取消]  [确认绑定]            │
└──────────────────────────────────────┘
```

### 7.3 已绑定设备列表

点击安装包的"已绑定数"可展开查看：
| 设备码 | 主机名 | 操作系统 | 绑定时间 | 操作 |
|--------|--------|---------|---------|------|
| ENX-A3F8-K9D2-M7X1 | DESKTOP-001 | Windows 11 | 2026-03-28 | [解绑] |

### 7.4 创建安装包增强

新增字段：**最大绑定设备数**（数字输入框，默认 1）

## 8. 边界与异常处理

| 场景 | 处理方式 |
|------|---------|
| 同一台电脑多次请求设备码 | 幂等，返回相同的设备码 |
| 设备码已被其他安装包绑定 | 拒绝，提示"该设备已绑定到其他安装包" |
| 绑定数已达上限 | 拒绝，提示"已达最大绑定设备数" |
| 管理员解绑设备 | bound_count -= 1，设备下次启动回到等待激活 |
| 管理员调大 max_devices | 立即生效 |
| 管理员调小 max_devices | 不能小于当前 bound_count |
| Agent 无网络 | 显示设备码（本地生成），提示离线状态 |
| 硬件变更（换主板/网卡） | 设备码改变，需管理员解绑旧码、绑定新码 |

## 9. 涉及的文件变更

### 后端 (platform-api)
| 文件 | 变更 |
|------|------|
| `migrations/000006_device_binding.up.sql` | 新增 device_bindings 表 + download_packages 新字段 |
| `domain/device_binding.go` | 新增实体 |
| `domain/package.go` | 增加 max_devices, bound_count 字段 |
| `dto/device_binding.go` | 新增 DTO |
| `dto/package.go` | 增加 max_devices, bound_count 响应字段 |
| `repository/mysql_device_binding_repo.go` | 新增仓库 |
| `service/device_binding/` | 新增服务（绑定、解绑、查询、激活状态） |
| `handler/http/package_handler.go` | 增加绑定管理路由 |
| `handler/agent/activate_handler.go` | 新增设备码上报 + 激活状态查询 |
| `cmd/platform-api/main.go` | 注册新服务和路由 |

### Agent Core
| 文件 | 变更 |
|------|------|
| `internal/hwinfo/fingerprint.go` | 新增硬件指纹采集 |
| `internal/activation/activation.go` | 新增激活模块（设备码获取、状态轮询、本地存储） |
| `internal/bootstrap/bootstrap.go` | 启动时先检查激活状态 |

### Agent Desktop
| 文件 | 变更 |
|------|------|
| `src/renderer/` | 新增激活等待页面（显示设备码 + 重试按钮） |

### 控制台前端 (console-web)
| 文件 | 变更 |
|------|------|
| `download-packages/page.tsx` | 增加绑定列、绑定弹窗、设备列表 |
| `dictionary.ts` | 新增 i18n 标签 |

## 10. 安全考虑

- 设备码上报 API 需要 rate limiting（防枚举）
- hardware_hash 不可逆（SHA-256），即使数据库泄露也无法反推硬件信息
- device_info 仅包含非敏感信息（OS 版本、主机名、CPU 型号）
- 设备码全局唯一，一台电脑只能绑定到一个安装包
