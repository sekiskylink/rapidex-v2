package dhis2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"basepro/backend/internal/logging"
	requests "basepro/backend/internal/sukumad/request"
)

type Client struct {
	executor              RequestExecutor
	outboundLoggingConfig func() OutboundLoggingConfig
}

type RequestExecutor interface {
	Do(context.Context, string, *http.Request) (*http.Response, error)
}

type outboundExecutor struct {
	httpClient interface {
		Do(*http.Request) (*http.Response, error)
	}
	limiter interface {
		Wait(context.Context, string) error
	}
}

func (c *Client) WithOutboundLoggingConfig(provider func() OutboundLoggingConfig) *Client {
	if c == nil {
		return c
	}
	c.outboundLoggingConfig = provider
	return c
}

func NewClient(httpClient *http.Client, limiter interface {
	Wait(context.Context, string) error
}) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		executor: outboundExecutor{
			httpClient: httpClient,
			limiter:    limiter,
		},
	}
}

func (c *Client) Submit(ctx context.Context, destinationKey string, input SubmissionInput) (*http.Response, []byte, error) {
	requestURL, body, defaultContentType, err := buildSubmissionRequest(input)
	if err != nil {
		return nil, nil, err
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("build dhis2 submit request: %w", err)
	}
	for key, value := range input.Headers {
		req.Header.Set(key, value)
	}
	if req.Header.Get("Content-Type") == "" && defaultContentType != "" {
		req.Header.Set("Content-Type", defaultContentType)
	}

	c.logOutboundRequest(ctx, destinationKey, req)
	response, err := c.executor.Do(ctx, destinationKey, req)
	if err != nil {
		return nil, nil, fmt.Errorf("submit to dhis2: %w", err)
	}
	defer response.Body.Close()

	bytes, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("read dhis2 submit response: %w", err)
	}
	return response, bytes, nil
}

func (c *Client) Poll(ctx context.Context, destinationKey string, pollURL string, headers map[string]string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(pollURL), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build dhis2 poll request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	c.logOutboundRequest(ctx, destinationKey, req)
	response, err := c.executor.Do(ctx, destinationKey, req)
	if err != nil {
		return nil, nil, fmt.Errorf("poll dhis2 async task: %w", err)
	}
	defer response.Body.Close()

	bytes, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("read dhis2 poll response: %w", err)
	}
	return response, bytes, nil
}

func (e outboundExecutor) Do(ctx context.Context, destinationKey string, req *http.Request) (*http.Response, error) {
	if e.limiter != nil {
		if err := e.limiter.Wait(ctx, destinationKey); err != nil {
			return nil, fmt.Errorf("wait for destination rate limit %q: %w", destinationKey, err)
		}
	}
	return e.httpClient.Do(req)
}

func (c *Client) logOutboundRequest(ctx context.Context, destinationKey string, req *http.Request) {
	if c == nil || c.outboundLoggingConfig == nil || req == nil {
		return
	}
	cfg := c.outboundLoggingConfig()
	if !cfg.Enabled {
		return
	}
	if cfg.BodyPreviewBytes <= 0 {
		cfg.BodyPreviewBytes = 256
	}
	bodyBytes, preview, truncated, bodyUnavailable := outboundBodyPreview(req, cfg.BodyPreviewBytes)
	logging.ForContext(ctx).Info("worker_outbound_request",
		slog.String("method", req.Method),
		slog.String("url", sanitizeOutboundURL(req.URL)),
		slog.String("destination_key", destinationKey),
		slog.Int("body_bytes", bodyBytes),
		slog.String("body_preview", preview),
		slog.Bool("body_preview_truncated", truncated),
		slog.Bool("body_unavailable", bodyUnavailable),
	)
}

func outboundBodyPreview(req *http.Request, limit int) (int, string, bool, bool) {
	if req == nil || req.Body == nil {
		return 0, "", false, false
	}
	if req.GetBody == nil {
		return -1, "", false, true
	}
	body, err := req.GetBody()
	if err != nil {
		return -1, "", false, true
	}
	defer body.Close()

	bytes, err := io.ReadAll(body)
	if err != nil {
		return -1, "", false, true
	}
	safeBody := redactSensitivePreview(normalizePreview(string(bytes)))
	sample, truncated := edgeSample([]byte(safeBody), limit)
	return len(bytes), sample, truncated, false
}

func edgeSample(body []byte, limit int) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	if limit <= 0 {
		limit = 256
	}
	if len(body) <= limit {
		return string(body), false
	}
	if limit < 32 {
		return string(body[:limit]), true
	}
	headLen := limit / 2
	tailLen := limit - headLen
	omitted := len(body) - headLen - tailLen
	return fmt.Sprintf("%s ...[%d bytes omitted]... %s", string(body[:headLen]), omitted, string(body[len(body)-tailLen:])), true
}

func sanitizeOutboundURL(input *url.URL) string {
	if input == nil {
		return ""
	}
	safeURL := *input
	safeURL.User = nil
	values := safeURL.Query()
	for key := range values {
		if isSensitiveKey(key) {
			values.Set(key, "[REDACTED]")
		}
	}
	safeURL.RawQuery = values.Encode()
	return safeURL.String()
}

func normalizePreview(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

func redactSensitivePreview(input string) string {
	if redacted, ok := redactJSONPreview(input); ok {
		return redacted
	}
	input = redactJSONStringFields(input)
	input = redactQueryLikeFields(input)
	return input
}

func redactJSONPreview(input string) (string, bool) {
	var parsed any
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return "", false
	}
	redacted := redactJSONValue(parsed)
	bytes, err := json.Marshal(redacted)
	if err != nil {
		return "", false
	}
	return string(bytes), true
}

func redactJSONValue(input any) any {
	switch typed := input.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, value := range typed {
			if isSensitiveKey(key) {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = redactJSONValue(value)
		}
		return redacted
	case []any:
		redacted := make([]any, 0, len(typed))
		for _, value := range typed {
			redacted = append(redacted, redactJSONValue(value))
		}
		return redacted
	default:
		return input
	}
}

var (
	jsonSensitiveFieldPattern  = regexp.MustCompile(`(?i)"(password|token|authorization|apiKey|api_key|secret)"\s*:\s*"[^"]*"`)
	querySensitiveFieldPattern = regexp.MustCompile(`(?i)\b(password|token|authorization|apiKey|api_key|secret)=([^&\s]+)`)
)

func redactJSONStringFields(input string) string {
	return jsonSensitiveFieldPattern.ReplaceAllStringFunc(input, func(match string) string {
		field := match[:strings.Index(match, ":")]
		return field + `:"[REDACTED]"`
	})
}

func redactQueryLikeFields(input string) string {
	return querySensitiveFieldPattern.ReplaceAllString(input, `$1=[REDACTED]`)
}

func isSensitiveKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "password", "token", "authorization", "apikey", "api_key", "secret":
		return true
	default:
		return false
	}
}

func buildSubmissionRequest(input SubmissionInput) (string, *bytes.Buffer, string, error) {
	queryParams := cloneURLParams(input.URLParams)
	body := bytes.NewBuffer(nil)
	defaultContentType := ""

	switch input.SubmissionBinding {
	case "", requests.SubmissionBindingBody:
		body.WriteString(strings.TrimSpace(input.PayloadBody))
		switch input.PayloadFormat {
		case "", requests.PayloadFormatJSON:
			defaultContentType = "application/json"
		case requests.PayloadFormatText:
			defaultContentType = "text/plain"
		default:
			return "", nil, "", fmt.Errorf("unsupported payload format %q", input.PayloadFormat)
		}
	case requests.SubmissionBindingQuery:
		values, err := payloadQueryParams(input.PayloadBody, input.PayloadFormat)
		if err != nil {
			return "", nil, "", err
		}
		mergeURLValues(queryParams, values)
	default:
		return "", nil, "", fmt.Errorf("unsupported submission binding %q", input.SubmissionBinding)
	}

	requestURL, err := resolveURL(input.BaseURL, input.URLSuffix, queryParams)
	if err != nil {
		return "", nil, "", err
	}
	return requestURL, body, defaultContentType, nil
}

func resolveURL(baseURL string, suffix string, params url.Values) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid dhis2 base url: %w", err)
	}
	if suffix != "" {
		if !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		parsed.Path = strings.TrimRight(parsed.Path, "/") + suffix
	}
	query := parsed.Query()
	for key, list := range params {
		query.Del(key)
		for _, value := range list {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func cloneURLParams(input map[string]string) url.Values {
	if len(input) == 0 {
		return url.Values{}
	}
	cloned := url.Values{}
	for key, value := range input {
		cloned.Set(key, value)
	}
	return cloned
}

func payloadQueryParams(payloadBody string, payloadFormat string) (url.Values, error) {
	switch payloadFormat {
	case "", requests.PayloadFormatJSON:
		return jsonPayloadQueryParams(payloadBody)
	case requests.PayloadFormatText:
		values, err := url.ParseQuery(strings.TrimSpace(payloadBody))
		if err != nil {
			return nil, fmt.Errorf("invalid text query params: %w", err)
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported payload format %q", payloadFormat)
	}
}

func jsonPayloadQueryParams(payloadBody string) (url.Values, error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payloadBody)), &parsed); err != nil {
		return nil, fmt.Errorf("invalid json query params: %w", err)
	}
	values := url.Values{}
	for key, value := range parsed {
		switch typed := value.(type) {
		case nil:
			values.Add(key, "")
		case []any:
			for _, item := range typed {
				values.Add(key, queryParamString(item))
			}
		default:
			values.Add(key, queryParamString(typed))
		}
	}
	return values, nil
}

func queryParamString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(bytes)
	}
}

func mergeURLValues(params url.Values, values url.Values) {
	for key, list := range values {
		params.Del(key)
		for _, value := range list {
			params.Add(key, value)
		}
	}
}
