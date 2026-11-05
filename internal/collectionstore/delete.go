package collectionstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func (s *Store) Delete(relPath string, isFolder bool) error {
	path, err := s.resolve(relPath)
	if err != nil {
		return err
	}
	if relPath == "" || filepath.Clean(path) == filepath.Clean(s.Root) {
		return errors.New("cannot delete collections root")
	}
	trashDir := filepath.Join(filepath.Dir(s.Root), ".termapi", "trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return fmt.Errorf("create recovery folder: %w", err)
	}
	name := filepath.Base(path)
	destination := filepath.Join(trashDir, time.Now().Format("20060102-150405.000000000")+"-"+name)
	if err := os.Rename(path, destination); err != nil {
		return fmt.Errorf("move selected item to recovery folder: %w", err)
	}
	return nil
}

func (s *Store) resolve(relPath string) (string, error) {
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	path := root
	if relPath != "" {
		path = filepath.Join(root, filepath.Clean(filepath.FromSlash(relPath)))
	}
	clean, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("folder path escapes collections directory")
	}
	return clean, nil
}

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugSeparators.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
