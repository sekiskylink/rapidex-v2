package reporter

import (
	"encoding/json"
	"strings"
	"time"
)

// Reporter represents a RapidPro contact that has permission to submit reports.
type Reporter struct {
	ID                int64      `db:"id" json:"id"`
	UID               string     `db:"uid" json:"uid"`
	Name              string     `db:"name" json:"name"`
	Telephone         string     `db:"telephone" json:"telephone"`
	WhatsApp          string     `db:"whatsapp" json:"whatsapp"`
	Telegram          string     `db:"telegram" json:"telegram"`
	OrgUnitID         int64      `db:"org_unit_id" json:"orgUnitId"`
	ReportingLocation string     `db:"reporting_location" json:"reportingLocation"`
	DistrictID        *int64     `db:"district_id" json:"districtId,omitempty"`
	TotalReports      int        `db:"total_reports" json:"totalReports"`
	LastReportingDate *time.Time `db:"last_reporting_date" json:"lastReportingDate,omitempty"`
	SMSCode           string     `db:"sms_code" json:"smsCode"`
	SMSCodeExpiresAt  *time.Time `db:"sms_code_expires_at" json:"smsCodeExpiresAt,omitempty"`
	MTUUID            string     `db:"mtuuid" json:"mtuuid"`
	Synced            bool       `db:"synced" json:"synced"`
	RapidProUUID      string     `db:"rapidpro_uuid" json:"rapidProUuid"`
	IsActive          bool       `db:"is_active" json:"isActive"`
	CreatedAt         time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updatedAt"`
	LastLoginAt       *time.Time `db:"last_login_at" json:"lastLoginAt,omitempty"`
	Groups            []string   `json:"groups"`
}

func (r *Reporter) UnmarshalJSON(data []byte) error {
	type reporterAlias Reporter
	aux := struct {
		reporterAlias
		DisplayName string   `json:"displayName"`
		PhoneNumber string   `json:"phoneNumber"`
		GroupNames  []string `json:"groupNames"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = Reporter(aux.reporterAlias)
	if strings.TrimSpace(r.Name) == "" {
		r.Name = strings.TrimSpace(aux.DisplayName)
	}
	if strings.TrimSpace(r.Telephone) == "" {
		r.Telephone = strings.TrimSpace(aux.PhoneNumber)
	}
	if len(r.Groups) == 0 && len(aux.GroupNames) > 0 {
		r.Groups = aux.GroupNames
	}
	return nil
}

type ListQuery struct {
	Page       int
	PageSize   int
	Search     string
	OrgUnitID  *int64
	OnlyActive bool
}

type ListResult struct {
	Items    []Reporter `json:"items"`
	Total    int        `json:"totalCount"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

type SyncResult struct {
	Reporter   Reporter `json:"reporter"`
	Operation  string   `json:"operation"`
	GroupCount int      `json:"groupCount"`
}

type SyncBatchResult struct {
	Requested     int        `json:"requested"`
	Scanned       int        `json:"scanned"`
	Synced        int        `json:"synced"`
	Created       int        `json:"created"`
	Updated       int        `json:"updated"`
	Failed        int        `json:"failed"`
	FailedIDs     []int64    `json:"failedIds,omitempty"`
	FailedNames   []string   `json:"failedNames,omitempty"`
	Reporters     []Reporter `json:"reporters,omitempty"`
	WatermarkFrom *time.Time `json:"watermarkFrom,omitempty"`
	WatermarkTo   *time.Time `json:"watermarkTo,omitempty"`
	DryRun        bool       `json:"dryRun"`
	OnlyActive    bool       `json:"onlyActive"`
}

type MessageResult struct {
	Reporter Reporter `json:"reporter"`
	Message  string   `json:"message"`
}

type BroadcastResult struct {
	ReporterIDs []int64 `json:"reporterIds"`
	Message     string  `json:"message"`
}

type RapidProContactSnapshot struct {
	UUID       string            `json:"uuid"`
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Language   string            `json:"language"`
	URNs       []string          `json:"urns"`
	Groups     []RapidProGroup   `json:"groups"`
	Fields     map[string]string `json:"fields"`
	Flow       *RapidProFlow     `json:"flow,omitempty"`
	CreatedOn  string            `json:"createdOn"`
	ModifiedOn string            `json:"modifiedOn"`
	LastSeenOn string            `json:"lastSeenOn"`
}

type RapidProGroup struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type RapidProFlow struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type RapidProContactDetailsResult struct {
	Reporter Reporter                 `json:"reporter"`
	Found    bool                     `json:"found"`
	Contact  *RapidProContactSnapshot `json:"contact,omitempty"`
}

type RapidProMessageRecord struct {
	ID          int64         `json:"id"`
	BroadcastID *int64        `json:"broadcastId,omitempty"`
	Direction   string        `json:"direction"`
	Type        string        `json:"type"`
	Status      string        `json:"status"`
	Visibility  string        `json:"visibility"`
	Text        string        `json:"text"`
	URN         string        `json:"urn"`
	Channel     *RapidProFlow `json:"channel,omitempty"`
	Flow        *RapidProFlow `json:"flow,omitempty"`
	CreatedOn   string        `json:"createdOn"`
	SentOn      string        `json:"sentOn"`
	ModifiedOn  string        `json:"modifiedOn"`
}

type RapidProMessageHistoryResult struct {
	Reporter Reporter                `json:"reporter"`
	Found    bool                    `json:"found"`
	Items    []RapidProMessageRecord `json:"items"`
	Next     string                  `json:"next,omitempty"`
}
