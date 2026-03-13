package dhis2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	requestURL, err := resolveURL(input.BaseURL, input.URLSuffix, input.URLParams)
	if err != nil {
		return nil, nil, err
	}

	body := bytes.NewBufferString(strings.TrimSpace(input.PayloadBody))
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("build dhis2 submit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range input.Headers {
		req.Header.Set(key, value)
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

func resolveURL(baseURL string, suffix string, params map[string]string) (string, error) {
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
	for key, value := range params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
