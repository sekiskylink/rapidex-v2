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
	rapidexWebhookMetadataCategory      = "rapidex"
	rapidexWebhookMetadataKey           = "webhook_metadata"
	defaultDHIS2ServerCode              = "dhis2"
	rapidexMetadataRefreshScopeCatalog  = "catalog"
	rapidexMetadataRefreshScopeDatasets = "datasets"
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
	GetDataSet(context.Context, dhis2metadata.Connection, string) (dhis2metadata.DataSet, error)
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

type RapidexDhis2DatasetOption struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PeriodType string `json:"periodType,omitempty"`
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

type RapidexDhis2DatasetMetadata struct {
	ID                    string                                   `json:"id"`
	Name                  string                                   `json:"name"`
	PeriodType            string                                   `json:"periodType,omitempty"`
	LastRefreshedAt       *time.Time                               `json:"lastRefreshedAt,omitempty"`
	DataElements          []RapidexDhis2DataElementOption          `json:"dataElements"`
	CategoryOptionCombos  []RapidexDhis2CategoryOptionComboOption  `json:"categoryOptionCombos"`
	AttributeOptionCombos []RapidexDhis2AttributeOptionComboOption `json:"attributeOptionCombos"`
}

type RapidexWebhookMetadataSnapshot struct {
	RapidProServerCode       string                                 `json:"rapidProServerCode"`
	Dhis2ServerCode          string                                 `json:"dhis2ServerCode"`
	LastRefreshedAt          *time.Time                             `json:"lastRefreshedAt,omitempty"`
	RapidProFlows            []RapidexRapidProFlowOption            `json:"rapidProFlows"`
	RapidProContactFields    []rapidpro.ContactField                `json:"rapidProContactFields"`
	Dhis2Datasets            []RapidexDhis2DatasetOption            `json:"dhis2Datasets"`
	Dhis2LoadedDatasetIDs    []string                               `json:"dhis2LoadedDatasetIds"`
	Dhis2DatasetMetadataByID map[string]RapidexDhis2DatasetMetadata `json:"dhis2DatasetMetadataById"`
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
	RapidProServerCode string   `json:"rapidProServerCode"`
	Dhis2ServerCode    string   `json:"dhis2ServerCode"`
	Scope              string   `json:"scope"`
	DatasetIDs         []string `json:"datasetIds"`
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
	scope := normalizeRapidexMetadataRefreshScope(input.Scope)
	dhis2Conn := dhis2metadata.Connection{
		BaseURL:   dhis2Record.BaseURL,
		Headers:   dhis2Record.Headers,
		URLParams: dhis2Record.URLParams,
	}
	storedMetadata, err := s.getRapidexWebhookMetadataStored(ctx)
	if err != nil {
		return RapidexWebhookMetadataResponse{}, err
	}
	snapshot := normalizeRapidexWebhookMetadataSnapshot(storedMetadata.Snapshot)
	if (snapshot.RapidProServerCode != "" && snapshot.RapidProServerCode != rapidProCode) || (snapshot.Dhis2ServerCode != "" && snapshot.Dhis2ServerCode != dhis2Code) {
		snapshot = normalizeRapidexWebhookMetadataSnapshot(RapidexWebhookMetadataSnapshot{})
	}
	now := time.Now().UTC()
	snapshot.RapidProServerCode = rapidProCode
	snapshot.Dhis2ServerCode = dhis2Code
	snapshot.LastRefreshedAt = &now

	switch scope {
	case rapidexMetadataRefreshScopeCatalog:
		flows, flowErr := s.rapidexRapidProClient.ListFlows(ctx, rapidpro.Connection{BaseURL: rapidProRecord.BaseURL, Headers: rapidProRecord.Headers})
		if flowErr != nil {
			return RapidexWebhookMetadataResponse{}, flowErr
		}
		fields, fieldErr := s.rapidexRapidProClient.ListContactFields(ctx, rapidpro.Connection{BaseURL: rapidProRecord.BaseURL, Headers: rapidProRecord.Headers})
		if fieldErr != nil {
			return RapidexWebhookMetadataResponse{}, fieldErr
		}
		dataSets, dataSetErr := s.rapidexDHIS2Client.ListDataSets(ctx, dhis2Conn)
		if dataSetErr != nil {
			return RapidexWebhookMetadataResponse{}, dataSetErr
		}
		snapshot.RapidProFlows = normalizeRapidexRapidProFlows(flows)
		snapshot.RapidProContactFields = normalizeRapidProFields(fields)
		snapshot.Dhis2Datasets = normalizeRapidexDhis2Datasets(dataSets)
	case rapidexMetadataRefreshScopeDatasets:
		datasetIDs := normalizeStringSlice(input.DatasetIDs)
		if len(datasetIDs) == 0 {
			return RapidexWebhookMetadataResponse{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"datasetIds": []string{"at least one dataset id is required when scope is datasets"},
			})
		}
		if len(snapshot.Dhis2Datasets) == 0 {
			dataSets, dataSetErr := s.rapidexDHIS2Client.ListDataSets(ctx, dhis2Conn)
			if dataSetErr != nil {
				return RapidexWebhookMetadataResponse{}, dataSetErr
			}
			snapshot.Dhis2Datasets = normalizeRapidexDhis2Datasets(dataSets)
		}
		for _, datasetID := range datasetIDs {
			dataSet, dataSetErr := s.rapidexDHIS2Client.GetDataSet(ctx, dhis2Conn, datasetID)
			if dataSetErr != nil {
				return RapidexWebhookMetadataResponse{}, dataSetErr
			}
			metadata := normalizeRapidexDhis2DatasetMetadata(dataSet, &now)
			snapshot.Dhis2DatasetMetadataByID[datasetID] = metadata
		}
		snapshot.Dhis2LoadedDatasetIDs = normalizeStringSlice(append(snapshot.Dhis2LoadedDatasetIDs, datasetIDs...))
	default:
		return RapidexWebhookMetadataResponse{}, apperror.ValidationWithDetails("validation failed", map[string]any{
			"scope": []string{"must be one of: catalog, datasets"},
		})
	}
	snapshot = normalizeRapidexWebhookMetadataSnapshot(snapshot)
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
			"scope":              scope,
			"flowCount":          len(snapshot.RapidProFlows),
			"datasetCount":       len(snapshot.Dhis2Datasets),
			"loadedDatasetCount": len(snapshot.Dhis2LoadedDatasetIDs),
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
	datasetMetadataByID := map[string]RapidexDhis2DatasetMetadata{}
	for key, value := range input.Dhis2DatasetMetadataByID {
		datasetID := strings.TrimSpace(key)
		if datasetID == "" {
			continue
		}
		datasetMetadataByID[datasetID] = normalizeRapidexDhis2DatasetMetadataOption(value, datasetID)
	}
	return RapidexWebhookMetadataSnapshot{
		RapidProServerCode:       normalizeRapidProServerCode(input.RapidProServerCode),
		Dhis2ServerCode:          normalizeRapidexDHIS2ServerCode(input.Dhis2ServerCode),
		LastRefreshedAt:          input.LastRefreshedAt,
		RapidProFlows:            normalizeRapidexRapidProFlowsOptions(input.RapidProFlows),
		RapidProContactFields:    normalizeRapidProFields(input.RapidProContactFields),
		Dhis2Datasets:            normalizeRapidexDhis2DatasetOptions(input.Dhis2Datasets),
		Dhis2LoadedDatasetIDs:    normalizeStringSlice(input.Dhis2LoadedDatasetIDs),
		Dhis2DatasetMetadataByID: datasetMetadataByID,
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
		output = append(output, RapidexDhis2DatasetOption{
			ID:         strings.TrimSpace(item.ID),
			Name:       strings.TrimSpace(item.Name),
			PeriodType: strings.TrimSpace(item.PeriodType),
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
		output = append(output, RapidexDhis2DatasetOption{ID: id, Name: strings.TrimSpace(item.Name), PeriodType: strings.TrimSpace(item.PeriodType)})
	}
	slices.SortFunc(output, func(left, right RapidexDhis2DatasetOption) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return output
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

func normalizeRapidexDhis2AOCs(input []dhis2metadata.CategoryOptionCombo) []RapidexDhis2AttributeOptionComboOption {
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

func normalizeRapidexDhis2DatasetMetadata(input dhis2metadata.DataSet, refreshedAt *time.Time) RapidexDhis2DatasetMetadata {
	dataElements := make([]RapidexDhis2DataElementOption, 0, len(input.DataSetElements))
	categoryOptionCombos := make([]dhis2metadata.CategoryOptionCombo, 0)
	for _, ref := range input.DataSetElements {
		dataElementID := strings.TrimSpace(ref.DataElement.ID)
		if dataElementID == "" {
			continue
		}
		dataElements = append(dataElements, RapidexDhis2DataElementOption{
			ID:        dataElementID,
			Name:      strings.TrimSpace(ref.DataElement.Name),
			ValueType: strings.TrimSpace(ref.DataElement.ValueType),
		})
		categoryOptionCombos = append(categoryOptionCombos, ref.DataElement.CategoryCombo.CategoryOptionCombos...)
	}
	return normalizeRapidexDhis2DatasetMetadataOption(RapidexDhis2DatasetMetadata{
		ID:                    strings.TrimSpace(input.ID),
		Name:                  strings.TrimSpace(input.Name),
		PeriodType:            strings.TrimSpace(input.PeriodType),
		LastRefreshedAt:       refreshedAt,
		DataElements:          dataElements,
		CategoryOptionCombos:  normalizeRapidexDhis2COCs(categoryOptionCombos),
		AttributeOptionCombos: normalizeRapidexDhis2AOCs(categoryOptionCombos),
	}, strings.TrimSpace(input.ID))
}

func normalizeRapidexDhis2DatasetMetadataOption(input RapidexDhis2DatasetMetadata, fallbackID string) RapidexDhis2DatasetMetadata {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = strings.TrimSpace(fallbackID)
	}
	return RapidexDhis2DatasetMetadata{
		ID:                    id,
		Name:                  strings.TrimSpace(input.Name),
		PeriodType:            strings.TrimSpace(input.PeriodType),
		LastRefreshedAt:       input.LastRefreshedAt,
		DataElements:          normalizeRapidexDhis2DataElementOptions(input.DataElements),
		CategoryOptionCombos:  normalizeRapidexDhis2COCOptions(input.CategoryOptionCombos),
		AttributeOptionCombos: normalizeRapidexDhis2AOCOptions(input.AttributeOptionCombos),
	}
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
	contactFieldKeys := map[string]struct{}{}
	for _, field := range snapshot.RapidProContactFields {
		contactFieldKeys[strings.ToLower(strings.TrimSpace(field.Key))] = struct{}{}
	}
	for _, mapping := range mappings {
		mappingFlowUUID := strings.TrimSpace(mapping.FlowUUID)
		mappingDatasetID := strings.TrimSpace(mapping.Dataset)
		if _, ok := flowKeys[strings.TrimSpace(mapping.FlowUUID)]; !ok {
			warnings = append(warnings, fmt.Sprintf("Mapped flow %q is not present in the last RapidPro metadata snapshot.", mapping.FlowUUID))
		}
		if _, ok := datasetIDs[mappingDatasetID]; !ok {
			warnings = append(warnings, fmt.Sprintf("Mapped dataset %q is not present in the last DHIS2 metadata snapshot.", mapping.Dataset))
		}
		datasetMetadata, datasetMetadataLoaded := snapshot.Dhis2DatasetMetadataByID[mappingDatasetID]
		if mappingDatasetID != "" && !datasetMetadataLoaded {
			warnings = append(warnings, fmt.Sprintf("Mapped dataset %q has not had its dataset metadata loaded yet.", mapping.Dataset))
		}
		dataElementIDs := map[string]struct{}{}
		if datasetMetadataLoaded {
			for _, item := range datasetMetadata.DataElements {
				dataElementIDs[strings.TrimSpace(item.ID)] = struct{}{}
			}
		}
		if value := strings.ToLower(strings.TrimSpace(mapping.OrgUnitVar)); value != "" {
			if !rapidexSnapshotHasSourceField(flowKeys[mappingFlowUUID], contactFieldKeys, value) {
				warnings = append(warnings, fmt.Sprintf("Org unit variable %q was not found in discovered RapidPro flow results or contact fields.", mapping.OrgUnitVar))
			}
		}
		if value := strings.ToLower(strings.TrimSpace(mapping.PeriodVar)); value != "" {
			if !rapidexSnapshotHasSourceField(flowKeys[mappingFlowUUID], contactFieldKeys, value) {
				warnings = append(warnings, fmt.Sprintf("Period variable %q was not found in discovered RapidPro flow results or contact fields.", mapping.PeriodVar))
			}
		}
		for _, row := range mapping.Mappings {
			if datasetMetadataLoaded {
				if _, ok := dataElementIDs[strings.TrimSpace(row.DataElement)]; !ok {
					warnings = append(warnings, fmt.Sprintf("Mapped data element %q is not present in the last metadata loaded for dataset %q.", row.DataElement, mapping.Dataset))
				}
			}
			if value := strings.ToLower(strings.TrimSpace(row.Field)); value != "" {
				if !rapidexSnapshotHasSourceField(flowKeys[mappingFlowUUID], contactFieldKeys, value) {
					warnings = append(warnings, fmt.Sprintf("Webhook field %q was not found in discovered RapidPro flow results or contact fields.", row.Field))
				}
			}
		}
	}
	return normalizeStringSlice(warnings)
}

func normalizeRapidexMetadataRefreshScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", rapidexMetadataRefreshScopeCatalog:
		return rapidexMetadataRefreshScopeCatalog
	case rapidexMetadataRefreshScopeDatasets:
		return rapidexMetadataRefreshScopeDatasets
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
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
