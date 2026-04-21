package rapidpro

import (
	"bytes"
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

type Group struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type Contact struct {
	UUID   string   `json:"uuid"`
	Name   string   `json:"name"`
	URNs   []string `json:"urns"`
	Groups []Group  `json:"groups"`
}

type Message struct {
	ID      int64   `json:"id"`
	Text    string  `json:"text"`
	Contact Contact `json:"contact"`
}

type Broadcast struct {
	ID       int64     `json:"id"`
	Contacts []Contact `json:"contacts"`
}

type UpsertContactInput struct {
	UUID   string
	URN    string
	Name   string
	URNs   []string
	Groups []string
}

type listResponse[T any] struct {
	Results []T `json:"results"`
}

func (c *Client) LookupContactByUUID(ctx context.Context, conn Connection, uuid string) (Contact, bool, error) {
	contacts, err := c.listContacts(ctx, conn, map[string]string{"uuid": strings.TrimSpace(uuid)})
	if err != nil {
		return Contact{}, false, err
	}
	if len(contacts) == 0 {
		return Contact{}, false, nil
	}
	return contacts[0], true, nil
}

func (c *Client) LookupContactByURN(ctx context.Context, conn Connection, urn string) (Contact, bool, error) {
	contacts, err := c.listContacts(ctx, conn, map[string]string{"urn": strings.TrimSpace(urn)})
	if err != nil {
		return Contact{}, false, err
	}
	if len(contacts) == 0 {
		return Contact{}, false, nil
	}
	return contacts[0], true, nil
}

func (c *Client) UpsertContact(ctx context.Context, conn Connection, input UpsertContactInput) (Contact, error) {
	body := map[string]any{
		"name":   strings.TrimSpace(input.Name),
		"urns":   normalizeStrings(input.URNs),
		"groups": normalizeStrings(input.Groups),
	}
	query := map[string]string{}
	if uuid := strings.TrimSpace(input.UUID); uuid != "" {
		query["uuid"] = uuid
	} else if urn := strings.TrimSpace(input.URN); urn != "" {
		query["urn"] = urn
	}
	var contact Contact
	if err := c.doJSON(ctx, conn, http.MethodPost, "/contacts.json", query, body, &contact); err != nil {
		return Contact{}, err
	}
	return contact, nil
}

func (c *Client) EnsureGroup(ctx context.Context, conn Connection, name string) (Group, bool, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return Group{}, false, fmt.Errorf("group name is required")
	}
	groups, err := c.listGroups(ctx, conn)
	if err != nil {
		return Group{}, false, err
	}
	for _, group := range groups {
		if strings.EqualFold(strings.TrimSpace(group.Name), normalized) {
			return group, false, nil
		}
	}
	var created Group
	if err := c.doJSON(ctx, conn, http.MethodPost, "/groups.json", nil, map[string]string{"name": normalized}, &created); err != nil {
		return Group{}, false, err
	}
	return created, true, nil
}

func (c *Client) SendMessage(ctx context.Context, conn Connection, contactUUID string, text string) (Message, error) {
	var message Message
	if err := c.doJSON(ctx, conn, http.MethodPost, "/messages.json", nil, map[string]any{
		"contact": strings.TrimSpace(contactUUID),
		"text":    strings.TrimSpace(text),
	}, &message); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (c *Client) SendBroadcast(ctx context.Context, conn Connection, contactUUIDs []string, text string) (Broadcast, error) {
	var broadcast Broadcast
	if err := c.doJSON(ctx, conn, http.MethodPost, "/broadcasts.json", nil, map[string]any{
		"contacts": normalizeStrings(contactUUIDs),
		"text": map[string]string{
			"eng": strings.TrimSpace(text),
		},
	}, &broadcast); err != nil {
		return Broadcast{}, err
	}
	return broadcast, nil
}

func (c *Client) listContacts(ctx context.Context, conn Connection, query map[string]string) ([]Contact, error) {
	var response listResponse[Contact]
	if err := c.doJSON(ctx, conn, http.MethodGet, "/contacts.json", query, nil, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (c *Client) listGroups(ctx context.Context, conn Connection) ([]Group, error) {
	var response listResponse[Group]
	if err := c.doJSON(ctx, conn, http.MethodGet, "/groups.json", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (c *Client) doJSON(ctx context.Context, conn Connection, method string, path string, query map[string]string, payload any, target any) error {
	requestURL, err := buildURL(conn.BaseURL, path, query)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		raw, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return fmt.Errorf("marshal RapidPro request: %w", marshalErr)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return fmt.Errorf("build RapidPro request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("perform RapidPro request: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read RapidPro response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("rapidpro request failed with status %d", resp.StatusCode)
	}
	if target == nil || len(bytes.TrimSpace(responseBody)) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode RapidPro response: %w", err)
	}
	return nil
}

func buildURL(baseURL string, path string, query map[string]string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("rapidpro base url is required")
	}
	if strings.HasSuffix(base, "/api/v2") {
		base += path
	} else {
		base += "/api/v2" + path
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse RapidPro URL: %w", err)
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

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
