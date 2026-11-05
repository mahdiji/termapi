package historydb

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestListForRequestFiltersMethodAndURL(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE collections (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			folder_path TEXT NOT NULL UNIQUE,
			base_url TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE history_runs (
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
		`CREATE TABLE captured_variables (
			id INTEGER PRIMARY KEY,
			run_id INTEGER NOT NULL REFERENCES history_runs(id) ON DELETE CASCADE,
			domain_scope TEXT NOT NULL,
			var_name TEXT NOT NULL,
			var_value TEXT NOT NULL
		)`,
		`INSERT INTO history_runs(id, request_name, target_domain, url, method, status_code, response_time, response_body, executed_at)
			VALUES(1, 'Users old', 'api.test', 'https://api.test/users', 'GET', 200, 10, '{"role":"user"}', '2026-01-01 00:00:00')`,
		`INSERT INTO history_runs(id, request_name, target_domain, url, method, status_code, response_time, response_body, executed_at)
			VALUES(2, 'Orders', 'api.test', 'https://api.test/orders', 'GET', 200, 10, '{}', '2026-01-02 00:00:00')`,
		`INSERT INTO history_runs(id, request_name, target_domain, url, method, status_code, response_time, response_body, executed_at)
			VALUES(3, 'Users new', 'api.test', 'https://api.test/users', 'GET', 200, 10, '{"role":"admin"}', '2026-01-03 00:00:00')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	store := &Store{db: db}
	runs, err := store.ListForRequest("GET", "https://api.test/users", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 matching runs, got %d", len(runs))
	}
	if runs[0].ID != 3 || runs[1].ID != 1 {
		t.Fatalf("expected newest first (3, 1), got %d, %d", runs[0].ID, runs[1].ID)
	}

	all, err := store.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 total runs, got %d", len(all))
	}
}
