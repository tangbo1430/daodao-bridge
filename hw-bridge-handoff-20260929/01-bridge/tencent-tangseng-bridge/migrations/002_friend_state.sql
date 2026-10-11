-- Additive migration. Existing status rows and legacy binaries remain valid.
SET @friend_state_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='friend_link' AND column_name='friend_state_json');
SET @friend_state_ddl = IF(@friend_state_exists=0, 'ALTER TABLE friend_link ADD COLUMN friend_state_json LONGTEXT NULL', 'SELECT 1');
PREPARE friend_state_stmt FROM @friend_state_ddl;
EXECUTE friend_state_stmt;
DEALLOCATE PREPARE friend_state_stmt;
