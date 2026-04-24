package orgunit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"slices"
	"strings"
	"time"

	"basepro/backend/internal/logging"
	"basepro/backend/internal/sukumad/server"
)

const (
	syncStatusRunning   = "running"
	syncStatusSucceeded = "succeeded"
	syncStatusFailed    = "failed"
)

type hierarchySyncRepository interface {
	ReplaceHierarchy(ctx context.Context, input replaceHierarchyInput) (SyncResult, error)
	GetSyncState(ctx context.Context) (SyncState, error)
}

type hierarchyServerLookup interface {
	GetServerByCode(context.Context, string) (server.Record, error)
	GetServerByUID(context.Context, string) (server.Record, error)
}

type replaceHierarchyInput struct {
	Request           SyncRequest
	ServerCode        string
	ResolvedLevelUID  string
	ResolvedLevelName string
	StartedAt         time.Time
	CompletedAt       *time.Time
	Status            string
	ErrorMessage      string
	Levels            []Level
	Groups            []Group
	Attributes        []Attribute
	OrgUnits          []OrgUnit
	GroupMembers      map[string][]string
}

type remoteLevel struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type remoteGroup struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type remoteAttribute struct {
	ID                        string `json:"id"`
	Code                      string `json:"code"`
	Name                      string `json:"name"`
	ShortName                 string `json:"shortName"`
	ValueType                 string `json:"valueType"`
	Unique                    bool   `json:"unique"`
	Mandatory                 bool   `json:"mandatory"`
	OrganisationUnitAttribute bool   `json:"organisationUnitAttribute"`
}

type remoteAttributeValue struct {
	Attribute struct {
		ID string `json:"id"`
	} `json:"attribute"`
	Value any `json:"value"`
}

type remoteOrgUnit struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	Level       int    `json:"level"`
	Path        string `json:"path"`
	Address     string `json:"address"`
	Email       string `json:"email"`
	URL         string `json:"url"`
	PhoneNumber string `json:"phoneNumber"`
	OpeningDate string `json:"openingDate"`
	Deleted     bool   `json:"deleted"`
	Parent      *struct {
		ID string `json:"id"`
	} `json:"parent"`
	OrganisationUnitGroups []struct {
		ID string `json:"id"`
	} `json:"organisationUnitGroups"`
	AttributeValues []remoteAttributeValue `json:"attributeValues"`
}

type HierarchySyncService struct {
	repo         hierarchySyncRepository
	serverLookup hierarchyServerLookup
	httpClient   *http.Client
}

func NewHierarchySyncService(repo hierarchySyncRepository, serverLookup hierarchyServerLookup, httpClient *http.Client) *HierarchySyncService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &HierarchySyncService{
		repo:         repo,
		serverLookup: serverLookup,
		httpClient:   httpClient,
	}
}

func (s *HierarchySyncService) GetState(ctx context.Context) (SyncState, error) {
	if s == nil || s.repo == nil {
		return SyncState{}, errors.New("hierarchy sync service is not configured")
	}
	return s.repo.GetSyncState(ctx)
}

func (s *HierarchySyncService) Sync(ctx context.Context, req SyncRequest) (SyncResult, error) {
	if s == nil || s.repo == nil || s.serverLookup == nil {
		return SyncResult{}, errors.New("hierarchy sync service is not configured")
	}

	normalized, err := normalizeSyncRequest(req)
	if err != nil {
		return SyncResult{}, err
	}

	serverRecord, err := s.lookupServer(ctx, normalized)
	if err != nil {
		return SyncResult{}, err
	}
	startedAt := time.Now().UTC()
	logger := logging.ForContext(ctx).With(
		slog.String("server_code", serverRecord.Code),
		slog.Bool("dry_run", normalized.DryRun),
		slog.Bool("full_refresh", normalized.FullRefresh),
		slog.String("district_level_name", normalized.DistrictLevelName),
		slog.String("district_level_code", normalized.DistrictLevelCode),
	)
	logger.Info("org_unit_hierarchy_sync_started")

	levels, err := s.fetchLevels(ctx, serverRecord)
	if err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, fmt.Errorf("fetch levels: %w", err))
	}
	groups, err := s.fetchGroups(ctx, serverRecord)
	if err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, fmt.Errorf("fetch groups: %w", err))
	}
	attributes, err := s.fetchAttributes(ctx, serverRecord)
	if err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, fmt.Errorf("fetch attributes: %w", err))
	}
	orgUnits, memberships, err := s.fetchOrgUnits(ctx, serverRecord)
	if err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, fmt.Errorf("fetch organisation units: %w", err))
	}

	levelUID, levelName, err := resolveDistrictLevel(levels, normalized.DistrictLevelName, normalized.DistrictLevelCode)
	if err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, err)
	}
	if err := validateHierarchy(levels, orgUnits, memberships); err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, err)
	}

	completedAt := time.Now().UTC()
	input := replaceHierarchyInput{
		Request:           normalized,
		ServerCode:        serverRecord.Code,
		ResolvedLevelUID:  levelUID,
		ResolvedLevelName: levelName,
		StartedAt:         startedAt,
		CompletedAt:       &completedAt,
		Status:            syncStatusSucceeded,
		Levels:            levels,
		Groups:            groups,
		Attributes:        attributes,
		OrgUnits:          orgUnits,
		GroupMembers:      memberships,
	}
	result, err := s.repo.ReplaceHierarchy(ctx, input)
	if err != nil {
		return s.failWithoutMutation(ctx, normalized, serverRecord.Code, startedAt, err)
	}
	logger.Info("org_unit_hierarchy_sync_completed",
		slog.Int("levels_count", result.LevelsCount),
		slog.Int("groups_count", result.GroupsCount),
		slog.Int("attributes_count", result.AttributesCount),
		slog.Int("org_units_count", result.OrgUnitsCount),
		slog.Int("group_members_count", result.GroupMembersCount),
		slog.Int("deleted_reporters", result.DeletedReporters),
		slog.Int("deleted_assignments", result.DeletedAssignments),
	)
	return result, nil
}

func normalizeSyncRequest(req SyncRequest) (SyncRequest, error) {
	normalized := SyncRequest{
		ServerUID:         strings.TrimSpace(req.ServerUID),
		ServerCode:        strings.TrimSpace(req.ServerCode),
		FullRefresh:       req.FullRefresh,
		DryRun:            req.DryRun,
		DistrictLevelName: strings.TrimSpace(req.DistrictLevelName),
		DistrictLevelCode: strings.TrimSpace(req.DistrictLevelCode),
	}
	if !normalized.FullRefresh {
		normalized.FullRefresh = true
	}
	if normalized.ServerUID == "" && normalized.ServerCode == "" {
		return SyncRequest{}, fmt.Errorf("serverUid or serverCode is required")
	}
	if normalized.DistrictLevelName == "" && normalized.DistrictLevelCode == "" {
		return SyncRequest{}, fmt.Errorf("districtLevelName or districtLevelCode is required")
	}
	return normalized, nil
}

func (s *HierarchySyncService) lookupServer(ctx context.Context, req SyncRequest) (server.Record, error) {
	if req.ServerUID != "" {
		return s.serverLookup.GetServerByUID(ctx, req.ServerUID)
	}
	return s.serverLookup.GetServerByCode(ctx, req.ServerCode)
}

func (s *HierarchySyncService) failWithoutMutation(ctx context.Context, req SyncRequest, serverCode string, startedAt time.Time, err error) (SyncResult, error) {
	completedAt := time.Now().UTC()
	result, replaceErr := s.repo.ReplaceHierarchy(ctx, replaceHierarchyInput{
		Request:           req,
		ServerCode:        serverCode,
		ResolvedLevelName: req.DistrictLevelName,
		StartedAt:         startedAt,
		CompletedAt:       &completedAt,
		Status:            syncStatusFailed,
		ErrorMessage:      err.Error(),
	})
	if replaceErr != nil {
		return SyncResult{}, errors.Join(err, replaceErr)
	}
	return result, err
}

func (s *HierarchySyncService) fetchLevels(ctx context.Context, record server.Record) ([]Level, error) {
	var payload struct {
		OrganisationUnitLevels []remoteLevel `json:"organisationUnitLevels"`
	}
	if err := s.fetchJSON(ctx, record, "/api/organisationUnitLevels.json", map[string]string{
		"paging": "false",
		"fields": "id,code,name,level",
	}, &payload); err != nil {
		return nil, err
	}
	items := make([]Level, 0, len(payload.OrganisationUnitLevels))
	for _, item := range payload.OrganisationUnitLevels {
		items = append(items, Level{
			UID:   strings.TrimSpace(item.ID),
			Code:  strings.TrimSpace(item.Code),
			Name:  strings.TrimSpace(item.Name),
			Level: item.Level,
		})
	}
	slices.SortFunc(items, func(a, b Level) int {
		return a.Level - b.Level
	})
	return items, nil
}

func (s *HierarchySyncService) fetchGroups(ctx context.Context, record server.Record) ([]Group, error) {
	var payload struct {
		OrganisationUnitGroups []remoteGroup `json:"organisationUnitGroups"`
	}
	if err := s.fetchJSON(ctx, record, "/api/organisationUnitGroups.json", map[string]string{
		"paging": "false",
		"fields": "id,code,name,shortName",
	}, &payload); err != nil {
		return nil, err
	}
	items := make([]Group, 0, len(payload.OrganisationUnitGroups))
	for _, item := range payload.OrganisationUnitGroups {
		items = append(items, Group{
			UID:       strings.TrimSpace(item.ID),
			Code:      strings.TrimSpace(item.Code),
			Name:      strings.TrimSpace(item.Name),
			ShortName: strings.TrimSpace(item.ShortName),
		})
	}
	return items, nil
}

func (s *HierarchySyncService) fetchAttributes(ctx context.Context, record server.Record) ([]Attribute, error) {
	var payload struct {
		Attributes []remoteAttribute `json:"attributes"`
	}
	if err := s.fetchJSON(ctx, record, "/api/attributes.json", map[string]string{
		"paging": "false",
		"fields": "id,code,name,shortName,valueType,unique,mandatory,organisationUnitAttribute",
		"filter": "organisationUnitAttribute:eq:true",
	}, &payload); err != nil {
		return nil, err
	}
	items := make([]Attribute, 0, len(payload.Attributes))
	for _, item := range payload.Attributes {
		items = append(items, Attribute{
			UID:                       strings.TrimSpace(item.ID),
			Code:                      strings.TrimSpace(item.Code),
			Name:                      strings.TrimSpace(item.Name),
			ShortName:                 strings.TrimSpace(item.ShortName),
			ValueType:                 strings.TrimSpace(item.ValueType),
			IsUnique:                  item.Unique,
			Mandatory:                 item.Mandatory,
			OrganisationUnitAttribute: item.OrganisationUnitAttribute,
		})
	}
	return items, nil
}

func (s *HierarchySyncService) fetchOrgUnits(ctx context.Context, record server.Record) ([]OrgUnit, map[string][]string, error) {
	var payload struct {
		OrganisationUnits []remoteOrgUnit `json:"organisationUnits"`
	}
	if err := s.fetchJSON(ctx, record, "/api/organisationUnits.json", map[string]string{
		"paging": "false",
		"fields": "id,code,name,shortName,description,level,path,address,email,url,phoneNumber,openingDate,deleted,parent[id],attributeValues[attribute[id],value],organisationUnitGroups[id]",
	}, &payload); err != nil {
		return nil, nil, err
	}
	items := make([]OrgUnit, 0, len(payload.OrganisationUnits))
	memberships := make(map[string][]string, len(payload.OrganisationUnits))
	for _, item := range payload.OrganisationUnits {
		var parentUID string
		if item.Parent != nil {
			parentUID = strings.TrimSpace(item.Parent.ID)
		}
		var openingDate *time.Time
		if parsed, ok := parseRemoteDate(item.OpeningDate); ok {
			openingDate = &parsed
		}
		attributes := JSONMap{}
		for _, value := range item.AttributeValues {
			key := strings.TrimSpace(value.Attribute.ID)
			if key == "" {
				continue
			}
			attributes[key] = value.Value
		}
		path := normalizePath(item.Path, item.ID)
		unit := OrgUnit{
			UID:             strings.TrimSpace(item.ID),
			Code:            strings.TrimSpace(item.Code),
			Name:            strings.TrimSpace(item.Name),
			ShortName:       firstNonEmpty(strings.TrimSpace(item.ShortName), strings.TrimSpace(item.Name)),
			Description:     strings.TrimSpace(item.Description),
			HierarchyLevel:  item.Level,
			Path:            path,
			Address:         strings.TrimSpace(item.Address),
			Email:           strings.TrimSpace(item.Email),
			URL:             strings.TrimSpace(item.URL),
			PhoneNumber:     strings.TrimSpace(item.PhoneNumber),
			Extras:          JSONMap{},
			AttributeValues: attributes,
			OpeningDate:     openingDate,
			Deleted:         item.Deleted,
		}
		if parentUID != "" {
			unit.Extras["parentUid"] = parentUID
		}
		items = append(items, unit)
		groupIDs := make([]string, 0, len(item.OrganisationUnitGroups))
		for _, group := range item.OrganisationUnitGroups {
			if uid := strings.TrimSpace(group.ID); uid != "" {
				groupIDs = append(groupIDs, uid)
			}
		}
		memberships[unit.UID] = groupIDs
	}
	slices.SortFunc(items, func(a, b OrgUnit) int {
		return strings.Compare(a.Path, b.Path)
	})
	return items, memberships, nil
}

func validateHierarchy(levels []Level, orgUnits []OrgUnit, memberships map[string][]string) error {
	if len(levels) == 0 {
		return fmt.Errorf("dhis2 organisation unit levels response is empty")
	}
	if len(orgUnits) == 0 {
		return fmt.Errorf("dhis2 organisation units response is empty")
	}
	levelSeen := make(map[int]struct{}, len(levels))
	for _, level := range levels {
		if strings.TrimSpace(level.UID) == "" || strings.TrimSpace(level.Name) == "" {
			return fmt.Errorf("dhis2 organisation unit levels contain blank uid or name")
		}
		if _, exists := levelSeen[level.Level]; exists {
			return fmt.Errorf("dhis2 organisation unit levels contain duplicate level %d", level.Level)
		}
		levelSeen[level.Level] = struct{}{}
	}
	unitSeen := make(map[string]OrgUnit, len(orgUnits))
	for _, unit := range orgUnits {
		if strings.TrimSpace(unit.UID) == "" || strings.TrimSpace(unit.Name) == "" {
			return fmt.Errorf("dhis2 organisation units contain blank uid or name")
		}
		if _, exists := unitSeen[unit.UID]; exists {
			return fmt.Errorf("dhis2 organisation units contain duplicate uid %s", unit.UID)
		}
		if _, ok := levelSeen[unit.HierarchyLevel]; !ok {
			return fmt.Errorf("dhis2 organisation unit %s references unknown level %d", unit.UID, unit.HierarchyLevel)
		}
		unitSeen[unit.UID] = unit
	}
	for uid, unit := range unitSeen {
		parentUID, _ := unit.Extras["parentUid"].(string)
		if parentUID == "" {
			continue
		}
		parent, ok := unitSeen[parentUID]
		if !ok {
			return fmt.Errorf("dhis2 organisation unit %s references missing parent %s", uid, parentUID)
		}
		if !strings.HasPrefix(unit.Path, parent.Path) {
			return fmt.Errorf("dhis2 organisation unit %s path does not match parent hierarchy", uid)
		}
	}
	for uid := range memberships {
		if _, ok := unitSeen[uid]; !ok {
			return fmt.Errorf("dhis2 group memberships reference unknown organisation unit %s", uid)
		}
	}
	return nil
}

func resolveDistrictLevel(levels []Level, districtLevelName string, districtLevelCode string) (string, string, error) {
	for _, level := range levels {
		if districtLevelName != "" && strings.EqualFold(strings.TrimSpace(level.Name), strings.TrimSpace(districtLevelName)) {
			return level.UID, level.Name, nil
		}
	}
	for _, level := range levels {
		if districtLevelCode != "" && strings.EqualFold(strings.TrimSpace(level.Code), strings.TrimSpace(districtLevelCode)) {
			return level.UID, level.Name, nil
		}
	}
	return "", "", fmt.Errorf("unable to resolve district level from DHIS2 metadata")
}

func parseRemoteDate(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func normalizePath(path string, uid string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = "/" + strings.TrimSpace(uid)
	}
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed + "/"
}

func (s *HierarchySyncService) fetchJSON(ctx context.Context, record server.Record, endpoint string, params map[string]string, dest any) error {
	requestURL, err := buildSyncURL(record.BaseURL, endpoint, record.URLParams, params)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	for key, value := range record.Headers {
		req.Header.Set(key, value)
	}
	response, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("dhis2 metadata request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(response.Body).Decode(dest)
}

func buildSyncURL(baseURL string, endpoint string, defaults map[string]string, params map[string]string) (string, error) {
	parsed, err := neturl.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return "", fmt.Errorf("invalid base url: %w", err)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + endpoint
	query := parsed.Query()
	for key, value := range defaults {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	for key, value := range params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
