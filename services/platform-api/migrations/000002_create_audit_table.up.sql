-- +goose Up
CREATE TABLE audit_events (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL,
    device_id VARCHAR(32) NOT NULL,
    session_id VARCHAR(32) DEFAULT '',
    action_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    payload JSON,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tenant_device (tenant_id, device_id),
    INDEX idx_created_at (created_at)
);

-- +goose Down
DROP TABLE IF EXISTS audit_events;
