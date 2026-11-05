package historydb

import (
	"fmt"
	"net/url"
	"time"
)

func (s *Store) Save(run Run, collectionName, collectionPath string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("history database is unavailable")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var collectionID any
	if collectionName != "" {
		folderPath := collectionPath
		if folderPath == "" {
			folderPath = collectionName
		}
		baseURL := ""
		if parsed, parseErr := url.Parse(run.URL); parseErr == nil && parsed.Host != "" {
			baseURL = parsed.Scheme + "://" + parsed.Host
		}
		_, err = tx.Exec(`INSERT INTO collections(name, folder_path, base_url) VALUES(?, ?, ?)
			ON CONFLICT(folder_path) DO UPDATE SET name=excluded.name, base_url=CASE WHEN excluded.base_url='' THEN collections.base_url ELSE excluded.base_url END`, collectionName, folderPath, baseURL)
		if err != nil {
			return fmt.Errorf("save history collection: %w", err)
		}
		var id int64
		if err := tx.QueryRow(`SELECT id FROM collections WHERE folder_path = ?`, folderPath).Scan(&id); err != nil {
			return fmt.Errorf("load history collection id: %w", err)
		}
		collectionID = id
	}
	if run.ExecutedAt.IsZero() {
		run.ExecutedAt = time.Now()
	}
	if run.TargetDomain == "" {
		if parsed, parseErr := url.Parse(run.URL); parseErr == nil {
			run.TargetDomain = parsed.Hostname()
		}
	}
	result, err := tx.Exec(`INSERT INTO history_runs(collection_id, request_name, target_domain, url, method, status_code, response_time, response_body, executed_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, collectionID, run.RequestName, run.TargetDomain, run.URL, run.Method, run.StatusCode, run.ResponseTime, run.ResponseBody, run.ExecutedAt.UTC())
	if err != nil {
		return fmt.Errorf("save request history: %w", err)
	}
	runID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get request history id: %w", err)
	}
	variables := ExtractVariables(run.ResponseBody, run.TargetDomain)
	for _, variable := range variables {
		if _, err := tx.Exec(`INSERT INTO captured_variables(run_id, domain_scope, var_name, var_value) VALUES(?, ?, ?, ?)`, runID, variable.DomainScope, variable.Name, variable.Value); err != nil {
			return fmt.Errorf("save captured response variable %q: %w", variable.Name, err)
		}
	}
	return tx.Commit()
}

// ExtractVariables flattens JSON scalar values into dot and array-index paths.
// Non-JSON bodies, nulls, and empty containers produce no captured variables.
