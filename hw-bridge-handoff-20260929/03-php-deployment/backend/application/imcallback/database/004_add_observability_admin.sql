ALTER TABLE `yy_im_callback_event`
  ADD COLUMN `manual_retry_count` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `retry_count`,
  ADD KEY `idx_received_at` (`received_at`),
  ADD KEY `idx_event_time` (`event_time`);

CREATE TABLE IF NOT EXISTS `yy_im_callback_worker_heartbeat` (
  `worker_name` VARCHAR(64) NOT NULL,
  `host_name` VARCHAR(255) NOT NULL DEFAULT '',
  `process_id` INT UNSIGNED NOT NULL DEFAULT 0,
  `last_seen_at` INT UNSIGNED NOT NULL,
  `last_batch_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`worker_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `yy_im_callback_admin_action` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `admin_id` INT UNSIGNED NOT NULL,
  `event_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `action` VARCHAR(40) NOT NULL,
  `before_status` VARCHAR(100) NOT NULL DEFAULT '',
  `after_status` VARCHAR(100) NOT NULL DEFAULT '',
  `result` VARCHAR(20) NOT NULL,
  `error_message` VARCHAR(500) NOT NULL DEFAULT '',
  `trace_id` VARCHAR(64) NOT NULL,
  `ip_address` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_event_created` (`event_id`,`created_at`),
  KEY `idx_admin_created` (`admin_id`,`created_at`),
  KEY `idx_action_created` (`action`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

ALTER TABLE `yy_im_group_snapshot`
  ADD COLUMN `source_event_time` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `source_event_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE `yy_im_group_member_snapshot`
  ADD COLUMN `source_event_time` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `source_event_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE `yy_im_relation_snapshot`
  ADD COLUMN `source_event_time` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `source_event_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE `yy_im_user_profile_snapshot`
  ADD COLUMN `source_event_time` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `source_event_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE `yy_im_user_state_snapshot`
  ADD COLUMN `source_event_time` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `source_event_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
