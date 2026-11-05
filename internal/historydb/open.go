package historydb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func OpenProject() (*Store, error) {
	root, err := projectRoot()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, ".termapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}
	db, err := sql.Open("sqlite3", filepath.Join(dir, "history.db"))
	if err != nil {
		return nil, fmt.Errorf("open history database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable history foreign keys: %w", err)
	}
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS collections (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			folder_path TEXT NOT NULL UNIQUE,
			base_url TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS history_runs (
			id INTEGER PRIMARY KEY,
			collection_id INTEGER REFERENCES collections(id) ON DELETE SET NULL,
			request_name TEXT NOT NULL,
			target_domain TEXT NOT NULL,
			url TEXT NOT NULL,
			method TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			response_time INTEGER NOT NULL,
			response_body TEXT NOT NULL,
			executed_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_runs_executed_at ON history_runs(executed_at DESC)`,
		`CREATE TABLE IF NOT EXISTS captured_variables (
			id INTEGER PRIMARY KEY,
			run_id INTEGER NOT NULL REFERENCES history_runs(id) ON DELETE CASCADE,
			domain_scope TEXT NOT NULL,
			var_name TEXT NOT NULL,
			var_value TEXT NOT NULL
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize history schema: %w", err)
		}
	}
	return &Store{db: db}, nil
}

