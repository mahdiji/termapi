package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	Scope string `yaml:"scope"`
	// ResponseRatio adalah proporsi tinggi area response terhadap tinggi area
	// konten (0..1). Nilai 0 berarti belum di-set -> pakai default.
	ResponseRatio float64 `yaml:"response_ratio,omitempty"`
	// Pinned menyimpan path request (relatif terhadap folder collections)
	// yang dipin ke bagian atas sidebar.
	Pinned []string `yaml:"pinned,omitempty"`
}

// DefaultResponseRatio memberi response ruang semaksimal mungkin; layout
// tetap menjaga tinggi minimum editor request.
const DefaultResponseRatio = 0.72

// LoadGlobalScope reads .termapi/global.yaml (relative to the project root) and returns the scope string.
// If the file does not exist or cannot be parsed, an empty string is returned.
func LoadGlobalScope(projectRoot string) (string, error) {
	cfgPath := filepath.Join(projectRoot, ".termapi", "global.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.Scope, nil
}

// SaveGlobalScope writes the scope string to .termapi/global.yaml (relative to project root).
func SaveGlobalScope(projectRoot string, scope string) error {
	cfg := loadGlobalConfig(projectRoot)
	cfg.Scope = scope
	return writeGlobalConfig(projectRoot, cfg)
}

// LoadResponseRatio reads the persisted response-area ratio from .termapi/global.yaml.
// Returns DefaultResponseRatio when unset or invalid.
func LoadResponseRatio(projectRoot string) float64 {
	cfg := loadGlobalConfig(projectRoot)
	if cfg.ResponseRatio <= 0.1 || cfg.ResponseRatio >= 0.9 {
		return DefaultResponseRatio
	}
	return cfg.ResponseRatio
}

// LoadPinned membaca daftar request yang dipin dari .termapi/global.yaml.
func LoadPinned(projectRoot string) []string {
	return loadGlobalConfig(projectRoot).Pinned
}

// SavePinned menyimpan daftar request yang dipin tanpa mengubah field lain.
func SavePinned(projectRoot string, pinned []string) error {
	cfg := loadGlobalConfig(projectRoot)
	cfg.Pinned = pinned
	return writeGlobalConfig(projectRoot, cfg)
}

// SaveResponseRatio persists the response-area ratio, preserving the scope field.
func SaveResponseRatio(projectRoot string, ratio float64) error {
	if ratio <= 0.1 || ratio >= 0.9 {
		ratio = DefaultResponseRatio
	}
	cfg := loadGlobalConfig(projectRoot)
	cfg.ResponseRatio = ratio
	return writeGlobalConfig(projectRoot, cfg)
}

func loadGlobalConfig(projectRoot string) GlobalConfig {
	cfgPath := filepath.Join(projectRoot, ".termapi", "global.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return GlobalConfig{}
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return GlobalConfig{}
	}
	return cfg
}

func writeGlobalConfig(projectRoot string, cfg GlobalConfig) error {
	cfgPath := filepath.Join(projectRoot, ".termapi", "global.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}
