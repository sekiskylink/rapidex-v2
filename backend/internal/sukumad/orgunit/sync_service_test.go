package orgunit

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	sukumadserver "basepro/backend/internal/sukumad/server"
)

type fakeHierarchySyncRepo struct {
	input replaceHierarchyInput
	state SyncState
}

func (f *fakeHierarchySyncRepo) ReplaceHierarchy(_ context.Context, input replaceHierarchyInput) (SyncResult, error) {
	f.input = input
	result := SyncResult{
		ServerCode:          input.ServerCode,
		DryRun:              input.Request.DryRun,
		FullRefresh:         input.Request.FullRefresh,
		DistrictLevelName:   firstNonEmpty(input.ResolvedLevelName, input.Request.DistrictLevelName),
		DistrictLevelCode:   input.Request.DistrictLevelCode,
		ResolvedDistrictUID: input.ResolvedLevelUID,
		ResolvedDistrict:    input.ResolvedLevelName,
		StartedAt:           input.StartedAt,
		CompletedAt:         input.CompletedAt,
		Status:              input.Status,
		LevelsCount:         len(input.Levels),
		GroupsCount:         len(input.Groups),
		AttributesCount:     len(input.Attributes),
		OrgUnitsCount:       len(input.OrgUnits),
		GroupMembersCount:   groupMemberCount(input.GroupMembers),
		ErrorMessage:        input.ErrorMessage,
	}
	return result, nil
}

func (f *fakeHierarchySyncRepo) GetSyncState(context.Context) (SyncState, error) {
	return f.state, nil
}

type fakeHierarchyServerLookup struct {
	record sukumadserver.Record
}

func (f fakeHierarchyServerLookup) GetServerByCode(context.Context, string) (sukumadserver.Record, error) {
	return f.record, nil
}

func (f fakeHierarchyServerLookup) GetServerByUID(context.Context, string) (sukumadserver.Record, error) {
	return f.record, nil
}

func TestHierarchySyncDryRunFetchesAndValidatesDHIS2Metadata(t *testing.T) {
	repo := &fakeHierarchySyncRepo{}
	service := NewHierarchySyncService(repo, fakeHierarchyServerLookup{record: sukumadserver.Record{Code: "dhis2", BaseURL: "https://dhis.test"}}, &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			var body string
			switch req.URL.Path {
			case "/api/organisationUnitLevels.json":
				body = `{"organisationUnitLevels":[{"id":"lvl1","name":"Country","level":1},{"id":"lvl2","name":"District","level":2}]}`
			case "/api/organisationUnitGroups.json":
				body = `{"organisationUnitGroups":[{"id":"grp1","name":"Hospitals","shortName":"Hospitals"}]}`
			case "/api/attributes.json":
				body = `{"attributes":[{"id":"attr1","name":"Facility Code","shortName":"Code","valueType":"TEXT","unique":false,"mandatory":false,"organisationUnitAttribute":true}]}`
			case "/api/organisationUnits.json":
				body = `{"organisationUnits":[
					{"id":"root1","name":"Uganda","level":1,"path":"/root1"},
					{"id":"dist1","name":"Kampala","level":2,"path":"/root1/dist1","parent":{"id":"root1"},"organisationUnitGroups":[{"id":"grp1"}],"attributeValues":[{"attribute":{"id":"attr1"},"value":"KLA"}]}
				]}`
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`not found`)),
					Request:    req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}),
	})

	result, err := service.Sync(context.Background(), SyncRequest{
		ServerCode:        "dhis2",
		FullRefresh:       true,
		DryRun:            true,
		DistrictLevelName: "District",
	})
	if err != nil {
		t.Fatalf("sync hierarchy: %v", err)
	}
	if result.Status != syncStatusSucceeded || !result.DryRun {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	if repo.input.Status != syncStatusSucceeded || len(repo.input.OrgUnits) != 2 || len(repo.input.Groups) != 1 {
		t.Fatalf("unexpected replace input: %+v", repo.input)
	}
	if repo.input.ResolvedLevelName != "District" || repo.input.ResolvedLevelUID != "lvl2" {
		t.Fatalf("expected resolved district level, got %+v", repo.input)
	}
	if repo.input.OrgUnits[1].Path != "/root1/dist1/" {
		t.Fatalf("expected normalized path, got %+v", repo.input.OrgUnits[1])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHierarchySyncPersistsFailureSummaryWithoutMutation(t *testing.T) {
	repo := &fakeHierarchySyncRepo{}
	service := NewHierarchySyncService(repo, fakeHierarchyServerLookup{record: sukumadserver.Record{Code: "dhis2", BaseURL: "http://127.0.0.1:1"}}, &http.Client{Timeout: 10 * time.Millisecond})

	result, err := service.Sync(context.Background(), SyncRequest{
		ServerCode:        "dhis2",
		FullRefresh:       true,
		DryRun:            true,
		DistrictLevelName: "District",
	})
	if err == nil {
		t.Fatal("expected sync failure")
	}
	if repo.input.Status != syncStatusFailed {
		t.Fatalf("expected failed sync status, got %+v", repo.input)
	}
	if result.Status != syncStatusFailed || result.ErrorMessage == "" {
		t.Fatalf("expected failed sync result, got %+v", result)
	}
}
