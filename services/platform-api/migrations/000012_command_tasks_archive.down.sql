ALTER TABLE command_tasks
    DROP INDEX idx_command_tasks_archived,
    DROP COLUMN archived_at;

