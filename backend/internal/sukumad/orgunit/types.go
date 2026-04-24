package orgunit

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JSONMap map[string]any

func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}

	var raw []byte
	switch value := src.(type) {
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:
		return fmt.Errorf("scan json map: unsupported type %T", src)
	}

	if len(raw) == 0 {
		*m = JSONMap{}
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*m = JSONMap(decoded)
	return nil
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(map[string]any(m))
}

// OrgUnit represents an organisation unit imported from DHIS2 or created locally.
type OrgUnit struct {
	ID              int64      `db:"id" json:"id"`
	UID             string     `db:"uid" json:"uid"`
	Code            string     `db:"code" json:"code"`
	Name            string     `db:"name" json:"name"`
	ShortName       string     `db:"short_name" json:"shortName"`
	Description     string     `db:"description" json:"description"`
	ParentID        *int64     `db:"parent_id" json:"parentId,omitempty"`
	HierarchyLevel  int        `db:"hierarchy_level" json:"hierarchyLevel"`
	Path            string     `db:"path" json:"path"`
	DisplayPath     string     `db:"display_path" json:"displayPath"`
	Address         string     `db:"address" json:"address"`
	Email           string     `db:"email" json:"email"`
	URL             string     `db:"url" json:"url"`
	PhoneNumber     string     `db:"phone_number" json:"phoneNumber"`
	Extras          JSONMap    `db:"extras" json:"extras"`
	AttributeValues JSONMap    `db:"attribute_values" json:"attributeValues"`
	OpeningDate     *time.Time `db:"opening_date" json:"openingDate,omitempty"`
	Deleted         bool       `db:"deleted" json:"deleted"`
	LastSyncDate    *time.Time `db:"last_sync_date" json:"lastSyncDate,omitempty"`
	HasChildren     bool       `db:"has_children" json:"hasChildren"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`
}

type Level struct {
	ID    int64  `db:"id" json:"id"`
	UID   string `db:"uid" json:"uid"`
	Code  string `db:"code" json:"code"`
	Name  string `db:"name" json:"name"`
	Level int    `db:"level" json:"level"`
}

type Group struct {
	ID        int64  `db:"id" json:"id"`
	UID       string `db:"uid" json:"uid"`
	Code      string `db:"code" json:"code"`
	Name      string `db:"name" json:"name"`
	ShortName string `db:"short_name" json:"shortName"`
}

type Attribute struct {
	ID                        int64  `db:"id" json:"id"`
	UID                       string `db:"uid" json:"uid"`
	Code                      string `db:"code" json:"code"`
	Name                      string `db:"name" json:"name"`
	ShortName                 string `db:"short_name" json:"shortName"`
	ValueType                 string `db:"value_type" json:"valueType"`
	IsUnique                  bool   `db:"is_unique" json:"isUnique"`
	Mandatory                 bool   `db:"mandatory" json:"mandatory"`
	OrganisationUnitAttribute bool   `db:"organisation_unit_attribute" json:"organisationUnitAttribute"`
}

type SyncRequest struct {
	ServerUID         string `json:"serverUid"`
	ServerCode        string `json:"serverCode"`
	FullRefresh       bool   `json:"fullRefresh"`
	DryRun            bool   `json:"dryRun"`
	DistrictLevelName string `json:"districtLevelName"`
	DistrictLevelCode string `json:"districtLevelCode"`
}

type SyncResult struct {
	ServerCode          string     `json:"serverCode"`
	DryRun              bool       `json:"dryRun"`
	FullRefresh         bool       `json:"fullRefresh"`
	DistrictLevelName   string     `json:"districtLevelName"`
	DistrictLevelCode   string     `json:"districtLevelCode"`
	ResolvedDistrictUID string     `json:"resolvedDistrictUid"`
	ResolvedDistrict    string     `json:"resolvedDistrictName"`
	StartedAt           time.Time  `json:"startedAt"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
	Status              string     `json:"status"`
	LevelsCount         int        `json:"levelsCount"`
	GroupsCount         int        `json:"groupsCount"`
	AttributesCount     int        `json:"attributesCount"`
	OrgUnitsCount       int        `json:"orgUnitsCount"`
	GroupMembersCount   int        `json:"groupMembersCount"`
	DeletedReporters    int        `json:"deletedReporters"`
	DeletedAssignments  int        `json:"deletedAssignments"`
	ErrorMessage        string     `json:"errorMessage,omitempty"`
}

type SyncState struct {
	LastStartedAt     *time.Time `db:"last_started_at" json:"lastStartedAt,omitempty"`
	LastCompletedAt   *time.Time `db:"last_completed_at" json:"lastCompletedAt,omitempty"`
	LastSyncedAt      *time.Time `db:"last_synced_at" json:"lastSyncedAt,omitempty"`
	LastStatus        string     `db:"last_status" json:"lastStatus"`
	LastError         string     `db:"last_error" json:"lastError"`
	SourceServerCode  string     `db:"source_server_code" json:"sourceServerCode"`
	DistrictLevelName string     `db:"district_level_name" json:"districtLevelName"`
	DistrictLevelCode string     `db:"district_level_code" json:"districtLevelCode"`
	LastCounts        JSONMap    `db:"last_counts" json:"lastCounts"`
}

type ListQuery struct {
	Page            int
	PageSize        int
	Search          string
	ParentID        *int64
	RootsOnly       bool
	LeafOnly        bool
	ScopeRestricted bool
	ScopeRootIDs    []int64
	ScopePaths      []string
}

type ListResult struct {
	Items    []OrgUnit `json:"items"`
	Total    int       `json:"totalCount"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}
