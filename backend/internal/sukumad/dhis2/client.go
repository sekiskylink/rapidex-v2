package dhis2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	requests "basepro/backend/internal/sukumad/request"
)

type Client struct {
	executor RequestExecutor
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
