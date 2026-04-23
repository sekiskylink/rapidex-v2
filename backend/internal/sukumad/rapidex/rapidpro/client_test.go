package rapidpro

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDoJSONIncludesErrorBodySummary(t *testing.T) {
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"fields":["Facility is invalid"]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	})
	err := client.doJSON(context.Background(), Connection{BaseURL: "https://rapidpro.example.com"}, http.MethodGet, "/fields.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected request to fail")
	}
	requestErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected request error, got %T", err)
	}
	if requestErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", requestErr.StatusCode)
	}
	if !strings.Contains(requestErr.Error(), "Facility is invalid") {
		t.Fatalf("expected summarized body in error, got %q", requestErr.Error())
	}
}

func TestDoJSONHandlesEmptyErrorBody(t *testing.T) {
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	})
	err := client.doJSON(context.Background(), Connection{BaseURL: "https://rapidpro.example.com"}, http.MethodGet, "/fields.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected request to fail")
	}
	requestErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected request error, got %T", err)
	}
	if requestErr.Body != "" {
		t.Fatalf("expected empty body summary, got %q", requestErr.Body)
	}
	if requestErr.Error() != "rapidpro request failed with status 400" {
		t.Fatalf("unexpected error string %q", requestErr.Error())
	}
}

func TestSummarizeErrorBodyTruncatesLongResponses(t *testing.T) {
	body := []byte(strings.Repeat("a", maxErrorBodySummaryLength+20) + "\n")
	summary := summarizeErrorBody(body)
	if !strings.HasSuffix(summary, "...") {
		t.Fatalf("expected truncated summary, got %q", summary)
	}
	if len(summary) != maxErrorBodySummaryLength+3 {
		t.Fatalf("unexpected summary length %d", len(summary))
	}
}

func TestUpsertContactSendsEmptyGroupsArrayInsteadOfNull(t *testing.T) {
	var requestBody map[string]any
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			raw, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			if err := json.Unmarshal(raw, &requestBody); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uuid":"contact-1"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := client.UpsertContact(context.Background(), Connection{BaseURL: "https://rapidpro.example.com"}, UpsertContactInput{
		Name: "Alice Reporter",
		URNs: []string{"tel:+256700000001"},
	})
	if err != nil {
		t.Fatalf("upsert contact: %v", err)
	}
	groups, ok := requestBody["groups"].([]any)
	if !ok {
		t.Fatalf("expected groups array, got %#v", requestBody["groups"])
	}
	if len(groups) != 0 {
		t.Fatalf("expected empty groups array, got %#v", groups)
	}
}

func TestUpsertContactUsesUUIDQueryWithoutURNQuery(t *testing.T) {
	var requestQuery string
	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestQuery = req.URL.RawQuery
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uuid":"contact-1"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := client.UpsertContact(context.Background(), Connection{BaseURL: "https://rapidpro.example.com"}, UpsertContactInput{
		UUID: "contact-1",
		Name: "Alice Reporter",
		URNs: []string{"tel:+256700000001"},
	})
	if err != nil {
		t.Fatalf("upsert contact: %v", err)
	}
	if !strings.Contains(requestQuery, "uuid=contact-1") {
		t.Fatalf("expected uuid query, got %q", requestQuery)
	}
	if strings.Contains(requestQuery, "urn=") {
		t.Fatalf("expected urn query to be omitted, got %q", requestQuery)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
