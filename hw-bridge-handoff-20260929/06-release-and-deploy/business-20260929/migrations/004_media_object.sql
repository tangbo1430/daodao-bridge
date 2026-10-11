SET @media_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='media_cache' AND column_name='object_json');
SET @media_ddl = IF(@media_exists=0, 'ALTER TABLE media_cache ADD COLUMN object_json LONGTEXT NULL', 'SELECT 1');
PREPARE media_stmt FROM @media_ddl;
EXECUTE media_stmt;
DEALLOCATE PREPARE media_stmt;
