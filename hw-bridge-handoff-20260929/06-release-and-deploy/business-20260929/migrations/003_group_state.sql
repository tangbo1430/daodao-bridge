SET @group_state_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='group_map' AND column_name='group_state_json');
SET @group_state_ddl = IF(@group_state_exists=0, 'ALTER TABLE group_map ADD COLUMN group_state_json LONGTEXT NULL', 'SELECT 1');
PREPARE group_state_stmt FROM @group_state_ddl;
EXECUTE group_state_stmt;
DEALLOCATE PREPARE group_state_stmt;
