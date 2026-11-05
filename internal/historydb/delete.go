package historydb

import "fmt"

// Delete removes one history run and its captured variables.
func (s *Store) Delete(runID int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("history database is unavailable")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM captured_variables WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("delete captured variables: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM history_runs WHERE id = ?`, runID); err != nil {
		return fmt.Errorf("delete history run: %w", err)
	}
	return tx.Commit()
}
