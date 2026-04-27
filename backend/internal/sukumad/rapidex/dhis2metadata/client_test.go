package dhis2metadata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListDataSetsBuildsURLFromInstanceRoot(t *testing.T) {
	var requestedPath string
	var requestedQuery string
	client := NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		return jsonResponse(http.StatusOK, `{"dataSets":[{"id":"ds1","name":"Dataset 1","periodType":"Monthly"}]}`), nil
	})})
	got, err := client.ListDataSets(context.Background(), Connection{
		BaseURL:   "https://dhis.example.com/dhis",
		URLParams: map[string]string{"locale": "en"},
	})
	if err != nil {
		t.Fatalf("list datasets: %v", err)
	}
	if len(got) != 1 || got[0].ID != "ds1" {
		t.Fatalf("unexpected datasets: %#v", got)
	}
	if requestedPath != "/dhis/api/dataSets.json" {
		t.Fatalf("expected /dhis/api/dataSets.json, got %q", requestedPath)
	}
	if !strings.Contains(requestedQuery, "locale=en") || !strings.Contains(requestedQuery, "paging=false") {
		t.Fatalf("expected merged query params, got %q", requestedQuery)
	}
}

func TestListDataSetsAvoidsDuplicatingAPIPath(t *testing.T) {
	var requestedPath string
	client := NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPath = r.URL.Path
		return jsonResponse(http.StatusOK, `{"dataSets":[]}`), nil
	})})

	if _, err := client.ListDataSets(context.Background(), Connection{BaseURL: "https://dhis.example.com/dhis/api"}); err != nil {
		t.Fatalf("list datasets: %v", err)
	}
	if requestedPath != "/dhis/api/dataSets.json" {
		t.Fatalf("expected /dhis/api/dataSets.json, got %q", requestedPath)
	}
}

func TestGetDataSetIncludesFailingResourceAndURL(t *testing.T) {
	client := NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, "Not Found"), nil
	})})
	_, err := client.GetDataSet(context.Background(), Connection{BaseURL: "https://dhis.example.com/dhis/api"}, "ds1")
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	if !strings.Contains(message, "dataSet") {
		t.Fatalf("expected resource name in error, got %q", message)
	}
	if !strings.Contains(message, "https://dhis.example.com/dhis/api/dataSets/ds1.json") {
		t.Fatalf("expected failing URL in error, got %q", message)
	}
	if !strings.Contains(message, "404") {
		t.Fatalf("expected status in error, got %q", message)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
