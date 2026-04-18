-- File access requests (M4: File Forensics)
CREATE TABLE IF NOT EXISTS file_access_requests (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL,
    device_id CHAR(26) NOT NULL,
    requested_by CHAR(26) NOT NULL,
    approved_by CHAR(26) NULL,
    path VARCHAR(1024) NOT NULL,
    action VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    result_json TEXT NULL,
    note TEXT NULL,
    expires_at DATETIME(3) NOT NULL,
    resolved_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_far_tenant (tenant_id),
    KEY idx_far_device (device_id),
    KEY idx_far_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
