SET @schema_name = DATABASE();

SET @sql = IF(
  EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'yy_group_v'
      AND COLUMN_NAME = 'client_request_id'
  ),
  'SELECT 1',
  'ALTER TABLE `yy_group_v` ADD COLUMN `client_request_id` VARCHAR(64) NULL COMMENT ''客户端建群幂等标识'' AFTER `system_type`'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = IF(
  EXISTS(
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'yy_group_v'
      AND INDEX_NAME = 'uk_group_create_request'
  ),
  'SELECT 1',
  'ALTER TABLE `yy_group_v` ADD UNIQUE KEY `uk_group_create_request` (`uid`, `client_request_id`)'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
