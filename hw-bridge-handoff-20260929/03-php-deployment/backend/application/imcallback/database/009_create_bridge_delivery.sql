CREATE TABLE IF NOT EXISTS `yy_im_bridge_delivery` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `attempts` INT UNSIGNED NOT NULL DEFAULT 0,
  `next_retry_at` INT UNSIGNED NOT NULL DEFAULT 0,
  `last_error` VARCHAR(1000) NOT NULL DEFAULT '',
  `delivered_at` INT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` INT UNSIGNED NOT NULL,
  `updated_at` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_bridge_event` (`event_id`),
  KEY `idx_bridge_due` (`status`, `next_retry_at`),
  CONSTRAINT `fk_bridge_event` FOREIGN KEY (`event_id`) REFERENCES `yy_im_callback_event` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;
