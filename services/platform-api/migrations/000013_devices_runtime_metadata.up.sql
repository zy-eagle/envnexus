ALTER TABLE devices
    ADD COLUMN runtime_metadata JSON NULL COMMENT 'Agent-reported runtime snapshot from heartbeat' AFTER arch;
