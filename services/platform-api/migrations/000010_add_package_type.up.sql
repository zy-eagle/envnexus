ALTER TABLE download_packages
    ADD COLUMN package_type VARCHAR(32) NOT NULL DEFAULT 'installer' AFTER version;

ALTER TABLE download_packages
    DROP INDEX uk_download_packages_profile_platform;

ALTER TABLE download_packages
    ADD UNIQUE INDEX uk_download_packages_profile_platform (agent_profile_id, distribution_mode, platform, arch, version, package_type);
