ALTER TABLE `yy_im_callback_event`
  ADD COLUMN `event_category` VARCHAR(32) NOT NULL DEFAULT 'unknown' AFTER `callback_command`,
  ADD COLUMN `request_id` VARCHAR(64) NOT NULL DEFAULT '' AFTER `trace_id`,
  ADD COLUMN `summary` VARCHAR(500) NOT NULL DEFAULT '' AFTER `payload_json`,
  ADD COLUMN `response_json` TEXT NULL AFTER `summary`,
  ADD COLUMN `queue_status` VARCHAR(20) NOT NULL DEFAULT 'none' AFTER `handler_status`,
  ADD COLUMN `retry_count` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `queue_status`,
  ADD COLUMN `next_retry_at` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `retry_count`,
  ADD COLUMN `duration_ms` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `processed_at`,
  ADD KEY `idx_queue_retry` (`queue_status`, `next_retry_at`),
  ADD KEY `idx_category_received` (`event_category`, `received_at`),
  ADD KEY `idx_request_id` (`request_id`);

CREATE TABLE IF NOT EXISTS `yy_im_group_snapshot` (
  `group_id` VARCHAR(255) NOT NULL,
  `group_name` VARCHAR(255) NOT NULL DEFAULT '',
  `owner_account` VARCHAR(255) NOT NULL DEFAULT '',
  `group_type` VARCHAR(50) NOT NULL DEFAULT '',
  `member_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `raw_json` LONGTEXT NOT NULL,
  `created_at` INT UNSIGNED NOT NULL DEFAULT 0,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`group_id`),
  KEY `idx_group_status` (`status`, `updated_at`),
  KEY `idx_group_owner` (`owner_account`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `yy_im_group_member_snapshot` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `group_id` VARCHAR(255) NOT NULL,
  `account` VARCHAR(255) NOT NULL,
  `role` VARCHAR(30) NOT NULL DEFAULT '',
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `join_time` INT UNSIGNED NOT NULL DEFAULT 0,
  `raw_json` LONGTEXT NOT NULL,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_group_account` (`group_id`, `account`),
  KEY `idx_member_account` (`account`),
  KEY `idx_member_status` (`status`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `yy_im_relation_snapshot` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_account` VARCHAR(255) NOT NULL,
  `peer_account` VARCHAR(255) NOT NULL,
  `relation_type` VARCHAR(20) NOT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `raw_json` LONGTEXT NOT NULL,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_relation` (`owner_account`, `peer_account`, `relation_type`),
  KEY `idx_relation_peer` (`peer_account`),
  KEY `idx_relation_status` (`relation_type`, `status`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `yy_im_user_profile_snapshot` (
  `account` VARCHAR(255) NOT NULL,
  `nick_name` VARCHAR(255) NOT NULL DEFAULT '',
  `avatar_url` VARCHAR(1000) NOT NULL DEFAULT '',
  `status` VARCHAR(20) NOT NULL DEFAULT 'active',
  `profile_json` LONGTEXT NOT NULL,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`account`),
  KEY `idx_profile_updated` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `yy_im_user_state_snapshot` (
  `account` VARCHAR(255) NOT NULL,
  `state` VARCHAR(50) NOT NULL DEFAULT '',
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `last_event_time` VARCHAR(32) NOT NULL DEFAULT '',
  `raw_json` LONGTEXT NOT NULL,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`account`),
  KEY `idx_state_updated` (`state`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;
