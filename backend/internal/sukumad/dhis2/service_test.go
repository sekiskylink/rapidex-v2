package dhis2

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"basepro/backend/internal/logging"
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

func TestServiceSubmitAsyncResponseUsesBodyLocation(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
			"httpStatus":"OK",
			"httpStatusCode":200,
			"status":"OK",
			"message":"Tracker job added",
			"response":{
				"id":"cHh2OCTJvRw",
				"location":"https://play.im.dhis2.org/dev/api/tracker/jobs/cHh2OCTJvRw"
			}
		}`, nil), nil
	}), nil)

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
	if !result.Async {
		t.Fatalf("expected async result, got %+v", result)
	}
	if result.RemoteJobID != "cHh2OCTJvRw" {
		t.Fatalf("expected remote job id from response body, got %+v", result)
	}
	if result.PollURL != "https://play.im.dhis2.org/dev/api/tracker/jobs/cHh2OCTJvRw" {
		t.Fatalf("expected poll url from response.location, got %+v", result)
	}
}

func TestServiceSubmitAsyncResponseUsesRelativeNotifierEndpoint(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
			"httpStatus":"OK",
			"httpStatusCode":200,
			"status":"OK",
			"message":"Initiated dataValueImport",
			"response":{
				"name":"dataValueImport",
				"id":"YR1UxOUXmzT",
				"created":"2018-08-20T14:17:28.429",
				"jobType":"DATAVALUE_IMPORT",
				"relativeNotifierEndpoint":"/api/system/tasks/DATAVALUE_IMPORT/YR1UxOUXmzT"
			}
		}`, nil), nil
	}), nil)

	result, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody: `{"dataValues":[]}`,
		URLSuffix:   "/dataValueSets",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://play.im.dhis2.org/dev",
			HTTPMethod: http.MethodPost,
			UseAsync:   true,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !result.Async {
		t.Fatalf("expected async result, got %+v", result)
	}
	if result.RemoteJobID != "YR1UxOUXmzT" {
		t.Fatalf("expected remote job id from response body, got %+v", result)
	}
	if result.PollURL != "https://play.im.dhis2.org/dev/api/system/tasks/DATAVALUE_IMPORT/YR1UxOUXmzT" {
		t.Fatalf("expected resolved poll url from relative notifier endpoint, got %+v", result)
	}
}

func TestServiceSubmitJSONQueryPayloadBuildsURLWithoutBody(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.RawQuery; got != "existing=1&orgUnit=ou-1&trackedEntity=abc" && got != "existing=1&trackedEntity=abc&orgUnit=ou-1" {
			t.Fatalf("unexpected query string: %s", got)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "" {
			t.Fatalf("expected empty body, got %q", string(body))
		}
		if contentType := req.Header.Get("Content-Type"); contentType != "" {
			t.Fatalf("expected no default content type for query submission, got %q", contentType)
		}
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), nil)

	_, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody:       `{"trackedEntity":"abc","orgUnit":"ou-1"}`,
		PayloadFormat:     "json",
		SubmissionBinding: "query",
		URLSuffix:         "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com?existing=1",
			HTTPMethod: http.MethodPost,
			URLParams:  map[string]string{"existing": "1"},
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
}

func TestServiceSubmitTextBodyDefaultsToTextPlain(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "raw text payload" {
			t.Fatalf("unexpected body: %q", string(body))
		}
		if contentType := req.Header.Get("Content-Type"); contentType != "text/plain" {
			t.Fatalf("expected text/plain, got %q", contentType)
		}
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), nil)

	_, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody:       `raw text payload`,
		PayloadFormat:     "text",
		SubmissionBinding: "body",
		URLSuffix:         "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: http.MethodPost,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
}

func TestServiceSubmitTextQueryPayloadBuildsURLWithoutBody(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.RawQuery; got != "existing=1&orgUnit=ou-1&trackedEntity=abc" && got != "existing=1&trackedEntity=abc&orgUnit=ou-1" {
			t.Fatalf("unexpected query string: %s", got)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "" {
			t.Fatalf("expected empty body, got %q", string(body))
		}
		if contentType := req.Header.Get("Content-Type"); contentType != "" {
			t.Fatalf("expected no default content type for query submission, got %q", contentType)
		}
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), nil)

	_, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody:       `trackedEntity=abc&orgUnit=ou-1`,
		PayloadFormat:     "text",
		SubmissionBinding: "query",
		URLSuffix:         "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com?existing=1",
			HTTPMethod: http.MethodPost,
			URLParams:  map[string]string{"existing": "1"},
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
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

func TestServicePollTaskCollectionCompletedSucceeds(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `[{
			"uid":"hpiaeMy7wFX",
			"level":"INFO",
			"category":"DATAVALUE_IMPORT",
			"time":"2015-09-02T07:43:14.595+0000",
			"message":"Import done",
			"completed":true
		}]`, nil), nil
	}), nil)

	result, err := service.Poll(context.Background(), asyncjobs.Record{
		PollURL:         "https://play.im.dhis2.org/api/system/tasks/DATAVALUE_IMPORT/YR1UxOUXmzT",
		DestinationCode: "dhis2-ug",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.TerminalState != asyncjobs.StateSucceeded {
		t.Fatalf("expected completed task collection to succeed, got %+v", result)
	}
	if result.NextPollAt != nil {
		t.Fatalf("expected no next poll for completed task collection, got %+v", result)
	}
}

func TestServiceSubmitFilteredAsyncResponseSanitizesRemoteResponse(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		header := map[string]string{
			"Content-Type": "text/html; charset=utf-8",
			"Location":     "https://dhis.example.com/tracker/jobs/job-77",
		}
		return jsonResponseWithContentType(http.StatusAccepted, `<!doctype html><html><body>Proxy Error</body></html>`, header), nil
	}), nil)

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
	if !result.ResponseBodyFiltered {
		t.Fatalf("expected filtered response, got %+v", result)
	}
	if raw, ok := result.RemoteResponse["raw"]; ok {
		t.Fatalf("expected sanitized remote response, found raw body %v", raw)
	}
	if filtered, ok := result.RemoteResponse["filtered"].(bool); !ok || !filtered {
		t.Fatalf("expected filtered marker in remote response, got %+v", result.RemoteResponse)
	}
}

func TestServicePollFilteredResponseSanitizesRemoteResponse(t *testing.T) {
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponseWithContentType(http.StatusBadGateway, `<!doctype html><html><body>Bad Gateway</body></html>`, map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		}), nil
	}), nil)

	result, err := service.Poll(context.Background(), asyncjobs.Record{
		PollURL:         "https://dhis.example.com/tracker/jobs/job-9",
		DestinationCode: "dhis2-ug",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if !result.ResponseBodyFiltered {
		t.Fatalf("expected filtered poll response, got %+v", result)
	}
	if raw, ok := result.RemoteResponse["raw"]; ok {
		t.Fatalf("expected sanitized poll remote response, found raw body %v", raw)
	}
	if filtered, ok := result.RemoteResponse["filtered"].(bool); !ok || !filtered {
		t.Fatalf("expected filtered marker in poll remote response, got %+v", result.RemoteResponse)
	}
}

func TestServiceSubmitOutboundLoggingDisabledEmitsNoRequestLog(t *testing.T) {
	logOutput := captureDHIS2Logs(t)
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), nil).WithOutboundLoggingConfig(func() OutboundLoggingConfig {
		return OutboundLoggingConfig{Enabled: false, BodyPreviewBytes: 32}
	})

	_, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody: `{"trackedEntity":"123"}`,
		URLSuffix:   "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: http.MethodPost,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if strings.Contains(logOutput.String(), "worker_outbound_request") {
		t.Fatalf("expected no outbound request log, got:\n%s", logOutput.String())
	}
}

func TestServiceSubmitOutboundLoggingSamplesRedactedBodyWithoutConsumingIt(t *testing.T) {
	logOutput := captureDHIS2Logs(t)
	transportSawBody := false
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		transportSawBody = string(body) == `{"trackedEntity":"123","token":"abc123","value":"done"}`
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), nil).WithOutboundLoggingConfig(func() OutboundLoggingConfig {
		return OutboundLoggingConfig{Enabled: true, BodyPreviewBytes: 256}
	})

	_, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody: `{"trackedEntity":"123","token":"abc123","value":"done"}`,
		URLSuffix:   "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://user:pass@dhis.example.com?password=query-secret",
			HTTPMethod: http.MethodPost,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !transportSawBody {
		t.Fatal("expected transport to receive the original request body")
	}
	assertDHIS2LogContains(t, logOutput.String(),
		"worker_outbound_request",
		`"method":"POST"`,
		`"url":"https://dhis.example.com/tracker?password=%5BREDACTED%5D"`,
		`"destination_key":"dhis2-ug"`,
		`"body_bytes":55`,
		`\"token\":\"[REDACTED]\"`,
		`"body_preview_truncated":false`,
	)
	if strings.Contains(logOutput.String(), "abc123") || strings.Contains(logOutput.String(), "query-secret") || strings.Contains(logOutput.String(), "user:pass") {
		t.Fatalf("expected sensitive values to be redacted, got:\n%s", logOutput.String())
	}
}

func TestServiceSubmitOutboundLoggingTruncatesBodyPreview(t *testing.T) {
	logOutput := captureDHIS2Logs(t)
	payload := `{"first":"` + strings.Repeat("a", 40) + `","last":"` + strings.Repeat("z", 40) + `"}`
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"status":"OK","response":{"status":"SUCCESS"}}`, nil), nil
	}), nil).WithOutboundLoggingConfig(func() OutboundLoggingConfig {
		return OutboundLoggingConfig{Enabled: true, BodyPreviewBytes: 40}
	})

	_, err := service.Submit(context.Background(), delivery.DispatchInput{
		PayloadBody: payload,
		URLSuffix:   "/tracker",
		Server: delivery.ServerSnapshot{
			Code:       "dhis2-ug",
			BaseURL:    "https://dhis.example.com",
			HTTPMethod: http.MethodPost,
		},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	assertDHIS2LogContains(t, logOutput.String(), "worker_outbound_request", "bytes omitted", `"body_preview_truncated":true`)
}

func TestServicePollOutboundLoggingIncludesURLAndEmptyBody(t *testing.T) {
	logOutput := captureDHIS2Logs(t)
	service := NewService(newTestHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"status":"SUCCESS"}`, nil), nil
	}), nil).WithOutboundLoggingConfig(func() OutboundLoggingConfig {
		return OutboundLoggingConfig{Enabled: true, BodyPreviewBytes: 64}
	})

	_, err := service.Poll(context.Background(), asyncjobs.Record{
		PollURL:         "https://dhis.example.com/tracker/jobs/job-9",
		DestinationCode: "dhis2-ug",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	assertDHIS2LogContains(t, logOutput.String(),
		"worker_outbound_request",
		`"method":"GET"`,
		`"url":"https://dhis.example.com/tracker/jobs/job-9"`,
		`"destination_key":"dhis2-ug"`,
		`"body_bytes":0`,
	)
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

func captureDHIS2Logs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logOutput bytes.Buffer
	logging.SetOutput(&logOutput)
	logging.ApplyConfig(logging.Config{Level: "info", Format: "json"})
	t.Cleanup(func() {
		logging.SetOutput(nil)
		logging.ApplyConfig(logging.Config{Level: "info", Format: "console"})
	})
	return &logOutput
}

func assertDHIS2LogContains(t *testing.T, logs string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(logs, fragment) {
			t.Fatalf("expected logs to contain %q, got:\n%s", fragment, logs)
		}
	}
}

func jsonResponseWithContentType(status int, body string, headers map[string]string) *http.Response {
	header := make(http.Header)
	for key, value := range headers {
		header.Set(key, value)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
