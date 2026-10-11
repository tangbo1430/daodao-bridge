CREATE TABLE IF NOT EXISTS `yy_im_account_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `uid` int unsigned NOT NULL,
  `im_user_id` varchar(255) NOT NULL,
  `sdk_app_id` varchar(32) NOT NULL,
  `import_status` varchar(20) NOT NULL DEFAULT 'pending',
  `import_error` varchar(1000) NOT NULL DEFAULT '',
  `imported_at` int unsigned NOT NULL DEFAULT 0,
  `created_at` int unsigned NOT NULL,
  `updated_at` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid_app` (`uid`,`sdk_app_id`),
  UNIQUE KEY `uk_im_user_app` (`im_user_id`,`sdk_app_id`),
  KEY `idx_import_status` (`import_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `yy_im_sig_audit` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `trace_id` varchar(64) NOT NULL,
  `uid` int unsigned NOT NULL DEFAULT 0,
  `im_user_id` varchar(255) NOT NULL DEFAULT '',
  `sdk_app_id` varchar(32) NOT NULL DEFAULT '',
  `result` varchar(20) NOT NULL,
  `error_message` varchar(500) NOT NULL DEFAULT '',
  `issued_at` int unsigned NOT NULL,
  `expires_at` int unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_uid_issued` (`uid`,`issued_at`),
  KEY `idx_trace_id` (`trace_id`),
  KEY `idx_result_issued` (`result`,`issued_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC;
