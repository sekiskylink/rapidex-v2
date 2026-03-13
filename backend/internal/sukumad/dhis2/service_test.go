package dhis2

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

type spyLimiter struct {
	keys []string
	err  error
}

func (s *spyLimiter) Wait(_ context.Context, destinationKey string) error {
	s.keys = append(s.keys, destinationKey)
	return s.err
}

func TestServiceSubmitSyncSuccess(t *testing.T) {
	limiter := &spyLimiter{}
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/tracker" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), limiter)

	result, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody: `{"trackedEntity":"123"}`,
		URLSuffix:   "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: http.MethodPost,
			UseAsync:   false,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !result.Terminal || !result.Succeeded || result.Async {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	if !slices.Equal(limiter.keys, []string{"dhis2-ug"}) {
		t.Fatalf("expected limiter wait for destination code, got %+v", limiter.keys)
	}
}

func TestServiceSubmitAsyncResponse(t *testing.T) {
	limiter := &spyLimiter{}
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusAccepted, `{"status":"PENDING","response":{"id":"job-77"}}`, map[string]string{
			"Location": "https://dhis.example.com/tracker/jobs/job-77",
		}), nil
	}), limiter)

	result, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody: `{"trackedEntity":"123"}`,
		URLSuffix:   "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: http.MethodPost,
			UseAsync:   true,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !result.Async || result.RemoteJobID != "job-77" || result.PollURL == "" {
		t.Fatalf("unexpected async result: %+v", result)
	}
	if !slices.Equal(limiter.keys, []string{"dhis2-ug"}) {
		t.Fatalf("expected limiter wait for async submit, got %+v", limiter.keys)
	}
}

func TestServicePollTerminalFailure(t *testing.T) {
	limiter := &spyLimiter{}
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"status":"FAILED","message":"validation failed"}`, nil), nil
	}), limiter)

	result, err := service.Poll(context.Background(), asyncjobs.Record{
		PollURL:         "https://dhis.example.com/tracker/jobs/job-9",
		DestinationCode: "dhis2-ug",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.TerminalState != asyncjobs.StateFailed || result.ErrorMessage == "" {
		t.Fatalf("unexpected poll result: %+v", result)
	}
	if !slices.Equal(limiter.keys, []string{"dhis2-ug"}) {
		t.Fatalf("expected poll limiter key from host, got %+v", limiter.keys)
	}
}

func TestServicePollMalformedBodyKeepsPolling(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusAccepted, `not-json`, nil), nil
	}), nil)

	result, err := service.Poll(context.Background(), asyncjobs.Record{
		PollURL: "https://dhis.example.com/tracker/jobs/job-9",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.TerminalState != "" || result.NextPollAt == nil {
		t.Fatalf("expected polling result for malformed body, got %+v", result)
	}
}

func jsonResponse(status int, body string, headers map[string]string) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	for key, value := range headers {
		header.Set(key, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
