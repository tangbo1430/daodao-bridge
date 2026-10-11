-- Additive; retain these records on rollback. No account credentials are stored.
CREATE TABLE IF NOT EXISTS bridge_message_record (
  record_key CHAR(64) NOT NULL PRIMARY KEY,
  tencent_key VARCHAR(255) NOT NULL, tang_id VARCHAR(255) NOT NULL,
  tencent_from VARCHAR(255) NOT NULL, tencent_to VARCHAR(255) NOT NULL,
  channel_type INT NOT NULL, occurred_at BIGINT NOT NULL,
  read_synced TINYINT NOT NULL DEFAULT 0, revoked TINYINT NOT NULL DEFAULT 0,
  record_json LONGTEXT NOT NULL,
  KEY idx_native_tencent (tencent_key), KEY idx_native_tang (tang_id),
  KEY idx_read_owner (tencent_to,tencent_from,occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS bridge_business_cursor (
  scope_key VARCHAR(255) NOT NULL PRIMARY KEY,
  cursor_json LONGTEXT NOT NULL, updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
