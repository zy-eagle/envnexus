-- Governance Rules (M6)
CREATE TABLE IF NOT EXISTS governance_rules (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    rule_type VARCHAR(64) NOT NULL,
    condition_json TEXT NOT NULL,
    action_json TEXT NULL,
    severity VARCHAR(16) NOT NULL DEFAULT 'warning',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_by CHAR(26) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_gr_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Tool Permissions (M6)
CREATE TABLE IF NOT EXISTS tool_permissions (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    tool_name VARCHAR(128) NOT NULL,
    role_id CHAR(26) NULL,
    allowed TINYINT(1) NOT NULL DEFAULT 1,
    max_risk VARCHAR(8) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_tp_tenant (tenant_id),
    KEY idx_tp_tool (tool_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
