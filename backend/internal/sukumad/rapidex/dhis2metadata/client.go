package dhis2metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

type Connection struct {
	BaseURL   string
	Headers   map[string]string
	URLParams map[string]string
}

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{httpClient: httpClient}
}

type DataSet struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	PeriodType      string               `json:"periodType"`
	DataSetElements []DataSetElementItem `json:"dataSetElements"`
}

type DataSetElementItem struct {
	DataElement DataElement `json:"dataElement"`
}

type DataElement struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	ValueType     string        `json:"valueType"`
	CategoryCombo CategoryCombo `json:"categoryCombo"`
}

type CategoryCombo struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	CategoryOptionCombos []CategoryOptionCombo `json:"categoryOptionCombos"`
}

type CategoryOptionCombo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListDataSets(ctx context.Context, conn Connection) ([]DataSet, error) {
	var payload struct {
		DataSets []DataSet `json:"dataSets"`
	}
	if err := c.doJSON(ctx, conn, "dataSets", "/api/dataSets.json", map[string]string{
		"paging": "false",
		"fields": "id,name,periodType",
	}, &payload); err != nil {
		return nil, err
	}
	return payload.DataSets, nil
}

func (c *Client) GetDataSet(ctx context.Context, conn Connection, datasetID string) (DataSet, error) {
	var payload DataSet
	if err := c.doJSON(ctx, conn, "dataSet", "/api/dataSets/"+strings.TrimSpace(datasetID)+".json", map[string]string{
		"fields": "id,name,periodType,dataSetElements[dataElement[id,name,valueType,categoryCombo[id,name,categoryOptionCombos[id,name]]]]",
	}, &payload); err != nil {
		return DataSet{}, err
	}
	return payload, nil
}

func (c *Client) doJSON(ctx context.Context, conn Connection, resource string, path string, query map[string]string, target any) error {
	requestURL, err := buildURL(conn.BaseURL, path, conn.URLParams, query)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build dhis2 metadata request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range conn.Headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform dhis2 metadata request: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read dhis2 metadata response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body := strings.TrimSpace(string(responseBody))
		if body != "" {
			if len(body) > 512 {
				body = body[:512]
			}
			return fmt.Errorf("dhis2 metadata request for %s failed: %s %s (%s)", resource, resp.Status, requestURL, body)
		}
		return fmt.Errorf("dhis2 metadata request for %s failed: %s %s", resource, resp.Status, requestURL)
	}
	if target == nil || len(strings.TrimSpace(string(responseBody))) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode dhis2 metadata response: %w", err)
	}
	return nil
}

func buildURL(baseURL string, path string, defaults map[string]string, query map[string]string) (string, error) {
	parsed, err := neturl.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return "", fmt.Errorf("parse DHIS2 metadata URL: %w", err)
	}
	if strings.TrimSpace(parsed.String()) == "" {
		return "", fmt.Errorf("dhis2 base url is required")
	}
	normalizedPath := normalizeMetadataPath(parsed.Path, path)
	if normalizedPath == "" {
		return "", fmt.Errorf("dhis2 metadata path is required")
	}
	parsed.Path = normalizedPath
	values := parsed.Query()
	for key, value := range defaults {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	for key, value := range query {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func normalizeMetadataPath(basePath string, resourcePath string) string {
	basePath = strings.TrimRight(strings.TrimSpace(basePath), "/")
	resourcePath = strings.TrimSpace(resourcePath)
	if resourcePath == "" {
		return basePath
	}
	if !strings.HasPrefix(resourcePath, "/") {
		resourcePath = "/" + resourcePath
	}
	if basePath == "" {
		return resourcePath
	}
	if strings.HasSuffix(basePath, "/api") && strings.HasPrefix(resourcePath, "/api/") {
		return basePath + strings.TrimPrefix(resourcePath, "/api")
	}
	return basePath + resourcePath
}
