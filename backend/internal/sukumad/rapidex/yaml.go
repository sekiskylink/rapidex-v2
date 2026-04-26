package rapidex

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// ParseMappingConfigsYAML parses one or more YAML documents into mapping configs.
func ParseMappingConfigsYAML(raw string) ([]MappingConfig, error) {
	decoder := yaml.NewDecoder(strings.NewReader(raw))
	configs := []MappingConfig{}
	for {
		var cfg MappingConfig
		err := decoder.Decode(&cfg)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode yaml: %w", err)
		}
		if strings.TrimSpace(cfg.FlowUUID) == "" &&
			strings.TrimSpace(cfg.FlowName) == "" &&
			strings.TrimSpace(cfg.Dataset) == "" &&
			len(cfg.Mappings) == 0 {
			continue
		}
		configs = append(configs, cfg)
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("no mapping documents found")
	}
	return configs, nil
}

// MarshalMappingConfigsYAML emits one YAML document per mapping config.
func MarshalMappingConfigsYAML(configs []MappingConfig) (string, error) {
	var buffer bytes.Buffer
	for i, cfg := range configs {
		if i > 0 {
			buffer.WriteString("---\n")
		}
		raw, err := yaml.Marshal(cfg)
		if err != nil {
			return "", fmt.Errorf("marshal yaml: %w", err)
		}
		buffer.Write(raw)
	}
	return buffer.String(), nil
}
