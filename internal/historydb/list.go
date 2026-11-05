package historydb

import (
	"fmt"
	"time"
)

func (s *Store) List(limit int) ([]Run, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("history database is unavailable")
	}
	limit = normalizeLimit(limit)
	return s.list(`SELECT h.id, COALESCE(c.name, ''), h.request_name, h.target_domain, h.url, h.method, h.status_code, h.response_time, h.response_body, h.executed_at
		FROM history_runs h LEFT JOIN collections c ON c.id = h.collection_id
		ORDER BY h.executed_at DESC, h.id DESC LIMIT ?`, limit)
}

// ListForRequest mengembalikan history dengan method dan URL yang sama,
// terbaru lebih dulu. Dipakai Diff Viewer agar baseline tetap relevan
// walaupun sudah banyak request lain yang dijalankan setelahnya.
func (s *Store) ListForRequest(method, url string, limit int) ([]Run, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("history database is unavailable")
	}
	if method == "" || url == "" {
		return nil, nil
	}
	limit = normalizeLimit(limit)
	return s.list(`SELECT h.id, COALESCE(c.name, ''), h.request_name, h.target_domain, h.url, h.method, h.status_code, h.response_time, h.response_body, h.executed_at
		FROM history_runs h LEFT JOIN collections c ON c.id = h.collection_id
		WHERE h.method = ? AND h.url = ?
		ORDER BY h.executed_at DESC, h.id DESC LIMIT ?`, method, url, limit)
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 1000 {
		return 500
	}
	return limit
}

func (s *Store) list(query string, args ...any) ([]Run, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("read request history: %w", err)
	}
	defer rows.Close()
	var runs []Run
	for rows.Next() {
		var run Run
		var executed string
		if err := rows.Scan(&run.ID, &run.Collection, &run.RequestName, &run.TargetDomain, &run.URL, &run.Method, &run.StatusCode, &run.ResponseTime, &run.ResponseBody, &executed); err != nil {
			return nil, fmt.Errorf("read history row: %w", err)
		}
		run.ExecutedAt, _ = time.Parse(time.RFC3339Nano, executed)
		if run.ExecutedAt.IsZero() {
			run.ExecutedAt, _ = time.Parse("2006-01-02 15:04:05-07:00", executed)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range runs {
		variables, err := s.listVariables(runs[i].ID)
		if err != nil {
			return nil, err
		}
		runs[i].CapturedVariables = variables
	}
	return runs, nil
}

func (s *Store) listVariables(runID int64) ([]CapturedVariable, error) {
	rows, err := s.db.Query(`SELECT domain_scope, var_name, var_value FROM captured_variables WHERE run_id = ? ORDER BY id`, runID)
	if err != nil {
		return nil, fmt.Errorf("read captured variables: %w", err)
	}
	defer rows.Close()
	var variables []CapturedVariable
	for rows.Next() {
		var variable CapturedVariable
		if err := rows.Scan(&variable.DomainScope, &variable.Name, &variable.Value); err != nil {
			return nil, err
		}
		variables = append(variables, variable)
	}
	return variables, rows.Err()
}
