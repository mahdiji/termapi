package historydb

import (
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func projectRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := current; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return strings.TrimSpace(current), nil
}
