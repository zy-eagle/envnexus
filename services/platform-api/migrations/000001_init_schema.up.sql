-- EnvNexus Platform - Consolidated Initial Schema
-- All 13 core tables per proposal §12.6

CREATE TABLE tenants (
    id CHAR(26) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    slug VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    plan_code VARCHAR(32) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_tenants_slug (slug),
    KEY idx_tenants_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE users (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    email VARCHAR(191) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    password_hash VARCHAR(255) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_login_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    UNIQUE KEY uk_users_tenant_email (tenant_id, email),
    KEY idx_users_tenant_status (tenant_id, status),
    CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE roles (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(64) NOT NULL,
    permissions_json JSON NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_roles_tenant_name (tenant_id, name),
    CONSTRAINT fk_roles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE model_profiles (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(128) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    model_name VARCHAR(128) NOT NULL,
    params_json JSON NOT NULL,
    secret_mode VARCHAR(32) NOT NULL,
    fallback_model_profile_id CHAR(26) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    KEY idx_model_profiles_tenant_status (tenant_id, status),
    CONSTRAINT fk_model_profiles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE UNIQUE INDEX uk_model_profiles_tenant_name
    ON model_profiles (tenant_id, name, (IFNULL(deleted_at, '1970-01-01 00:00:00')));

CREATE TABLE policy_profiles (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(128) NOT NULL,
    policy_json JSON NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    KEY idx_policy_profiles_tenant_status (tenant_id, status),
    CONSTRAINT fk_policy_profiles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE UNIQUE INDEX uk_policy_profiles_tenant_name
    ON policy_profiles (tenant_id, name, (IFNULL(deleted_at, '1970-01-01 00:00:00')));

CREATE TABLE agent_profiles (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(128) NOT NULL,
    model_profile_id CHAR(26) NOT NULL,
    policy_profile_id CHAR(26) NOT NULL,
    capabilities_json JSON NOT NULL,
    update_channel VARCHAR(32) NOT NULL DEFAULT 'stable',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    KEY idx_agent_profiles_tenant_status (tenant_id, status),
    KEY idx_agent_profiles_model_policy (model_profile_id, policy_profile_id),
    CONSTRAINT fk_agent_profiles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE UNIQUE INDEX uk_agent_profiles_tenant_name
    ON agent_profiles (tenant_id, name, (IFNULL(deleted_at, '1970-01-01 00:00:00')));

CREATE TABLE download_packages (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    agent_profile_id CHAR(26) NOT NULL,
    distribution_mode VARCHAR(32) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    arch VARCHAR(32) NOT NULL,
    version VARCHAR(32) NOT NULL,
    package_name VARCHAR(255) NOT NULL,
    download_url VARCHAR(1024) NOT NULL DEFAULT '',
    artifact_path VARCHAR(1024) NOT NULL DEFAULT '',
    artifact_size BIGINT UNSIGNED NOT NULL DEFAULT 0,
    checksum VARCHAR(128) NOT NULL DEFAULT '',
    bootstrap_manifest_json JSON NULL,
    branding_version INT UNSIGNED NOT NULL DEFAULT 1,
    build_version VARCHAR(64) NOT NULL DEFAULT '',
    sign_status VARCHAR(32) NOT NULL DEFAULT 'unsigned',
    sign_metadata_json JSON NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    published_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_download_packages_profile_platform (agent_profile_id, distribution_mode, platform, arch, version),
    KEY idx_download_packages_tenant_created (tenant_id, created_at),
    KEY idx_download_packages_tenant_status (tenant_id, status, published_at),
    CONSTRAINT fk_download_packages_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE enrollment_tokens (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    agent_profile_id CHAR(26) NOT NULL,
    download_package_id CHAR(26) NOT NULL DEFAULT '',
    token_hash CHAR(64) NOT NULL,
    channel VARCHAR(32) NOT NULL DEFAULT 'stable',
    expires_at DATETIME(3) NOT NULL,
    max_uses INT UNSIGNED NOT NULL DEFAULT 1,
    used_count INT UNSIGNED NOT NULL DEFAULT 0,
    issued_by_user_id CHAR(26) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_enrollment_tokens_hash (token_hash),
    KEY idx_enrollment_tokens_tenant_status (tenant_id, status, expires_at),
    KEY idx_enrollment_tokens_package (download_package_id, status),
    CONSTRAINT fk_enrollment_tokens_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE devices (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    agent_profile_id CHAR(26) NOT NULL DEFAULT '',
    device_name VARCHAR(128) NOT NULL,
    hostname VARCHAR(191) NULL,
    platform VARCHAR(32) NOT NULL,
    arch VARCHAR(32) NOT NULL DEFAULT 'amd64',
    environment_type VARCHAR(32) NOT NULL DEFAULT 'physical',
    agent_version VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending_activation',
    policy_version INT UNSIGNED NOT NULL DEFAULT 1,
    last_seen_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    KEY idx_devices_tenant_status (tenant_id, status, last_seen_at),
    KEY idx_devices_tenant_profile (tenant_id, agent_profile_id),
    KEY idx_devices_hostname (hostname),
    CONSTRAINT fk_devices_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE sessions (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    device_id CHAR(26) NOT NULL,
    transport VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    initiator_type VARCHAR(32) NOT NULL DEFAULT 'user',
    started_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    ended_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_sessions_device_started (device_id, started_at),
    KEY idx_sessions_tenant_status (tenant_id, status, started_at),
    CONSTRAINT fk_sessions_device FOREIGN KEY (device_id) REFERENCES devices(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE tool_invocations (
    id CHAR(26) PRIMARY KEY,
    session_id CHAR(26) NOT NULL,
    device_id CHAR(26) NOT NULL,
    tool_name VARCHAR(128) NOT NULL,
    risk_level VARCHAR(16) NOT NULL,
    input_json JSON NOT NULL,
    output_json JSON NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    duration_ms INT UNSIGNED NULL,
    started_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    finished_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_tool_invocations_session (session_id, started_at),
    KEY idx_tool_invocations_device_tool (device_id, tool_name),
    KEY idx_tool_invocations_status (status, created_at),
    CONSTRAINT fk_tool_invocations_session FOREIGN KEY (session_id) REFERENCES sessions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE approval_requests (
    id CHAR(26) PRIMARY KEY,
    session_id CHAR(26) NOT NULL,
    device_id CHAR(26) NOT NULL,
    requested_action_json JSON NOT NULL,
    risk_level VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'drafted',
    approver_user_id CHAR(26) NULL,
    approved_at DATETIME(3) NULL,
    expires_at DATETIME(3) NULL,
    executed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_approval_requests_session (session_id, status),
    KEY idx_approval_requests_device (device_id, created_at),
    KEY idx_approval_requests_approver (approver_user_id, status),
    CONSTRAINT fk_approval_requests_session FOREIGN KEY (session_id) REFERENCES sessions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE audit_events (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    device_id CHAR(26) NULL,
    session_id CHAR(26) NULL,
    event_type VARCHAR(64) NOT NULL,
    event_payload_json JSON NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_audit_events_tenant_created (tenant_id, created_at),
    KEY idx_audit_events_device_created (device_id, created_at),
    KEY idx_audit_events_session_created (session_id, created_at),
    KEY idx_audit_events_type_created (event_type, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
