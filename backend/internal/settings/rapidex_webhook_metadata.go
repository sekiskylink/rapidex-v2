package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/rapidex"
	dhis2metadata "basepro/backend/internal/sukumad/rapidex/dhis2metadata"
	"basepro/backend/internal/sukumad/rapidex/rapidpro"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

const (
	rapidexWebhookMetadataCategory = "rapidex"
	rapidexWebhookMetadataKey      = "webhook_metadata"
	defaultDHIS2ServerCode         = "dhis2"
)

type rapidexMetadataServerCatalog interface {
	ListServers(context.Context, sukumadserver.ListQuery) (sukumadserver.ListResult, error)
	GetServerByCode(context.Context, string) (sukumadserver.Record, error)
}

type rapidexRapidProMetadataClient interface {
	ListFlows(context.Context, rapidpro.Connection) ([]rapidpro.Flow, error)
	ListContactFields(context.Context, rapidpro.Connection) ([]rapidpro.ContactField, error)
}

type rapidexDHIS2MetadataClient interface {
	ListDataSets(context.Context, dhis2metadata.Connection) ([]dhis2metadata.DataSet, error)
	ListDataElements(context.Context, dhis2metadata.Connection) ([]dhis2metadata.DataElement, error)
	ListCategoryOptionCombos(context.Context, dhis2metadata.Connection) ([]dhis2metadata.CategoryOptionCombo, error)
	ListAttributeOptionCombos(context.Context, dhis2metadata.Connection) ([]dhis2metadata.AttributeOptionCombo, error)
}

type RapidexIntegrationServerOption struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	SystemType string `json:"systemType"`
	Suspended  bool   `json:"suspended"`
}

type RapidexRapidProFlowResultOption struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Categories []string `json:"categories"`
}

type RapidexRapidProFlowOption struct {
	UUID       string                            `json:"uuid"`
	Name       string                            `json:"name"`
	Type       string                            `json:"type,omitempty"`
	Archived   bool                              `json:"archived"`
	ParentRefs []string                          `json:"parentRefs,omitempty"`
	ModifiedOn string                            `json:"modifiedOn,omitempty"`
	Results    []RapidexRapidProFlowResultOption `json:"results"`
}

type RapidexDhis2DataElementRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RapidexDhis2DatasetOption struct {
	ID           string                       `json:"id"`
	Name         string                       `json:"name"`
	PeriodType   string                       `json:"periodType,omitempty"`
	DataElements []RapidexDhis2DataElementRef `json:"dataElements"`
}

type RapidexDhis2DataElementOption struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ValueType string `json:"valueType,omitempty"`
}

type RapidexDhis2CategoryOptionComboOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RapidexDhis2AttributeOptionComboOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RapidexWebhookMetadataSnapshot struct {
	RapidProServerCode         string                                   `json:"rapidProServerCode"`
	Dhis2ServerCode            string                                   `json:"dhis2ServerCode"`
	LastRefreshedAt            *time.Time                               `json:"lastRefreshedAt,omitempty"`
	RapidProFlows              []RapidexRapidProFlowOption              `json:"rapidProFlows"`
	RapidProContactFields      []rapidpro.ContactField                  `json:"rapidProContactFields"`
	Dhis2Datasets              []RapidexDhis2DatasetOption              `json:"dhis2Datasets"`
	Dhis2DataElements          []RapidexDhis2DataElementOption          `json:"dhis2DataElements"`
	Dhis2CategoryOptionCombos  []RapidexDhis2CategoryOptionComboOption  `json:"dhis2CategoryOptionCombos"`
	Dhis2AttributeOptionCombos []RapidexDhis2AttributeOptionComboOption `json:"dhis2AttributeOptionCombos"`
}

type RapidexWebhookMetadataResponse struct {
	RapidProServerCode string                           `json:"rapidProServerCode"`
	Dhis2ServerCode    string                           `json:"dhis2ServerCode"`
	RapidProServers    []RapidexIntegrationServerOption `json:"rapidProServers"`
	Dhis2Servers       []RapidexIntegrationServerOption `json:"dhis2Servers"`
	Snapshot           RapidexWebhookMetadataSnapshot   `json:"snapshot"`
	Warnings           []string                         `json:"warnings"`
}

type RapidexWebhookMetadataRefreshInput struct {
	RapidProServerCode string `json:"rapidProServerCode"`
	Dhis2ServerCode    string `json:"dhis2ServerCode"`
}

type rapidexWebhookMetadataStored struct {
	Snapshot RapidexWebhookMetadataSnapshot `json:"snapshot"`
}

func (s *Service) GetRapidexWebhookMetadata(ctx context.Context) (RapidexWebhookMetadataResponse, error) {
	if s.rapidexMetadataServerCatalog == nil {
		return RapidexWebhookMetadataResponse{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidex": []string{"RapidEx metadata integration is not configured"},
		})
	}
	mappings, err := s.getRapidexWebhookMappingsStored(ctx)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	stored, err := s.getRapidexWebhookMetadataStored(ctx)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	rapidProServers, dhis2Servers, err := s.listRapidexMetadataServers(ctx)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	snapshot := normalizeRapidexWebhookMetadataSnapshot(stored.Snapshot)
	rapidProCode := normalizeRapidProServerCode(mappings.RapidProServerCode)
	dhis2Code := normalizeRapidexDHIS2ServerCode(mappings.Dhis2ServerCode)
	if snapshot.RapidProServerCode != "" {
		rapidProCode = snapshot.RapidProServerCode
	}
	if snapshot.Dhis2ServerCode != "" {
		dhis2Code = snapshot.Dhis2ServerCode
	}
	snapshot.RapidProServerCode = rapidProCode
	snapshot.Dhis2ServerCode = dhis2Code
	return RapidexWebhookMetadataResponse{
		RapidProServerCode: rapidProCode,
		Dhis2ServerCode:    dhis2Code,
		RapidProServers:    rapidProServers,
		Dhis2Servers:       dhis2Servers,
		Snapshot:           snapshot,
		Warnings:           buildRapidexMetadataWarnings(mappings.Mappings, snapshot),
	}, nil
}

func (s *Service) RefreshRapidexWebhookMetadata(ctx context.Context, input RapidexWebhookMetadataRefreshInput, actorUserID *int64) (RapidexWebhookMetadataResponse, error) {
	if s.rapidexMetadataServerCatalog == nil || s.rapidexRapidProClient == nil || s.rapidexDHIS2Client == nil {
		return RapidexWebhookMetadataResponse{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"rapidex": []string{"RapidEx metadata integration is not configured"},
		})
	}
	storedMappings, err := s.getRapidexWebhookMappingsStored(ctx)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	rapidProCode := normalizeRapidProServerCode(input.RapidProServerCode)
	if rapidProCode == "" {
		rapidProCode = normalizeRapidProServerCode(storedMappings.RapidProServerCode)
	}
	dhis2Code := normalizeRapidexDHIS2ServerCode(input.Dhis2ServerCode)
	if dhis2Code == "" {
		dhis2Code = normalizeRapidexDHIS2ServerCode(storedMappings.Dhis2ServerCode)
	}
	rapidProRecord, err := s.rapidexMetadataServerCatalog.GetServerByCode(ctx, rapidProCode)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	dhis2Record, err := s.rapidexMetadataServerCatalog.GetServerByCode(ctx, dhis2Code)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	if rapidProRecord.Suspended {
		return RapidexWebhookMetadataResponse{}, apperror.ValidationWithDetails("validation failed", map[string]any{"rapidProServerCode": []string{"selected RapidPro server is suspended"}})
	}
	if dhis2Record.Suspended {
		return RapidexWebhookMetadataResponse{}, apperror.ValidationWithDetails("validation failed", map[string]any{"dhis2ServerCode": []string{"selected DHIS2 server is suspended"}})
	}
	flows, err := s.rapidexRapidProClient.ListFlows(ctx, rapidpro.Connection{BaseURL: rapidProRecord.BaseURL, Headers: rapidProRecord.Headers})
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	fields, err := s.rapidexRapidProClient.ListContactFields(ctx, rapidpro.Connection{BaseURL: rapidProRecord.BaseURL, Headers: rapidProRecord.Headers})
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	dataSets, err := s.rapidexDHIS2Client.ListDataSets(ctx, dhis2metadata.Connection{BaseURL: dhis2Record.BaseURL, Headers: dhis2Record.Headers})
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	dataElements, err := s.rapidexDHIS2Client.ListDataElements(ctx, dhis2metadata.Connection{BaseURL: dhis2Record.BaseURL, Headers: dhis2Record.Headers})
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	cocs, err := s.rapidexDHIS2Client.ListCategoryOptionCombos(ctx, dhis2metadata.Connection{BaseURL: dhis2Record.BaseURL, Headers: dhis2Record.Headers})
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	aocs, err := s.rapidexDHIS2Client.ListAttributeOptionCombos(ctx, dhis2metadata.Connection{BaseURL: dhis2Record.BaseURL, Headers: dhis2Record.Headers})
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	now := time.Now().UTC()
	snapshot := normalizeRapidexWebhookMetadataSnapshot(RapidexWebhookMetadataSnapshot{
		RapidProServerCode:         rapidProCode,
		Dhis2ServerCode:            dhis2Code,
		LastRefreshedAt:            &now,
		RapidProFlows:              normalizeRapidexRapidProFlows(flows),
		RapidProContactFields:      normalizeRapidProFields(fields),
		Dhis2Datasets:              normalizeRapidexDhis2Datasets(dataSets),
		Dhis2DataElements:          normalizeRapidexDhis2DataElements(dataElements),
		Dhis2CategoryOptionCombos:  normalizeRapidexDhis2COCs(cocs),
		Dhis2AttributeOptionCombos: normalizeRapidexDhis2AOCs(aocs),
	})
	if err := s.saveRapidexWebhookMetadataStored(ctx, rapidexWebhookMetadataStored{Snapshot: snapshot}, actorUserID); err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	storedMappings.RapidProServerCode = rapidProCode
	storedMappings.Dhis2ServerCode = dhis2Code
	if err := s.saveRapidexWebhookMappingsStored(ctx, storedMappings, actorUserID); err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	rapidProServers, dhis2Servers, err := s.listRapidexMetadataServers(ctx)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	s.logAudit(ctx, audit.Event{
		Action:      "settings.rapidex_webhook_metadata.refresh",
		ActorUserID: actorUserID,
		EntityType:  "settings",
		EntityID:    strPtr("rapidex.webhook_metadata"),
		Metadata: map[string]any{
			"rapidProServerCode": rapidProCode,
			"dhis2ServerCode":    dhis2Code,
			"flowCount":          len(snapshot.RapidProFlows),
			"datasetCount":       len(snapshot.Dhis2Datasets),
		},
	})
	return RapidexWebhookMetadataResponse{
		RapidProServerCode: rapidProCode,
		Dhis2ServerCode:    dhis2Code,
		RapidProServers:    rapidProServers,
		Dhis2Servers:       dhis2Servers,
		Snapshot:           snapshot,
		Warnings:           buildRapidexMetadataWarnings(storedMappings.Mappings, snapshot),
	}, nil
}

func (s *Service) getRapidexWebhookMetadataStored(ctx context.Context) (rapidexWebhookMetadataStored, error) {
	raw, err := s.repo.Get(ctx, rapidexWebhookMetadataCategory, rapidexWebhookMetadataKey)
	if err != nil {
		if err == ErrNotFound {
			return rapidexWebhookMetadataStored{}, nil
		}
		return rapidexWebhookMetadataStored{}, err
	}
	var stored rapidexWebhookMetadataStored
	if unmarshalErr := json.Unmarshal(raw, &stored); unmarshalErr != nil {
		return rapidexWebhookMetadataStored{}, nil
	}
	stored.Snapshot = normalizeRapidexWebhookMetadataSnapshot(stored.Snapshot)
	return stored, nil
}

func (s *Service) saveRapidexWebhookMetadataStored(ctx context.Context, stored rapidexWebhookMetadataStored, actorUserID *int64) error {
	payload, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal rapidex webhook metadata setting: %w", err)
	}
	return s.repo.Upsert(ctx, rapidexWebhookMetadataCategory, rapidexWebhookMetadataKey, payload, actorUserID, time.Now().UTC())
}

func (s *Service) listRapidexMetadataServers(ctx context.Context) ([]RapidexIntegrationServerOption, []RapidexIntegrationServerOption, error) {
	list, err := s.rapidexMetadataServerCatalog.ListServers(ctx, sukumadserver.ListQuery{Page: 1, PageSize: 200})
	if err != nil {
		return nil, nil, err
	}
	rapidProServers := make([]RapidexIntegrationServerOption, 0)
	dhis2Servers := make([]RapidexIntegrationServerOption, 0)
	for _, item := range list.Items {
		option := RapidexIntegrationServerOption{Code: strings.TrimSpace(item.Code), Name: strings.TrimSpace(item.Name), SystemType: strings.TrimSpace(item.SystemType), Suspended: item.Suspended}
		switch strings.ToLower(strings.TrimSpace(item.SystemType)) {
		case "rapidpro":
			rapidProServers = append(rapidProServers, option)
		case "dhis2":
			dhis2Servers = append(dhis2Servers, option)
		}
	}
	slices.SortFunc(rapidProServers, func(left, right RapidexIntegrationServerOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	slices.SortFunc(dhis2Servers, func(left, right RapidexIntegrationServerOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return rapidProServers, dhis2Servers, nil
}

func normalizeRapidexWebhookMetadataSnapshot(input RapidexWebhookMetadataSnapshot) RapidexWebhookMetadataSnapshot {
	return RapidexWebhookMetadataSnapshot{
		RapidProServerCode:         normalizeRapidProServerCode(input.RapidProServerCode),
		Dhis2ServerCode:            normalizeRapidexDHIS2ServerCode(input.Dhis2ServerCode),
		LastRefreshedAt:            input.LastRefreshedAt,
		RapidProFlows:              normalizeRapidexRapidProFlowsOptions(input.RapidProFlows),
		RapidProContactFields:      normalizeRapidProFields(input.RapidProContactFields),
		Dhis2Datasets:              normalizeRapidexDhis2DatasetOptions(input.Dhis2Datasets),
		Dhis2DataElements:          normalizeRapidexDhis2DataElementOptions(input.Dhis2DataElements),
		Dhis2CategoryOptionCombos:  normalizeRapidexDhis2COCOptions(input.Dhis2CategoryOptionCombos),
		Dhis2AttributeOptionCombos: normalizeRapidexDhis2AOCOptions(input.Dhis2AttributeOptionCombos),
	}
}

func normalizeRapidexRapidProFlows(input []rapidpro.Flow) []RapidexRapidProFlowOption {
	output := make([]RapidexRapidProFlowOption, 0, len(input))
	for _, item := range input {
		results := make([]RapidexRapidProFlowResultOption, 0, len(item.Results))
		for _, result := range item.Results {
			key := strings.TrimSpace(result.Key)
			if key == "" {
				continue
			}
			results = append(results, RapidexRapidProFlowResultOption{
				Key:        key,
				Name:       strings.TrimSpace(result.Name),
				Categories: normalizeStringSlice(result.Categories),
			})
		}
		slices.SortFunc(results, func(left, right RapidexRapidProFlowResultOption) int {
			return strings.Compare(strings.ToLower(left.Key), strings.ToLower(right.Key))
		})
		output = append(output, RapidexRapidProFlowOption{
			UUID:       strings.TrimSpace(item.UUID),
			Name:       strings.TrimSpace(item.Name),
			Type:       strings.TrimSpace(item.Type),
			Archived:   item.Archived,
			ParentRefs: normalizeStringSlice(item.ParentRefs),
			ModifiedOn: strings.TrimSpace(item.ModifiedOn),
			Results:    results,
		})
	}
	return normalizeRapidexRapidProFlowsOptions(output)
}

func normalizeRapidexRapidProFlowsOptions(input []RapidexRapidProFlowOption) []RapidexRapidProFlowOption {
	output := make([]RapidexRapidProFlowOption, 0, len(input))
	for _, item := range input {
		uuid := strings.TrimSpace(item.UUID)
		if uuid == "" {
			continue
		}
		results := make([]RapidexRapidProFlowResultOption, 0, len(item.Results))
		for _, result := range item.Results {
			key := strings.TrimSpace(result.Key)
			if key == "" {
				continue
			}
			results = append(results, RapidexRapidProFlowResultOption{
				Key:        key,
				Name:       strings.TrimSpace(result.Name),
				Categories: normalizeStringSlice(result.Categories),
			})
		}
		slices.SortFunc(results, func(left, right RapidexRapidProFlowResultOption) int {
			return strings.Compare(strings.ToLower(left.Key), strings.ToLower(right.Key))
		})
		output = append(output, RapidexRapidProFlowOption{
			UUID:       uuid,
			Name:       strings.TrimSpace(item.Name),
			Type:       strings.TrimSpace(item.Type),
			Archived:   item.Archived,
			ParentRefs: normalizeStringSlice(item.ParentRefs),
			ModifiedOn: strings.TrimSpace(item.ModifiedOn),
			Results:    results,
		})
	}
	slices.SortFunc(output, func(left, right RapidexRapidProFlowOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return output
}

func normalizeRapidexDhis2Datasets(input []dhis2metadata.DataSet) []RapidexDhis2DatasetOption {
	output := make([]RapidexDhis2DatasetOption, 0, len(input))
	for _, item := range input {
		refs := make([]RapidexDhis2DataElementRef, 0, len(item.DataSetElements))
		for _, ref := range item.DataSetElements {
			if strings.TrimSpace(ref.DataElement.ID) == "" {
				continue
			}
			refs = append(refs, RapidexDhis2DataElementRef{ID: strings.TrimSpace(ref.DataElement.ID), Name: strings.TrimSpace(ref.DataElement.Name)})
		}
		slices.SortFunc(refs, func(left, right RapidexDhis2DataElementRef) int {
			return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
		})
		output = append(output, RapidexDhis2DatasetOption{
			ID:           strings.TrimSpace(item.ID),
			Name:         strings.TrimSpace(item.Name),
			PeriodType:   strings.TrimSpace(item.PeriodType),
			DataElements: refs,
		})
	}
	return normalizeRapidexDhis2DatasetOptions(output)
}

func normalizeRapidexDhis2DatasetOptions(input []RapidexDhis2DatasetOption) []RapidexDhis2DatasetOption {
	output := make([]RapidexDhis2DatasetOption, 0, len(input))
	for _, item := range input {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		refs := make([]RapidexDhis2DataElementRef, 0, len(item.DataElements))
		for _, ref := range item.DataElements {
			refID := strings.TrimSpace(ref.ID)
			if refID == "" {
				continue
			}
			refs = append(refs, RapidexDhis2DataElementRef{ID: refID, Name: strings.TrimSpace(ref.Name)})
		}
		slices.SortFunc(refs, func(left, right RapidexDhis2DataElementRef) int {
			return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
		})
		output = append(output, RapidexDhis2DatasetOption{ID: id, Name: strings.TrimSpace(item.Name), PeriodType: strings.TrimSpace(item.PeriodType), DataElements: refs})
	}
	slices.SortFunc(output, func(left, right RapidexDhis2DatasetOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return output
}

func normalizeRapidexDhis2DataElements(input []dhis2metadata.DataElement) []RapidexDhis2DataElementOption {
	output := make([]RapidexDhis2DataElementOption, 0, len(input))
	for _, item := range input {
		output = append(output, RapidexDhis2DataElementOption{ID: strings.TrimSpace(item.ID), Name: strings.TrimSpace(item.Name), ValueType: strings.TrimSpace(item.ValueType)})
	}
	return normalizeRapidexDhis2DataElementOptions(output)
}

func normalizeRapidexDhis2DataElementOptions(input []RapidexDhis2DataElementOption) []RapidexDhis2DataElementOption {
	output := make([]RapidexDhis2DataElementOption, 0, len(input))
	for _, item := range input {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		output = append(output, RapidexDhis2DataElementOption{ID: id, Name: strings.TrimSpace(item.Name), ValueType: strings.TrimSpace(item.ValueType)})
	}
	slices.SortFunc(output, func(left, right RapidexDhis2DataElementOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return output
}

func normalizeRapidexDhis2COCs(input []dhis2metadata.CategoryOptionCombo) []RapidexDhis2CategoryOptionComboOption {
	output := make([]RapidexDhis2CategoryOptionComboOption, 0, len(input))
	for _, item := range input {
		output = append(output, RapidexDhis2CategoryOptionComboOption{ID: strings.TrimSpace(item.ID), Name: strings.TrimSpace(item.Name)})
	}
	return normalizeRapidexDhis2COCOptions(output)
}

func normalizeRapidexDhis2COCOptions(input []RapidexDhis2CategoryOptionComboOption) []RapidexDhis2CategoryOptionComboOption {
	output := make([]RapidexDhis2CategoryOptionComboOption, 0, len(input))
	for _, item := range input {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		output = append(output, RapidexDhis2CategoryOptionComboOption{ID: id, Name: strings.TrimSpace(item.Name)})
	}
	slices.SortFunc(output, func(left, right RapidexDhis2CategoryOptionComboOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return output
}

func normalizeRapidexDhis2AOCs(input []dhis2metadata.AttributeOptionCombo) []RapidexDhis2AttributeOptionComboOption {
	output := make([]RapidexDhis2AttributeOptionComboOption, 0, len(input))
	for _, item := range input {
		output = append(output, RapidexDhis2AttributeOptionComboOption{ID: strings.TrimSpace(item.ID), Name: strings.TrimSpace(item.Name)})
	}
	return normalizeRapidexDhis2AOCOptions(output)
}

func normalizeRapidexDhis2AOCOptions(input []RapidexDhis2AttributeOptionComboOption) []RapidexDhis2AttributeOptionComboOption {
	output := make([]RapidexDhis2AttributeOptionComboOption, 0, len(input))
	for _, item := range input {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		output = append(output, RapidexDhis2AttributeOptionComboOption{ID: id, Name: strings.TrimSpace(item.Name)})
	}
	slices.SortFunc(output, func(left, right RapidexDhis2AttributeOptionComboOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return output
}

func buildRapidexMetadataWarnings(mappings []rapidex.MappingConfig, snapshot RapidexWebhookMetadataSnapshot) []string {
	warnings := []string{}
	flowKeys := map[string]RapidexRapidProFlowOption{}
	for _, flow := range snapshot.RapidProFlows {
		flowKeys[strings.TrimSpace(flow.UUID)] = flow
	}
	datasetIDs := map[string]struct{}{}
	for _, dataset := range snapshot.Dhis2Datasets {
		datasetIDs[strings.TrimSpace(dataset.ID)] = struct{}{}
	}
	dataElementIDs := map[string]struct{}{}
	for _, item := range snapshot.Dhis2DataElements {
		dataElementIDs[strings.TrimSpace(item.ID)] = struct{}{}
	}
	contactFieldKeys := map[string]struct{}{}
	for _, field := range snapshot.RapidProContactFields {
		contactFieldKeys[strings.ToLower(strings.TrimSpace(field.Key))] = struct{}{}
	}
	for _, mapping := range mappings {
		if _, ok := flowKeys[strings.TrimSpace(mapping.FlowUUID)]; !ok {
			warnings = append(warnings, fmt.Sprintf("Mapped flow %q is not present in the last RapidPro metadata snapshot.", mapping.FlowUUID))
		}
		if _, ok := datasetIDs[strings.TrimSpace(mapping.Dataset)]; !ok {
			warnings = append(warnings, fmt.Sprintf("Mapped dataset %q is not present in the last DHIS2 metadata snapshot.", mapping.Dataset))
		}
		if value := strings.ToLower(strings.TrimSpace(mapping.OrgUnitVar)); value != "" {
			if !rapidexSnapshotHasSourceField(flowKeys[strings.TrimSpace(mapping.FlowUUID)], contactFieldKeys, value) {
				warnings = append(warnings, fmt.Sprintf("Org unit variable %q was not found in discovered RapidPro flow results or contact fields.", mapping.OrgUnitVar))
			}
		}
		if value := strings.ToLower(strings.TrimSpace(mapping.PeriodVar)); value != "" {
			if !rapidexSnapshotHasSourceField(flowKeys[strings.TrimSpace(mapping.FlowUUID)], contactFieldKeys, value) {
				warnings = append(warnings, fmt.Sprintf("Period variable %q was not found in discovered RapidPro flow results or contact fields.", mapping.PeriodVar))
			}
		}
		for _, row := range mapping.Mappings {
			if _, ok := dataElementIDs[strings.TrimSpace(row.DataElement)]; !ok {
				warnings = append(warnings, fmt.Sprintf("Mapped data element %q is not present in the last DHIS2 metadata snapshot.", row.DataElement))
			}
			if value := strings.ToLower(strings.TrimSpace(row.Field)); value != "" {
				if !rapidexSnapshotHasSourceField(flowKeys[strings.TrimSpace(mapping.FlowUUID)], contactFieldKeys, value) {
					warnings = append(warnings, fmt.Sprintf("Webhook field %q was not found in discovered RapidPro flow results or contact fields.", row.Field))
				}
			}
		}
	}
	return normalizeStringSlice(warnings)
}

func rapidexSnapshotHasSourceField(flow RapidexRapidProFlowOption, contactFields map[string]struct{}, key string) bool {
	for _, result := range flow.Results {
		if strings.EqualFold(strings.TrimSpace(result.Key), key) {
			return true
		}
	}
	_, ok := contactFields[key]
	return ok
}

func normalizeRapidexDHIS2ServerCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return defaultDHIS2ServerCode
}

func normalizeStringSlice(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	output := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, trimmed)
	}
	slices.SortFunc(output, func(left, right string) int { return strings.Compare(strings.ToLower(left), strings.ToLower(right)) })
	return output
}
