package rapidex

import (
    "errors"
    "fmt"
    "strings"
)

// ValidateMappingConfig performs basic validation on a MappingConfig.  It
// ensures mandatory fields are present and trims whitespace.  It also
// validates each mapping entry and checks for duplicate (field,dataElement)
// pairs which could lead to ambiguous data values.
func ValidateMappingConfig(cfg MappingConfig) error {
    if strings.TrimSpace(cfg.FlowUUID) == "" {
        return errors.New("flow_uuid is required")
    }
    if strings.TrimSpace(cfg.Dataset) == "" {
        return errors.New("dataset is required")
    }
    if strings.TrimSpace(cfg.OrgUnitVar) == "" {
        return errors.New("org_unit_var is required")
    }
    if strings.TrimSpace(cfg.PeriodVar) == "" {
        return errors.New("period_var is required")
    }
    if len(cfg.Mappings) == 0 {
        return errors.New("at least one mapping is required")
    }
    seen := make(map[string]bool)
    for i, m := range cfg.Mappings {
        field := strings.TrimSpace(m.Field)
        de := strings.TrimSpace(m.DataElement)
        if field == "" {
            return fmt.Errorf("mapping[%d] field is required", i)
        }
        if de == "" {
            return fmt.Errorf("mapping[%d] data_element is required", i)
        }
        // Trim combos
        cfg.Mappings[i].Field = field
        cfg.Mappings[i].DataElement = de
        cfg.Mappings[i].CategoryOptionCombo = strings.TrimSpace(m.CategoryOptionCombo)
        cfg.Mappings[i].AttributeOptionCombo = strings.TrimSpace(m.AttributeOptionCombo)
        key := field + ":" + de
        if seen[key] {
            return fmt.Errorf("duplicate mapping for field %s and data element %s", field, de)
        }
        seen[key] = true
    }
    return nil
}