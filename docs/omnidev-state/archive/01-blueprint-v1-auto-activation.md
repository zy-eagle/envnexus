---
id: activation-system
title: 安装包激活与设备绑定系统
status: draft
complexity: L
created_at: 2026-03-28
---

# 蓝图：安装包激活与设备绑定系统

## 1. 需求概述

当前安装包下载后可无限制安装使用。需要增加**激活机制**：
- 每个安装包生成时自动关联一个**激活码（License Key）**
- 安装包安装后需要激活才能正常使用
- 每个激活码有**最大可绑定设备数**限制（管理员可配置）
- 一台设备激活后绑定到该激活码，不可重复激活
- 管理员可在控制台查看和管理激活状态

## 2. 现状分析

### 已有基础设施
| 组件 | 现状 |
|------|------|
| `download_packages` 表 | 存在，无激活码字段 |
| `licenses` 表 | 存在但是**租户级**许可证（plan_code, max_devices），与安装包无关 |
| `enrollment_tokens` 表 | 用于设备注册，有 `max_uses` 字段，但与激活是不同概念 |
| `devices` 表 | 有 `status` 字段，注册时设为 `active` |
| Package Build Worker | 构建安装包时会注入配置到二进制尾部（`ENX_CONF_START:` + JSON） |
| Agent Bootstrap | 启动时读取注入的配置，用 enrollment token 注册设备 |

### 关键决策：复用 vs 新建

**方案 A**：复用 `enrollment_tokens` 的 `max_uses` 作为激活限制
- 优点：改动最小
- 缺点：enrollment token 是注册凭证，语义不同；无法追踪"哪台设备绑定了哪个激活码"

**方案 B（推荐）**：在 `download_packages` 上增加激活码字段 + 新建 `package_activations` 表记录绑定关系
- 优点：语义清晰，可追踪每台设备的激活状态，管理员可灵活配置
- 缺点：需要新增数据库表和 API

## 3. 最终状态描述（方案 B）

### 3.1 数据模型

```
download_packages（修改）
├── + activation_key VARCHAR(64)     -- 安装包激活码（唯一，自动生成）
├── + max_activations INT DEFAULT 1  -- 最大可激活设备数（管理员可配置）
└── + activated_count INT DEFAULT 0  -- 已激活设备数

package_activations（新增）
├── id                    -- 激活记录 ID
├── package_id            -- 关联的安装包
├── tenant_id             -- 租户
├── device_id             -- 绑定的设备
├── hardware_fingerprint  -- 硬件指纹（CPU ID + MAC + 主板序列号的哈希）
├── activated_at          -- 激活时间
├── status                -- active / revoked
└── created_at, updated_at
```

### 3.2 激活流程

```
用户下载安装包 → 安装到电脑 → Agent 首次启动
    ↓
Agent 采集硬件指纹（CPU + MAC + 主板）
    ↓
Agent 调用 POST /agent/v1/activate
  请求体: { activation_key, hardware_fingerprint, device_info }
    ↓
Platform API 验证:
  1. activation_key 是否存在且有效
  2. activated_count < max_activations
  3. hardware_fingerprint 是否已被激活过（防重复）
    ↓
验证通过 → 创建 package_activations 记录
         → activated_count += 1
         → 返回设备令牌（复用现有 enrollment 流程）
         → Agent 保存激活状态到本地
    ↓
验证失败 → 返回错误（已达上限 / 激活码无效 / 设备已激活）
         → Agent 显示激活失败提示
```

### 3.3 管理员控制台

**安装包列表页增强：**
- 显示激活码（可复制）
- 显示 `已激活/最大激活数`（如 `3/10`）
- 点击查看已激活设备列表

**创建安装包时：**
- 新增"最大激活设备数"输入框（默认 1）
- 激活码自动生成

**安装包详情/编辑：**
- 可修改最大激活设备数（只能增大，不能小于已激活数）
- 可吊销某台设备的激活（释放名额）

### 3.4 安装包构建增强

Package Build Worker 在构建安装包时，将 `activation_key` 注入到二进制配置中：
```json
{
  "platform_url": "https://...",
  "ws_url": "wss://...",
  "activation_key": "ENX-XXXX-XXXX-XXXX-XXXX"
}
```

### 3.5 Agent 客户端增强

Agent Core 启动时：
1. 检查本地是否已有激活记录（`~/.envnexus/activation.json`）
2. 如果未激活：采集硬件指纹 → 调用激活 API → 保存激活状态
3. 如果已激活：正常启动，进入 enrollment 流程
4. 激活失败：显示错误信息，Agent 不进入正常工作模式

## 4. 边界与异常处理

| 场景 | 处理方式 |
|------|---------|
| 同一台电脑重复激活同一个激活码 | 幂等处理，返回成功（已激活） |
| 硬件变更（如换网卡） | 指纹不匹配，需要管理员吊销旧激活后重新激活 |
| 激活码达到上限 | 返回 403，提示"激活码已达最大设备数" |
| 管理员增加 max_activations | 立即生效，新设备可继续激活 |
| 管理员吊销某设备激活 | activated_count -= 1，该设备下次启动需重新激活 |
| 安装包状态非 ready | 不允许激活 |

## 5. 涉及的文件变更

### 后端 (platform-api)
- `migrations/000006_package_activation.up.sql` — 新增表和字段
- `domain/package.go` — 增加激活码字段
- `domain/package_activation.go` — 新增激活记录实体
- `dto/package.go` — 增加激活相关字段
- `dto/package_activation.go` — 新增激活 DTO
- `repository/mysql_package_repo.go` — 增加激活码生成
- `repository/mysql_package_activation_repo.go` — 新增激活记录仓库
- `service/package/package_service.go` — 增加激活逻辑
- `handler/http/package_handler.go` — 增加激活管理 API
- `handler/agent/activate_handler.go` — 新增 Agent 激活端点

### Job Runner
- `worker/package_build.go` — 构建时注入 activation_key

### Agent Core
- `internal/config/config.go` — 解析 activation_key
- `internal/activation/` — 新增激活模块（采集指纹、调用 API、本地存储）
- `internal/bootstrap/bootstrap.go` — 启动时先检查激活

### 控制台前端 (console-web)
- `download-packages/page.tsx` — 增加激活码显示、max_activations 配置
- `dictionary.ts` — 新增 i18n 标签

## 6. 安全考虑

- 激活码格式：`ENX-XXXX-XXXX-XXXX-XXXX`（Base32 编码，易于人工输入）
- 硬件指纹：SHA-256(CPU_ID + MAC_ADDRESSES + BOARD_SERIAL)，不可逆
- 激活 API 无需 JWT（Agent 未注册前无令牌），但需要 rate limiting 防暴力破解
- activation_key 在数据库中存储哈希值，API 返回明文仅在创建时一次
