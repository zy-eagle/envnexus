-- +goose Up
CREATE TABLE download_packages (
    id VARCHAR(32) PRIMARY KEY,
    tenant_id VARCHAR(32) NOT NULL,
    agent_profile_id VARCHAR(32) NOT NULL,
    distribution_mode VARCHAR(50) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    arch VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    package_name VARCHAR(255) NOT NULL,
    download_url TEXT,
    artifact_path TEXT,
    checksum VARCHAR(255),
    sign_status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_tenant_platform (tenant_id, platform)
);

-- +goose Down
DROP TABLE IF EXISTS download_packages;
