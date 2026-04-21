package rapidex

import (
    "fmt"
    "strings"
)

// extractField looks up a variable from the RapidPro webhook payload.  It
// searches in results, then fields, then contact.fields and returns the
// value as a string.  If not found, the empty string is returned.
func extractField(webhook RapidProWebhook, key string) string {
    lower := strings.ToLower(key)
    // search results
    if v, ok := webhook.Results[lower]; ok {
        return fmt.Sprintf("%v", v)
    }
    // search fields
    if v, ok := webhook.Fields[lower]; ok {
        return fmt.Sprintf("%v", v)
    }
    // search contact fields
    if v, ok := webhook.Contact.Fields[lower]; ok {
        return fmt.Sprintf("%v", v)
    }
    return ""
}

// MapToAggregate converts a RapidProWebhook event into an AggregatePayload
// according to a MappingConfig.  The returned payload may be incomplete if
// mandatory variables are missing; callers should validate the result.
func MapToAggregate(event RapidProWebhook, cfg MappingConfig) (AggregatePayload, error) {
    payload := AggregatePayload{
        DataSet:              cfg.Dataset,
        OrgUnit:              extractField(event, cfg.OrgUnitVar),
        Period:               extractField(event, cfg.PeriodVar),
        AttributeOptionCombo: strings.TrimSpace(cfg.PayloadAOC),
    }
    dataValues := make([]DataValue, 0, len(cfg.Mappings))
    for _, m := range cfg.Mappings {
        val := extractField(event, m.Field)
        if strings.TrimSpace(val) == "" {
            continue
        }
        dv := DataValue{
            DataElement:          m.DataElement,
            CategoryOptionCombo:  strings.TrimSpace(m.CategoryOptionCombo),
            AttributeOptionCombo: strings.TrimSpace(m.AttributeOptionCombo),
            Value:                val,
        }
        // Inherit attribute option combo from payload if not specified
        if dv.AttributeOptionCombo == "" {
            dv.AttributeOptionCombo = payload.AttributeOptionCombo
        }
        dataValues = append(dataValues, dv)
    }
    payload.DataValues = dataValues
    return payload, nil
}