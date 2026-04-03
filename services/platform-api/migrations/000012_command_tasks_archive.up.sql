-- Add archived_at for command tasks soft-archive (hide from default list)

ALTER TABLE command_tasks
    ADD COLUMN archived_at DATETIME(3) NULL AFTER completed_at,
    ADD KEY idx_command_tasks_archived (tenant_id, archived_at, created_at);

