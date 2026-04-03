-- Remote Command Approval Module Tables

CREATE TABLE IF NOT EXISTS command_tasks (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    created_by_user_id CHAR(26) NOT NULL,
    approver_user_id CHAR(26) NULL,
    approver_role_id CHAR(26) NULL,
    approved_by_id CHAR(26) NULL,
    title VARCHAR(255) NOT NULL,
    command_type VARCHAR(64) NOT NULL,
    command_payload TEXT NOT NULL,
    device_ids TEXT NOT NULL,
    risk_level VARCHAR(8) NOT NULL,
    effective_risk VARCHAR(8) NOT NULL,
    bypass_approval TINYINT(1) NOT NULL DEFAULT 0,
    bypass_reason TEXT NULL,
    emergency TINYINT(1) NOT NULL DEFAULT 0,
    policy_snapshot_id CHAR(26) NULL,
    target_env VARCHAR(32) NULL DEFAULT '',
    change_ticket VARCHAR(255) NULL DEFAULT '',
    business_app VARCHAR(255) NULL DEFAULT '',
    note TEXT NULL,
    status VARCHAR(32) NOT NULL,
    approval_note TEXT NULL,
    expires_at DATETIME(3) NOT NULL,
    approved_at DATETIME(3) NULL,
    completed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_command_tasks_tenant_status (tenant_id, status, created_at),
    KEY idx_command_tasks_created_by (created_by_user_id, created_at),
    KEY idx_command_tasks_approver (approver_user_id, status),
    KEY idx_command_tasks_approver_role (approver_role_id, status),
    KEY idx_command_tasks_expires (status, expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS command_executions (
    id CHAR(26) PRIMARY KEY,
    task_id CHAR(26) NOT NULL,
    device_id CHAR(26) NOT NULL,
    status VARCHAR(32) NOT NULL,
    output TEXT NULL,
    error_message TEXT NULL,
    exit_code INT NULL,
    duration_ms INT NULL,
    sent_at DATETIME(3) NULL,
    started_at DATETIME(3) NULL,
    finished_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_command_executions_task (task_id, status),
    KEY idx_command_executions_device (device_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS approval_policies (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(128) NOT NULL,
    risk_level VARCHAR(8) NOT NULL,
    approver_user_id CHAR(26) NULL,
    approver_role_id CHAR(26) NULL,
    auto_approve TINYINT(1) NOT NULL DEFAULT 0,
    approval_rule VARCHAR(32) NOT NULL DEFAULT 'single',
    separation_of_duty TINYINT(1) NOT NULL DEFAULT 0,
    expires_minutes INT NOT NULL DEFAULT 30,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    priority INT NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_approval_policies_tenant_risk (tenant_id, risk_level, priority, status),
    KEY idx_approval_policies_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS im_providers (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    name VARCHAR(128) NOT NULL,
    config_json TEXT NOT NULL,
    webhook_url VARCHAR(512) NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_im_providers_tenant (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_notification_channels (
    id CHAR(26) PRIMARY KEY,
    user_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    provider_id CHAR(26) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    external_name VARCHAR(128) NULL DEFAULT '',
    chat_id VARCHAR(255) NULL DEFAULT '',
    priority INT NOT NULL DEFAULT 0,
    verified TINYINT(1) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_user_notification_channels_user (user_id, status),
    KEY idx_user_notification_channels_tenant (tenant_id),
    KEY idx_user_notification_channels_provider (provider_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
