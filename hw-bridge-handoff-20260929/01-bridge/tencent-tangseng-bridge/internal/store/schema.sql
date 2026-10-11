CREATE TABLE IF NOT EXISTS bridge_identity_mapping (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_system VARCHAR(20) NOT NULL,
  source_id VARCHAR(255) NOT NULL,
  target_system VARCHAR(20) NOT NULL,
  target_id VARCHAR(255) NOT NULL,
  entity_type VARCHAR(20) NOT NULL DEFAULT 'user',
  display_name VARCHAR(255) NOT NULL DEFAULT '',
  avatar_url VARCHAR(1000) NOT NULL DEFAULT '',
  shadow TINYINT(1) NOT NULL DEFAULT 1,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_identity_source (source_system, source_id),
  UNIQUE KEY uk_identity_target (target_system, target_id),
  KEY idx_identity_status (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS bridge_inbox_event (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event_id VARCHAR(128) NOT NULL,
  idempotency_key VARCHAR(255) NOT NULL,
  trace_id VARCHAR(128) NOT NULL DEFAULT '',
  source_system VARCHAR(20) NOT NULL,
  command_name VARCHAR(128) NOT NULL DEFAULT '',
  occurred_at BIGINT NOT NULL DEFAULT 0,
  payload_json LONGTEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'accepted',
  created_at BIGINT NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_inbox_idempotency (idempotency_key),
  KEY idx_inbox_status (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS bridge_message_mapping (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_system VARCHAR(20) NOT NULL,
  source_message_id VARCHAR(255) NOT NULL,
  target_system VARCHAR(20) NOT NULL,
  target_message_id VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'done',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_message_source (source_system, source_message_id),
  KEY idx_message_target (target_system, target_message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS bridge_outbox_job (
  id VARCHAR(128) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  target_system VARCHAR(20) NOT NULL,
  message_json LONGTEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  attempts INT UNSIGNED NOT NULL DEFAULT 0,
  next_attempt_at BIGINT NOT NULL,
  last_error VARCHAR(1000) NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (id),
  KEY idx_outbox_due (status, next_attempt_at),
  KEY idx_outbox_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS bridge_proxy_session (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id VARCHAR(255) NOT NULL,
  node_id VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'offline',
  token_expires_at BIGINT NOT NULL DEFAULT 0,
  last_seen_at BIGINT NOT NULL DEFAULT 0,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_proxy_account (account_id),
  KEY idx_proxy_status (status, last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS bridge_dead_letter (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  job_id VARCHAR(128) NOT NULL, event_id VARCHAR(128) NOT NULL, reason VARCHAR(1000) NOT NULL DEFAULT '',
  payload_json LONGTEXT NOT NULL, attempts INT UNSIGNED NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'dead', created_at BIGINT NOT NULL, retried_at BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id), UNIQUE KEY uk_dead_job (job_id), KEY idx_dead_status (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- A+ account, relation, group, cursor and media projections. The original
-- bridge_* tables remain the compatibility inbox/outbox store.
CREATE TABLE IF NOT EXISTS hw_user (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tencent_user_id VARCHAR(255) NOT NULL,
  display_name VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(20) NOT NULL DEFAULT 'reserved',
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  PRIMARY KEY (id), UNIQUE KEY uk_hw_user (tencent_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS tsdd_shadow (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  hw_user_id BIGINT UNSIGNED NOT NULL,
  username VARCHAR(255) NOT NULL, uid VARCHAR(255) NOT NULL DEFAULT '',
  tenant_id VARCHAR(255) NOT NULL DEFAULT '', device_id VARCHAR(255) NOT NULL DEFAULT '',
  token_ciphertext VARBINARY(4096) NOT NULL, token_expires_at BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'reserved', created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  PRIMARY KEY (id), UNIQUE KEY uk_tsdd_hw_user (hw_user_id), UNIQUE KEY uk_tsdd_uid (uid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS msg_map (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, source_system VARCHAR(20) NOT NULL,
  source_message_id VARCHAR(255) NOT NULL, target_system VARCHAR(20) NOT NULL,
  target_message_id VARCHAR(255) NOT NULL DEFAULT '', conversation_id VARCHAR(255) NOT NULL DEFAULT '',
  message_seq BIGINT NOT NULL DEFAULT 0, status VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL, PRIMARY KEY (id),
  UNIQUE KEY uk_msg_source (source_system, source_message_id), KEY idx_msg_conversation (conversation_id, message_seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS group_map (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, source_group_id VARCHAR(255) NOT NULL,
  target_group_id VARCHAR(255) NOT NULL DEFAULT '', source_system VARCHAR(20) NOT NULL,
  target_system VARCHAR(20) NOT NULL, status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL, PRIMARY KEY (id),
  UNIQUE KEY uk_group_source (source_system, source_group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS group_member_mirror (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, group_source VARCHAR(20) NOT NULL,
  group_id VARCHAR(255) NOT NULL, member_source VARCHAR(20) NOT NULL,
  member_id VARCHAR(255) NOT NULL, role VARCHAR(20) NOT NULL DEFAULT 'member', status VARCHAR(20) NOT NULL DEFAULT 'active',
  updated_at BIGINT NOT NULL, PRIMARY KEY (id), UNIQUE KEY uk_group_member (group_source, group_id, member_source, member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS friend_link (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, source_system VARCHAR(20) NOT NULL,
  owner_id VARCHAR(255) NOT NULL, peer_id VARCHAR(255) NOT NULL, relation_type VARCHAR(20) NOT NULL DEFAULT 'friend',
  status VARCHAR(20) NOT NULL DEFAULT 'active', updated_at BIGINT NOT NULL, PRIMARY KEY (id),
  UNIQUE KEY uk_friend_link (source_system, owner_id, peer_id, relation_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS vercode_cache (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, username VARCHAR(255) NOT NULL,
  code_ciphertext VARBINARY(1024) NOT NULL, expires_at BIGINT NOT NULL, created_at BIGINT NOT NULL,
  PRIMARY KEY (id), UNIQUE KEY uk_vercode_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
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
  scope_key VARCHAR(255) NOT NULL PRIMARY KEY, cursor_json LONGTEXT NOT NULL, updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS sync_cursor (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, source_system VARCHAR(20) NOT NULL,
  account_id VARCHAR(255) NOT NULL, cursor_value VARCHAR(255) NOT NULL DEFAULT '',
  registered_at BIGINT NOT NULL, updated_at BIGINT NOT NULL, PRIMARY KEY (id),
  UNIQUE KEY uk_sync_cursor (source_system, account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS tim_cursor (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, scope_key VARCHAR(255) NOT NULL,
  cursor_value VARCHAR(255) NOT NULL DEFAULT '', registered_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  PRIMARY KEY (id), UNIQUE KEY uk_tim_cursor (scope_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS job (
  id VARCHAR(128) NOT NULL, kind VARCHAR(50) NOT NULL, payload_json LONGTEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending', attempts INT UNSIGNED NOT NULL DEFAULT 0,
  next_attempt_at BIGINT NOT NULL, last_error VARCHAR(1000) NOT NULL DEFAULT '', created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  PRIMARY KEY (id), KEY idx_job_due (status, next_attempt_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS media_cache (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, sha256 CHAR(64) NOT NULL, source_url VARCHAR(2000) NOT NULL DEFAULT '',
  object_url VARCHAR(2000) NOT NULL DEFAULT '', media_type VARCHAR(255) NOT NULL DEFAULT '', size_bytes BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'pending', created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  PRIMARY KEY (id), UNIQUE KEY uk_media_sha256 (sha256)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS audit_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, trace_id VARCHAR(128) NOT NULL DEFAULT '', action_name VARCHAR(128) NOT NULL,
  actor VARCHAR(255) NOT NULL DEFAULT '', payload_json LONGTEXT NOT NULL, created_at BIGINT NOT NULL,
  PRIMARY KEY (id), KEY idx_audit_trace (trace_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
