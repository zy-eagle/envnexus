ALTER TABLE `device_bindings` DROP FOREIGN KEY `fk_db_package`;
ALTER TABLE `device_bindings` DROP FOREIGN KEY `fk_db_tenant`;
ALTER TABLE `licenses` DROP FOREIGN KEY `fk_lic_tenant`;
ALTER TABLE `session_messages` DROP FOREIGN KEY `fk_sm_session`;
ALTER TABLE `usage_metrics` DROP FOREIGN KEY `fk_um_tenant`;
ALTER TABLE `webhook_deliveries` DROP FOREIGN KEY `fk_wd_sub`;
ALTER TABLE `webhook_subscriptions` DROP FOREIGN KEY `fk_ws_tenant`;
