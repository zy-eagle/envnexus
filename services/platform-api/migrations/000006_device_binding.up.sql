-- Device Activation & Binding System

-- Extend download_packages with activation fields
ALTER TABLE download_packages
  ADD COLUMN activation_mode VARCHAR(16) NOT NULL DEFAULT 'auto' COMMENT 'auto, manual, or both' AFTER sign_metadata_json,
  ADD COLUMN activation_key_hash VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'SHA-256 of activation key (auto mode only)' AFTER activation_mode,
  ADD COLUMN max_devices INT NOT NULL DEFAULT 1 COMMENT 'Max bindable devices' AFTER activation_key_hash,
  ADD COLUMN bound_count INT NOT NULL DEFAULT 0 COMMENT 'Currently bound device count' AFTER max_devices;

-- Device bindings: tracks which devices are bound to which packages
CREATE TABLE IF NOT EXISTS device_bindings (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL,
    package_id      CHAR(26) NOT NULL,
    device_code     VARCHAR(20) NOT NULL COMMENT 'ENX-XXXX-XXXX-XXXX',
    hardware_hash   VARCHAR(64) NOT NULL COMMENT 'SHA-256 composite fingerprint',
    device_info     JSON NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    bound_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    bound_by        VARCHAR(26) NOT NULL DEFAULT '' COMMENT 'admin user ID or system',
    revoked_at      DATETIME(3) NULL,
    last_heartbeat  DATETIME(3) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    UNIQUE KEY uk_device_bindings_code (device_code),
    KEY idx_device_bindings_package (package_id, status),
    KEY idx_device_bindings_tenant (tenant_id),
    CONSTRAINT fk_db_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_db_package FOREIGN KEY (package_id) REFERENCES download_packages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Device components: individual hardware component hashes for tolerance matching
CREATE TABLE IF NOT EXISTS device_components (
    id              CHAR(26) PRIMARY KEY,
    device_code     VARCHAR(20) NOT NULL,
    component_type  VARCHAR(16) NOT NULL COMMENT 'cpu/board/mac/disk/gpu',
    component_hash  VARCHAR(64) NOT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_device_components_code (device_code),
    KEY idx_device_components_type_hash (component_type, component_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Pending devices: devices that reported their code but are not yet bound
CREATE TABLE IF NOT EXISTS pending_devices (
    id              CHAR(26) PRIMARY KEY,
    device_code     VARCHAR(20) NOT NULL,
    hardware_hash   VARCHAR(64) NOT NULL,
    device_info     JSON NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    UNIQUE KEY uk_pending_devices_code (device_code),
    KEY idx_pending_devices_hash (hardware_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Activation audit logs
CREATE TABLE IF NOT EXISTS activation_audit_logs (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL,
    package_id      CHAR(26) NOT NULL,
    device_code     VARCHAR(20) NOT NULL DEFAULT '',
    action          VARCHAR(32) NOT NULL COMMENT 'activate/bind/unbind/revoke/heartbeat_fail',
    actor           VARCHAR(26) NOT NULL DEFAULT '',
    detail          JSON NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_aal_tenant_time (tenant_id, created_at),
    KEY idx_aal_package (package_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
