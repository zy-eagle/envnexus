-- Phase 2/3/4/6 Extension Tables

-- Role bindings (Phase 3: RBAC)
CREATE TABLE IF NOT EXISTS role_bindings (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    role_id CHAR(26) NOT NULL,
    granted_by CHAR(26) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_role_bindings_user_role (user_id, role_id),
    KEY idx_role_bindings_tenant (tenant_id),
    KEY idx_role_bindings_user (user_id),
    CONSTRAINT fk_rb_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_rb_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_rb_role FOREIGN KEY (role_id) REFERENCES roles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Device heartbeats (Phase 2)
CREATE TABLE IF NOT EXISTS device_heartbeats (
    id CHAR(26) PRIMARY KEY,
    device_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    agent_version VARCHAR(64) NULL,
    platform VARCHAR(64) NULL,
    arch VARCHAR(32) NULL,
    ip_address VARCHAR(64) NULL,
    metadata_json JSON NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_heartbeats_device_time (device_id, created_at),
    KEY idx_heartbeats_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Session messages (Phase 2)
CREATE TABLE IF NOT EXISTS session_messages (
    id CHAR(26) PRIMARY KEY,
    session_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    role VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_session_messages_session (session_id, created_at),
    CONSTRAINT fk_sm_session FOREIGN KEY (session_id) REFERENCES sessions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Governance baselines (Phase 2)
CREATE TABLE IF NOT EXISTS governance_baselines (
    id CHAR(26) PRIMARY KEY,
    device_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    snapshot_json JSON NOT NULL,
    captured_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_gov_baselines_device (device_id, captured_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Governance drifts (Phase 2)
CREATE TABLE IF NOT EXISTS governance_drifts (
    id CHAR(26) PRIMARY KEY,
    device_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    baseline_id CHAR(26) NULL,
    drift_type VARCHAR(64) NOT NULL,
    key_name VARCHAR(255) NOT NULL,
    expected_value TEXT NULL,
    actual_value TEXT NULL,
    severity VARCHAR(32) NOT NULL DEFAULT 'medium',
    detected_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    resolved_at DATETIME(3) NULL,
    KEY idx_gov_drifts_device (device_id, detected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Webhook subscriptions (Phase 4)
CREATE TABLE IF NOT EXISTS webhook_subscriptions (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(128) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    event_types JSON NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_webhooks_tenant (tenant_id, status),
    CONSTRAINT fk_ws_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Webhook deliveries (Phase 4)
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id CHAR(26) PRIMARY KEY,
    subscription_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_json JSON NOT NULL,
    idempotency_key VARCHAR(128) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    http_status INT NULL,
    response_body TEXT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    next_retry_at DATETIME(3) NULL,
    delivered_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_deliveries_idempotency (subscription_id, idempotency_key),
    KEY idx_deliveries_status (status, next_retry_at),
    CONSTRAINT fk_wd_sub FOREIGN KEY (subscription_id) REFERENCES webhook_subscriptions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Jobs (Phase 4)
CREATE TABLE IF NOT EXISTS jobs (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NULL,
    job_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    payload_json JSON NULL,
    result_json JSON NULL,
    error_message TEXT NULL,
    priority INT NOT NULL DEFAULT 5,
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    scheduled_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    started_at DATETIME(3) NULL,
    completed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_jobs_status_priority (status, priority, scheduled_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Usage metrics (Phase 6)
CREATE TABLE IF NOT EXISTS usage_metrics (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    metric_type VARCHAR(64) NOT NULL,
    value BIGINT NOT NULL DEFAULT 0,
    period_start DATETIME(3) NOT NULL,
    period_end DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_usage_tenant_metric_period (tenant_id, metric_type, period_start),
    KEY idx_usage_tenant_time (tenant_id, period_start),
    CONSTRAINT fk_um_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Licenses (Phase 6)
CREATE TABLE IF NOT EXISTS licenses (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    license_key VARCHAR(512) NOT NULL,
    plan_code VARCHAR(32) NOT NULL DEFAULT 'enterprise',
    max_devices INT NOT NULL DEFAULT 100,
    features_json JSON NULL,
    issued_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at DATETIME(3) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_licenses_key (license_key),
    KEY idx_licenses_tenant (tenant_id),
    CONSTRAINT fk_lic_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Policy snapshots (Phase 5)
CREATE TABLE IF NOT EXISTS policy_snapshots (
    id CHAR(26) PRIMARY KEY,
    policy_profile_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    version INT NOT NULL,
    policy_json JSON NOT NULL,
    changed_by CHAR(26) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_ps_profile_version (policy_profile_id, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Device labels (Phase 4)
CREATE TABLE IF NOT EXISTS device_labels (
    id CHAR(26) PRIMARY KEY,
    device_id CHAR(26) NOT NULL,
    tenant_id CHAR(26) NOT NULL,
    label_key VARCHAR(64) NOT NULL,
    label_value VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_device_label (device_id, label_key),
    KEY idx_device_labels_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
