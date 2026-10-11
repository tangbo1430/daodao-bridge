package store

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed schema.sql
var schemaSQL string

func ApplySchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(schemaSQL, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='friend_link' AND column_name='friend_state_json'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE friend_link ADD COLUMN friend_state_json LONGTEXT NULL`); err != nil {
			return err
		}
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='group_map' AND column_name='group_state_json'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE group_map ADD COLUMN group_state_json LONGTEXT NULL`); err != nil {
			return err
		}
	}
	return ensureMediaSchema(ctx, db)
}

func ensureMediaSchema(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='media_cache' AND column_name='object_json'").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, "ALTER TABLE media_cache ADD COLUMN object_json LONGTEXT NULL"); err != nil {
			return err
		}
	}
	// Office MIME types exceed the legacy 50-character column. Widen existing
	// installations too; CREATE TABLE IF NOT EXISTS only handles fresh databases.
	var length int
	if err := db.QueryRowContext(ctx, "SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='media_cache' AND column_name='media_type'").Scan(&length); err != nil {
		return err
	}
	if length < 255 {
		_, err := db.ExecContext(ctx, "ALTER TABLE media_cache MODIFY COLUMN media_type VARCHAR(255) NOT NULL DEFAULT ''")
		return err
	}
	return nil
}
