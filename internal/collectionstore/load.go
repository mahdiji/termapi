package collectionstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func (s *Store) Load() ([]Entry, error) {
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return nil, fmt.Errorf("create collections directory: %w", err)
	}
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return nil, fmt.Errorf("read collections directory: %w", err)
	}

	var result []Entry
	for _, item := range entries {
		if !item.IsDir() || strings.HasPrefix(item.Name(), ".") {
			continue
		}
		entry, err := s.loadFolder(filepath.Join(s.Root, item.Name()), "")
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, nil
}

func (s *Store) loadFolder(absPath, parentRel string) (Entry, error) {
	name := filepath.Base(absPath)
	relPath := filepath.Join(parentRel, name)
	meta := FolderMeta{ID: slug(name), Name: name, Type: "folder"}
	metaPath := filepath.Join(absPath, folderFile)
	if data, err := os.ReadFile(metaPath); err == nil {
		if err := yaml.Unmarshal(data, &meta); err != nil {
			return Entry{}, fmt.Errorf("parse %s: %w", metaPath, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Entry{}, fmt.Errorf("read %s: %w", metaPath, err)
	}
	if meta.Name == "" {
		meta.Name = name
	}
	if meta.ID == "" {
		meta.ID = slug(name)
	}

	entry := Entry{ID: meta.ID, Name: meta.Name, RelPath: filepath.ToSlash(relPath), IsFolder: true}
	children, err := os.ReadDir(absPath)
	if err != nil {
		return Entry{}, fmt.Errorf("read %s: %w", absPath, err)
	}
	for _, child := range children {
		childPath := filepath.Join(absPath, child.Name())
		if child.IsDir() {
			folder, err := s.loadFolder(childPath, relPath)
			if err != nil {
				return Entry{}, err
			}
			entry.Children = append(entry.Children, folder)
			continue
		}
		if child.Name() == folderFile || strings.HasPrefix(child.Name(), ".") || !strings.EqualFold(filepath.Ext(child.Name()), ".yaml") {
			continue
		}
		data, err := os.ReadFile(childPath)
		if err != nil {
			return Entry{}, fmt.Errorf("read %s: %w", childPath, err)
		}
		var request Request
		if err := yaml.Unmarshal(data, &request); err != nil {
			return Entry{}, fmt.Errorf("parse %s: %w", childPath, err)
		}
		if request.Name == "" {
			request.Name = strings.TrimSuffix(child.Name(), filepath.Ext(child.Name()))
		}
		if request.ID == "" {
			request.ID = slug(strings.TrimSuffix(child.Name(), filepath.Ext(child.Name())))
		}
		if request.Method == "" {
			request.Method = "GET"
		}
		request.Normalize()
		entry.Children = append(entry.Children, Entry{
			ID:       request.ID,
			Name:     request.Name,
			RelPath:  filepath.ToSlash(filepath.Join(relPath, child.Name())),
			IsFolder: false,
			Request:  &request,
		})
	}
	return entry, nil
}
