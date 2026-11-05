package collectionstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func (s *Store) CreateFolder(parentRelPath, name, kind string) (Entry, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Entry{}, errors.New("folder name cannot be empty")
	}
	if kind != "collection" {
		kind = "folder"
	}

	parentPath, err := s.resolve(parentRelPath)
	if err != nil {
		return Entry{}, err
	}
	if err := os.MkdirAll(parentPath, 0o755); err != nil {
		return Entry{}, fmt.Errorf("create parent folder: %w", err)
	}
	baseName := slug(name)
	if baseName == "" {
		return Entry{}, errors.New("folder name has no usable characters")
	}

	folderName := baseName
	for suffix := 2; ; suffix++ {
		candidate := filepath.Join(parentPath, folderName)
		err := os.Mkdir(candidate, 0o755)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return Entry{}, fmt.Errorf("create folder: %w", err)
		}
		folderName = fmt.Sprintf("%s-%d", baseName, suffix)
	}

	folderPath := filepath.Join(parentPath, folderName)
	meta := FolderMeta{ID: folderName, Name: name, Type: kind}
	data, err := yaml.Marshal(meta)
	if err != nil {
		return Entry{}, fmt.Errorf("encode folder metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(folderPath, folderFile), data, 0o644); err != nil {
		_ = os.RemoveAll(folderPath)
		return Entry{}, fmt.Errorf("write folder metadata: %w", err)
	}
	relPath, err := filepath.Rel(s.Root, folderPath)
	if err != nil {
		return Entry{}, fmt.Errorf("make folder path relative: %w", err)
	}
	return Entry{
		ID:       folderName,
		Name:     name,
		RelPath:  filepath.ToSlash(relPath),
		IsFolder: true,
	}, nil
}

func (s *Store) CreateRequest(parentRelPath, name string) (Entry, error) {
	return s.CreateRequestWithMethod(parentRelPath, name, "GET")
}

func (s *Store) CreateRequestWithMethod(parentRelPath, name, method string) (Entry, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Entry{}, errors.New("request name cannot be empty")
	}
	parentPath, err := s.resolve(parentRelPath)
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(parentPath)
	if err != nil {
		return Entry{}, fmt.Errorf("open request folder: %w", err)
	}
	if !info.IsDir() {
		return Entry{}, errors.New("request parent is not a folder")
	}

	baseName := slug(name)
	if baseName == "" {
		return Entry{}, errors.New("request name has no usable characters")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	request := Request{ID: baseName, Name: name, Method: method}
	for suffix := 1; ; suffix++ {
		fileName := baseName + ".yaml"
		if suffix > 1 {
			fileName = fmt.Sprintf("%s-%d.yaml", baseName, suffix)
			request.ID = fmt.Sprintf("%s-%d", baseName, suffix)
		}
		filePath := filepath.Join(parentPath, fileName)
		data, err := yaml.Marshal(request)
		if err != nil {
			return Entry{}, fmt.Errorf("encode request: %w", err)
		}
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return Entry{}, fmt.Errorf("create request YAML: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			_ = os.Remove(filePath)
			return Entry{}, fmt.Errorf("write request YAML: %w", err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(filePath)
			return Entry{}, fmt.Errorf("close request YAML: %w", err)
		}
		relPath, err := filepath.Rel(s.Root, filePath)
		if err != nil {
			return Entry{}, fmt.Errorf("make request path relative: %w", err)
		}
		return Entry{
			ID:       request.ID,
			Name:     request.Name,
			RelPath:  filepath.ToSlash(relPath),
			Request:  &request,
			IsFolder: false,
		}, nil
	}
}

// SaveRequest creates a request file or updates the supplied existing request path.
