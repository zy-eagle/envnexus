ALTER TABLE devices ADD COLUMN IF NOT EXISTS distribution_package_version VARCHAR(64) DEFAULT '' NOT NULL;
