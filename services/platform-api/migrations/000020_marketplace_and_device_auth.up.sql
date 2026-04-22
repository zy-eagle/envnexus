CREATE TABLE `marketplace_items` (
  `id` varchar(26) NOT NULL,
  `type` varchar(32) NOT NULL,
  `name` varchar(255) NOT NULL,
  `description` text,
  `version` varchar(64) NOT NULL,
  `author` varchar(255) DEFAULT NULL,
  `payload` json NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'draft',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_marketplace_items_type` (`type`),
  KEY `idx_marketplace_items_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `tenant_subscriptions` (
  `id` varchar(26) NOT NULL,
  `tenant_id` varchar(26) NOT NULL,
  `item_id` varchar(26) NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_tenant_subscriptions_tenant_item` (`tenant_id`,`item_id`),
  KEY `idx_tenant_subscriptions_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `device_auth_codes` (
  `device_code` varchar(512) NOT NULL,
  `user_code` varchar(32) NOT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'pending',
  `expires_at` datetime(3) NOT NULL,
  `user_id` varchar(26) DEFAULT NULL,
  `tenant_id` varchar(26) DEFAULT NULL,
  `device_info` json DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`device_code`),
  KEY `idx_device_auth_codes_user_code` (`user_code`),
  KEY `idx_device_auth_codes_status` (`status`),
  KEY `idx_device_auth_codes_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `ide_client_tokens` (
  `id` varchar(26) NOT NULL,
  `user_id` varchar(26) NOT NULL,
  `tenant_id` varchar(26) NOT NULL,
  `name` varchar(255) NOT NULL,
  `access_token_hash` varchar(64) NOT NULL,
  `refresh_token_hash` varchar(64) NOT NULL,
  `access_expires_at` datetime(3) NOT NULL,
  `refresh_expires_at` datetime(3) NOT NULL,
  `last_used_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ide_client_tokens_user_id` (`user_id`),
  KEY `idx_ide_client_tokens_tenant_id` (`tenant_id`),
  KEY `idx_ide_client_tokens_access_token_hash` (`access_token_hash`),
  KEY `idx_ide_client_tokens_refresh_token_hash` (`refresh_token_hash`),
  KEY `idx_ide_client_tokens_access_expires_at` (`access_expires_at`),
  KEY `idx_ide_client_tokens_refresh_expires_at` (`refresh_expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;