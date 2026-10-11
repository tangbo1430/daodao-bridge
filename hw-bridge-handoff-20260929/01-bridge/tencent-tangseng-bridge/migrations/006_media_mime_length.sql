-- Additive widening; retain this width when rolling back the application.
SET @mime_length = (SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='media_cache' AND column_name='media_type');
SET @mime_ddl = IF(@mime_length < 255, 'ALTER TABLE media_cache MODIFY COLUMN media_type VARCHAR(255) NOT NULL DEFAULT ''''', 'SELECT 1');
PREPARE mime_stmt FROM @mime_ddl;
EXECUTE mime_stmt;
DEALLOCATE PREPARE mime_stmt;
