package envstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// EnvFile represents a named environment with key-value variables.
type EnvFile struct {
	Name      string            `yaml:"name"`
	Variables map[string]string `yaml:"variables"`
}

// Store manages .termapi/envs/ directory.
type Store struct {
	dir string // absolute path to .termapi/envs/
}

// New creates a new Store rooted at projectRoot.
// New(".") → .termapi/envs/
func New(projectRoot string) *Store {
	dir := filepath.Join(projectRoot, ".termapi", "envs")
	return &Store{dir: dir}
}

func (s *Store) ensureDir() error {
	return os.MkdirAll(s.dir, 0o755)
}

func (s *Store) filePath(name string) string {
	return filepath.Join(s.dir, name+".yaml")
}

// List returns the names of all environments (file names without .yaml), sorted.
func (s *Store) List() ([]string, error) {
	if err := s.ensureDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		}
	}
	sort.Strings(names)
	return names, nil
}

// Load reads a single environment by name.
func (s *Store) Load(name string) (*EnvFile, error) {
	data, err := os.ReadFile(s.filePath(name))
	if err != nil {
		return nil, fmt.Errorf("load env %q: %w", name, err)
	}
	var ef EnvFile
	if err := yaml.Unmarshal(data, &ef); err != nil {
		return nil, fmt.Errorf("parse env %q: %w", name, err)
	}
	if ef.Variables == nil {
		ef.Variables = make(map[string]string)
	}
	return &ef, nil
}

// Save persists an environment to disk.
func (s *Store) Save(env *EnvFile) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal env %q: %w", env.Name, err)
	}
	if err := os.WriteFile(s.filePath(env.Name), data, 0o644); err != nil {
		return fmt.Errorf("write env %q: %w", env.Name, err)
	}
	return nil
}

// Delete removes an environment file.
func (s *Store) Delete(name string) error {
	err := os.Remove(s.filePath(name))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete env %q: %w", name, err)
	}
	return nil
}

// Create creates a new empty environment and saves it.
func (s *Store) Create(name string) (*EnvFile, error) {
	ef := &EnvFile{
		Name:      name,
		Variables: make(map[string]string),
	}
	if err := s.Save(ef); err != nil {
		return nil, err
	}
	return ef, nil
}
