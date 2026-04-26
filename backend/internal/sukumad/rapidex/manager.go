package rapidex

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v3"
)

// Manager manages mapping configurations loaded from a directory.  It maps
// RapidPro flow UUIDs to their corresponding MappingConfig.  It can be
// reloaded on demand but does not watch for changes automatically.
type Manager struct {
	dir      string
	mu       sync.RWMutex
	mappings map[string]MappingConfig
}

// NewManager reads all YAML files in dir and returns a Manager.  Each file
// must contain a single MappingConfig document.  Files that do not parse
// correctly are skipped and reported via error.  The returned manager
// may be reloaded by calling Reload().
func NewManager(dir string) (*Manager, error) {
	m := &Manager{dir: dir, mappings: make(map[string]MappingConfig)}
	if err := m.Reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// Reload clears the current mappings and re-parses all YAML files in the
// manager’s directory.  If any file fails to parse or validate, an error
// will be returned and the previously loaded mappings will remain intact.
func (m *Manager) Reload() error {
	files, err := ioutil.ReadDir(m.dir)
	if err != nil {
		return fmt.Errorf("failed to read mappings directory: %w", err)
	}
	newMappings := make(map[string]MappingConfig)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".yaml") && !strings.HasSuffix(strings.ToLower(file.Name()), ".yml") {
			continue
		}
		path := filepath.Join(m.dir, file.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read mapping file %s: %w", path, err)
		}
		var cfg MappingConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse mapping file %s: %w", path, err)
		}
		if err := ValidateMappingConfig(cfg); err != nil {
			return fmt.Errorf("invalid mapping file %s: %w", path, err)
		}
		newMappings[strings.TrimSpace(cfg.FlowUUID)] = cfg
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mappings = newMappings
	return nil
}

// Get returns the mapping for a given flow UUID.  The second return value is
// false when no mapping is found.
func (m *Manager) Get(flowUUID string) (MappingConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.mappings[strings.TrimSpace(flowUUID)]
	return cfg, ok
}

func (m *Manager) GetByFlowUUID(_ context.Context, flowUUID string) (MappingConfig, bool, error) {
	cfg, ok := m.Get(flowUUID)
	return cfg, ok, nil
}
