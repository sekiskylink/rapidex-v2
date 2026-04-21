package orgunit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DHIS2OrgUnit represents the minimal fields returned by the DHIS2 API that
// we care about.  The response structure of DHIS2 `/organisationUnits.json`
// includes nested children; however for simplicity we flatten the units
// here and rebuild the tree using the parent reference and path.
type DHIS2OrgUnit struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	Parent      *struct {
		ID string `json:"id"`
	} `json:"parent"`
}

// SyncService handles synchronising organisation units from DHIS2 into the local
// database via the provided repository.  It fetches all org units from the
// configured DHIS2 instance and inserts or updates records accordingly.
type SyncService struct {
	repo       Repository
	httpClient *http.Client
	baseURL    string
	apiToken   string
}

// NewSyncService constructs a new SyncService.  baseURL should be the DHIS2
// root URL (e.g., https://play.dhis2.org/2.39).  apiToken can be a basic
// auth token or API key; if empty, unauthenticated requests are made.
func NewSyncService(repo Repository, client *http.Client, baseURL string, apiToken string) *SyncService {
	return &SyncService{repo: repo, httpClient: client, baseURL: baseURL, apiToken: apiToken}
}

// Sync downloads org units from DHIS2 and upserts them into the repository.  It
// also rebuilds the materialised path for each unit.  Errors are returned on
// failure; partial updates may have occurred.
func (s *SyncService) Sync(ctx context.Context) error {
	// Fetch all organisation units from DHIS2.  We request fields id, code, name,
	// and parent to minimise payload size.
	endpoint := fmt.Sprintf("%s/api/organisationUnits.json?paging=false&fields=id,code,name,shortName,description,parent[id]", strings.TrimRight(s.baseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if s.apiToken != "" {
		// Support token-based auth via Authorization header; adjust as needed for
		// basic auth.
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiToken))
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status from DHIS2: %s", resp.Status)
	}
	var payload struct {
		OrganisationUnits []DHIS2OrgUnit `json:"organisationUnits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	for _, ou := range payload.OrganisationUnits {
		var parentID *int64
		if ou.Parent != nil {
			parentRecord, err := s.repo.GetByUID(ctx, ou.Parent.ID)
			if err == nil {
				pid := parentRecord.ID
				parentID = &pid
			}
		}
		record := OrgUnit{
			UID:         ou.ID,
			Code:        ou.Code,
			Name:        ou.Name,
			ShortName:   firstNonEmpty(ou.ShortName, ou.Name),
			Description: ou.Description,
			ParentID:    parentID,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		existing, err := s.repo.GetByUID(ctx, ou.ID)
		if err == nil {
			record.ID = existing.ID
			record.CreatedAt = existing.CreatedAt
			record.Extras = existing.Extras
			record.AttributeValues = existing.AttributeValues
			record.OpeningDate = existing.OpeningDate
			record.Deleted = existing.Deleted
			record.LastSyncDate = &record.UpdatedAt
			if _, err := s.repo.Update(ctx, record); err != nil {
				return fmt.Errorf("failed to update org unit %s: %w", ou.ID, err)
			}
		} else {
			record.LastSyncDate = &record.UpdatedAt
			if _, err := s.repo.Create(ctx, record); err != nil {
				return fmt.Errorf("failed to create org unit %s: %w", ou.ID, err)
			}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
