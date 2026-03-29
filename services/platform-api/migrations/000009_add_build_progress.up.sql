ALTER TABLE download_packages
  ADD COLUMN build_stage VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'Current build stage: queued, downloading, packaging, uploading, done' AFTER status,
  ADD COLUMN build_progress TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Build progress percentage 0-100' AFTER build_stage;
