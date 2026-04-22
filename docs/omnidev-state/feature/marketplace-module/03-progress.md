---
current_phase: 3
active_group: 1
---

## State Snapshot

### Group 1 (进行中)
- [ ] **T1** [backend] 创建市场模块和设备授权的数据库模型 (`MarketplaceItem`, `TenantSubscription`, `DeviceAuthCode`, `IdeClientToken`)
- [ ] **T2** [backend] 创建市场模块和设备授权的 Repository 层

### 阻塞与问题 (Blockers & Issues)
- 无

### 下一步 (Next Action)
- 实现 T1：在 `domain` 目录下创建 `marketplace.go` 和 `device_auth.go`，定义 GORM 模型。