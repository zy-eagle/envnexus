-- Device Groups (M5: Batch Operations)
CREATE TABLE IF NOT EXISTS device_groups (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    filter_json TEXT NULL,
    created_by CHAR(26) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_dg_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Device Group Members
CREATE TABLE IF NOT EXISTS device_group_members (
    id CHAR(26) PRIMARY KEY,
    device_group_id CHAR(26) NOT NULL,
    device_id CHAR(26) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_dgm_group_device (device_group_id, device_id),
    KEY idx_dgm_group (device_group_id),
    KEY idx_dgm_device (device_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Batch Tasks
CREATE TABLE IF NOT EXISTS batch_tasks (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    device_group_id CHAR(26) NOT NULL,
    command_task_id CHAR(26) NOT NULL,
    strategy VARCHAR(32) NOT NULL DEFAULT 'all_at_once',
    batch_size INT NOT NULL DEFAULT 0,
    total_devices INT NOT NULL,
    completed INT NOT NULL DEFAULT 0,
    failed INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_by CHAR(26) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_bt_tenant (tenant_id),
    KEY idx_bt_group (device_group_id),
    KEY idx_bt_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
