package dhis2metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Connection struct {
	BaseURL string
	Headers map[string]string
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
	ID        string `json:"id"`
	Name      string `json:"name"`
	ValueType string `json:"valueType"`
}

type CategoryOptionCombo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AttributeOptionCombo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListDataSets(ctx context.Context, conn Connection) ([]DataSet, error) {
	var payload struct {
		DataSets []DataSet `json:"dataSets"`
	}
	if err := c.doJSON(ctx, conn, "/api/dataSets.json", map[string]string{
		"paging": "false",
		"fields": "id,name,periodType,dataSetElements[dataElement[id,name,valueType]]",
	}, &payload); err != nil {
		return nil, err
	}
	return payload.DataSets, nil
}

func (c *Client) ListDataElements(ctx context.Context, conn Connection) ([]DataElement, error) {
	var payload struct {
		DataElements []DataElement `json:"dataElements"`
	}
	if err := c.doJSON(ctx, conn, "/api/dataElements.json", map[string]string{
		"paging": "false",
		"fields": "id,name,valueType",
	}, &payload); err != nil {
		return nil, err
	}
	return payload.DataElements, nil
}

func (c *Client) ListCategoryOptionCombos(ctx context.Context, conn Connection) ([]CategoryOptionCombo, error) {
	var payload struct {
		CategoryOptionCombos []CategoryOptionCombo `json:"categoryOptionCombos"`
	}
	if err := c.doJSON(ctx, conn, "/api/categoryOptionCombos.json", map[string]string{
		"paging": "false",
		"fields": "id,name",
	}, &payload); err != nil {
		return nil, err
	}
	return payload.CategoryOptionCombos, nil
}

func (c *Client) ListAttributeOptionCombos(ctx context.Context, conn Connection) ([]AttributeOptionCombo, error) {
	var payload struct {
		AttributeOptionCombos []AttributeOptionCombo `json:"attributeOptionCombos"`
	}
	if err := c.doJSON(ctx, conn, "/api/attributeOptionCombos.json", map[string]string{
		"paging": "false",
		"fields": "id,name",
	}, &payload); err != nil {
		return nil, err
	}
	return payload.AttributeOptionCombos, nil
}

func (c *Client) doJSON(ctx context.Context, conn Connection, path string, query map[string]string, target any) error {
	requestURL, err := buildURL(conn.BaseURL, path, query)
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
		return fmt.Errorf("dhis2 metadata request failed with status %d", resp.StatusCode)
	}
	if target == nil || len(strings.TrimSpace(string(responseBody))) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode dhis2 metadata response: %w", err)
	}
	return nil
}

func buildURL(baseURL string, path string, query map[string]string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("dhis2 base url is required")
	}
	base += path
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse DHIS2 metadata URL: %w", err)
	}
	values := parsed.Query()
	for key, value := range query {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}
