package collectionstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func (s *Store) SaveRequest(parentRelPath, existingRelPath string, request Request) (Entry, error) {
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		return Entry{}, errors.New("request name cannot be empty")
	}
	request.Method = strings.ToUpper(strings.TrimSpace(request.Method))
	if request.Method == "" {
		request.Method = "GET"
	}
	if request.ID == "" {
		request.ID = slug(request.Name)
	}
	request.Normalize()
	// Persist only the structured fields; loading still accepts the legacy ones.
	request.Params = ""
	request.Headers = nil
	request.Body = ""
	data, err := yaml.Marshal(request)
	if err != nil {
		return Entry{}, fmt.Errorf("encode request: %w", err)
	}
	var filePath string
	if existingRelPath != "" {
		filePath, err = s.resolve(existingRelPath)
		if err != nil {
			return Entry{}, err
		}
		info, statErr := os.Stat(filePath)
		if statErr != nil {
			return Entry{}, fmt.Errorf("open saved request: %w", statErr)
		}
		if info.IsDir() {
			return Entry{}, errors.New("request path points to a folder")
		}
	} else {
		parent, resolveErr := s.resolve(parentRelPath)
		if resolveErr != nil {
			return Entry{}, resolveErr
		}
		info, statErr := os.Stat(parent)
		if statErr != nil || !info.IsDir() {
			return Entry{}, errors.New("request parent folder does not exist")
		}
		base := slug(request.Name)
		if base == "" {
			return Entry{}, errors.New("request name has no usable characters")
		}
		for suffix := 1; ; suffix++ {
			name := base + ".yaml"
			if suffix > 1 {
				name = fmt.Sprintf("%s-%d.yaml", base, suffix)
				request.ID = fmt.Sprintf("%s-%d", base, suffix)
				data, err = yaml.Marshal(request)
				if err != nil {
					return Entry{}, err
				}
			}
			candidate := filepath.Join(parent, name)
			if _, statErr := os.Stat(candidate); errors.Is(statErr, os.ErrNotExist) {
				filePath = candidate
				break
			} else if statErr != nil {
				return Entry{}, fmt.Errorf("check request path: %w", statErr)
			}
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(filePath), ".termapi-request-*.yaml")
	if err != nil {
		return Entry{}, fmt.Errorf("create temporary request: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return Entry{}, fmt.Errorf("set request permissions: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return Entry{}, fmt.Errorf("write request: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return Entry{}, fmt.Errorf("close request: %w", err)
	}
	if err = os.Rename(tmpName, filePath); err != nil {
		return Entry{}, fmt.Errorf("save request: %w", err)
	}
	relPath, err := filepath.Rel(s.Root, filePath)
	if err != nil {
		return Entry{}, fmt.Errorf("make request path relative: %w", err)
	}
	return Entry{ID: request.ID, Name: request.Name, RelPath: filepath.ToSlash(relPath), Request: &request}, nil
}
