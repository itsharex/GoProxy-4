package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaVersion = 1

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("数据库路径未设置")
	}
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE IF NOT EXISTS web_users (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			username         TEXT NOT NULL UNIQUE,
			password         TEXT NOT NULL,
			must_change_pwd  INTEGER NOT NULL DEFAULT 0,
			created_at       TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS web_settings (
			id               INTEGER PRIMARY KEY CHECK (id = 1),
			jwt_secret       TEXT NOT NULL DEFAULT '',
			jwt_expire_hours INTEGER NOT NULL DEFAULT 24,
			updated_at       TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS auth_users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT NOT NULL UNIQUE,
			password   TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS route_rule_sets (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			file_name   TEXT NOT NULL UNIQUE,
			version     INTEGER NOT NULL DEFAULT 1,
			description TEXT NOT NULL DEFAULT '',
			is_active   INTEGER NOT NULL DEFAULT 0,
			created_at  TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS route_rules (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			rule_set_id       INTEGER NOT NULL REFERENCES route_rule_sets(id) ON DELETE CASCADE,
			rule_id           TEXT NOT NULL,
			name              TEXT NOT NULL,
			enabled           INTEGER NOT NULL DEFAULT 1,
			priority          INTEGER NOT NULL,
			match_type        TEXT NOT NULL,
			protocols         TEXT NOT NULL DEFAULT '[]',
			targets           TEXT NOT NULL DEFAULT '[]',
			outbound_mode     TEXT NOT NULL DEFAULT 'default',
			outbound_local_ip TEXT NOT NULL DEFAULT '',
			outbound_interface TEXT NOT NULL DEFAULT '',
			remark            TEXT NOT NULL DEFAULT '',
			sort_order        INTEGER NOT NULL DEFAULT 0,
			UNIQUE(rule_set_id, rule_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("建表失败: %w", err)
	}

	var version int
	err = s.db.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key = 'schema_version'").Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("读取 schema 版本失败: %w", err)
	}

	if version == 0 {
		_, err = s.db.Exec("INSERT OR REPLACE INTO schema_meta (key, value) VALUES ('schema_version', ?)", schemaVersion)
		if err != nil {
			return fmt.Errorf("写入 schema 版本失败: %w", err)
		}
	}

	return nil
}

func (s *Store) IsFreshDB() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM schema_meta WHERE key = 'initialized'").Scan(&count)
	return err != nil || count == 0
}

func (s *Store) MarkInitialized() error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO schema_meta (key, value) VALUES ('initialized', 'true')")
	return err
}
