package request

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	MetadataColumnTypeString   = "string"
	MetadataColumnTypeNumber   = "number"
	MetadataColumnTypeBoolean  = "boolean"
	MetadataColumnTypeDateTime = "datetime"
)

type MetadataColumn struct {
	Key              string `json:"key"`
	Label            string `json:"label"`
	Type             string `json:"type"`
	Searchable       bool   `json:"searchable"`
	VisibleByDefault bool   `json:"visibleByDefault"`
}

func normalizeMetadataColumns(columns []MetadataColumn) []MetadataColumn {
	if len(columns) == 0 {
		return []MetadataColumn{}
	}
	normalized := make([]MetadataColumn, 0, len(columns))
	seen := map[string]struct{}{}
	for _, column := range columns {
		key := strings.TrimSpace(column.Key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		label := strings.TrimSpace(column.Label)
		if label == "" {
			label = key
		}
		columnType := strings.ToLower(strings.TrimSpace(column.Type))
		switch columnType {
		case MetadataColumnTypeString, MetadataColumnTypeNumber, MetadataColumnTypeBoolean, MetadataColumnTypeDateTime:
		default:
			columnType = MetadataColumnTypeString
		}
		normalized = append(normalized, MetadataColumn{
			Key:              key,
			Label:            label,
			Type:             columnType,
			Searchable:       column.Searchable,
			VisibleByDefault: column.VisibleByDefault,
		})
	}
	return normalized
}

func applyMetadataColumns(items []Record, columns []MetadataColumn) {
	normalized := normalizeMetadataColumns(columns)
	for index := range items {
		items[index].ProjectedMetadata = buildProjectedMetadata(items[index].Extras, normalized)
	}
}

func buildProjectedMetadata(extras map[string]any, columns []MetadataColumn) map[string]any {
	if len(columns) == 0 {
		return map[string]any{}
	}
	projected := make(map[string]any, len(columns))
	for _, column := range normalizeMetadataColumns(columns) {
		value, ok := extras[column.Key]
		if !ok {
			projected[column.Key] = nil
			continue
		}
		projected[column.Key] = coerceMetadataValue(value, column.Type)
	}
	return projected
}

func searchableMetadataColumns(columns []MetadataColumn) []MetadataColumn {
	searchable := make([]MetadataColumn, 0, len(columns))
	for _, column := range normalizeMetadataColumns(columns) {
		if column.Searchable {
			searchable = append(searchable, column)
		}
	}
	return searchable
}

func coerceMetadataValue(value any, columnType string) any {
	switch columnType {
	case MetadataColumnTypeNumber:
		return coerceMetadataNumber(value)
	case MetadataColumnTypeBoolean:
		return coerceMetadataBoolean(value)
	case MetadataColumnTypeDateTime:
		return coerceMetadataDateTime(value)
	default:
		return coerceMetadataString(value)
	}
}

func coerceMetadataString(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return typed
	case json.Number:
		return typed.String()
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func coerceMetadataNumber(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return integer
		}
		if floatValue, err := typed.Float64(); err == nil {
			return floatValue
		}
	case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return typed
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		if integer, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return integer
		}
		if floatValue, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return floatValue
		}
	}
	return coerceMetadataString(value)
}

func coerceMetadataBoolean(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		return typed
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		if trimmed == "" {
			return nil
		}
		if parsed, err := strconv.ParseBool(trimmed); err == nil {
			return parsed
		}
	}
	return coerceMetadataString(value)
}

func coerceMetadataDateTime(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
		return trimmed
	default:
		return coerceMetadataString(value)
	}
}
