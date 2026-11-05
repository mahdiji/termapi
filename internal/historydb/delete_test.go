package historydb

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDeleteRunRemovesCapturedVariables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE history_runs (
			id INTEGER PRIMARY KEY,
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
			VALUES(7, 'Login', 'example.com', 'https://example.com/login', 'POST', 200, 12, '{}', '2026-01-01 00:00:00')`,
		`INSERT INTO captured_variables(run_id, domain_scope, var_name, var_value)
			VALUES(7, 'example.com', 'token', 'abc')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	store := &Store{db: db}
	if err := store.Delete(7); err != nil {
		t.Fatal(err)
	}

	var runs, vars int
	if err := db.QueryRow(`SELECT COUNT(*) FROM history_runs`).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM captured_variables`).Scan(&vars); err != nil {
		t.Fatal(err)
	}
	if runs != 0 || vars != 0 {
		t.Fatalf("expected run and variables deleted, got runs=%d vars=%d", runs, vars)
	}
}
