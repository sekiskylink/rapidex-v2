package dhis2

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func interpretSubmission(response *http.Response, body []byte, useAsync bool, baseURL string) SubmissionResult {
	statusCode := response.StatusCode
	contentType := response.Header.Get("Content-Type")
	result := SubmissionResult{
		HTTPStatus:         intPtr(statusCode),
		ResponseBody:       strings.TrimSpace(string(body)),
		ResponseContentType: contentType,
	}

	payload := decodeBody(body)
	result.RemoteResponse = payload
	result.RemoteStatus = detectRemoteStatus(payload)
	result.RemoteJobID = detectRemoteJobID(payload)
	result.PollURL = detectPollURL(response, payload, baseURL)

	pending := isPendingState(result.RemoteStatus)
	failed := isFailedState(result.RemoteStatus)
	succeeded := isSucceededState(result.RemoteStatus)

	if useAsync && (pending || result.PollURL != "" || result.RemoteJobID != "" || statusCode == http.StatusAccepted) {
		result.Async = true
		result.Terminal = false
		if result.RemoteStatus == "" {
			result.RemoteStatus = "pending"
		}
		return result
	}

	if failed || statusCode >= http.StatusBadRequest {
		result.Terminal = true
		result.Succeeded = false
		result.ErrorMessage = firstText(
			stringValue(payload["message"]),
			stringValue(payload["description"]),
			stringValue(payload["error"]),
			http.StatusText(statusCode),
		)
		return result
	}

	result.Terminal = true
	result.Succeeded = succeeded || statusCode < http.StatusMultipleChoices
	if result.RemoteStatus == "" {
		if result.Succeeded {
			result.RemoteStatus = "succeeded"
		} else {
			result.RemoteStatus = "failed"
		}
	}
	return result
}

func interpretPollResponse(response *http.Response, body []byte) PollResult {
	statusCode := response.StatusCode
	payload := decodeBody(body)
	remoteStatus := detectRemoteStatus(payload)

	result := PollResult{
		StatusCode:         intPtr(statusCode),
		ResponseBody:       strings.TrimSpace(string(body)),
		ResponseContentType: response.Header.Get("Content-Type"),
		RemoteStatus:       remoteStatus,
		RemoteResponse:     payload,
	}

	if remoteStatus == "" {
		remoteStatus = detectTaskCollectionStatus(payload)
		result.RemoteStatus = remoteStatus
	}
	if remoteStatus == "" {
		remoteStatus = "polling"
		result.RemoteStatus = remoteStatus
	}

	switch {
	case isFailedState(remoteStatus) || statusCode >= http.StatusBadRequest:
		result.TerminalState = "failed"
		result.ErrorMessage = firstText(
			detectTaskCollectionError(payload),
			stringValue(payload["message"]),
			stringValue(payload["description"]),
			stringValue(payload["error"]),
			http.StatusText(statusCode),
		)
	case isSucceededState(remoteStatus):
		result.TerminalState = "succeeded"
	default:
		next := time.Now().UTC().Add(30 * time.Second)
		result.NextPollAt = &next
	}

	return result
}

func summarizeBody(contentType string, body []byte) map[string]any {
	text := strings.TrimSpace(string(body))
	summary := map[string]any{
		"contentType": strings.TrimSpace(contentType),
		"bodyLength":  len(body),
	}
	if text == "" {
		return summary
	}
	snippet := text
	if len(snippet) > 240 {
		snippet = snippet[:240]
	}
	if looksLikeHTML(contentType, text) {
		summary["looksLikeHTML"] = true
		snippet = stripHTML(snippet)
	}
	summary["snippet"] = strings.TrimSpace(snippet)
	return summary
}

func stripHTML(value string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	cleaned := re.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(cleaned), " ")
}

func looksLikeHTML(contentType string, body string) bool {
	if strings.Contains(strings.ToLower(contentType), "html") {
		return true
	}
	trimmed := strings.ToLower(strings.TrimSpace(body))
	return strings.HasPrefix(trimmed, "<!doctype html") || strings.HasPrefix(trimmed, "<html")
}

func decodeBody(body []byte) map[string]any {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		return map[string]any{}
	}

	var generic any
	if err := json.Unmarshal(body, &generic); err != nil {
		return map[string]any{
			"raw": string(body),
		}
	}

	switch typed := generic.(type) {
	case map[string]any:
		if typed == nil {
			return map[string]any{}
		}
		return typed
	case []any:
		return map[string]any{
			"items": typed,
		}
	default:
		return map[string]any{
			"value": typed,
		}
	}
}

func detectRemoteStatus(payload map[string]any) string {
	for _, candidate := range []string{
		stringValue(payload["status"]),
		stringValue(payload["state"]),
		nestedString(payload, "response", "status"),
		nestedString(payload, "response", "state"),
		nestedString(payload, "jobStatus"),
		nestedString(payload, "taskStatus"),
	} {
		if candidate != "" {
			return normalizeStatus(candidate)
		}
	}
	return ""
}

func detectRemoteJobID(payload map[string]any) string {
	for _, candidate := range []string{
		stringValue(payload["jobId"]),
		stringValue(payload["taskId"]),
		stringValue(payload["id"]),
		stringValue(payload["uid"]),
		nestedString(payload, "response", "jobId"),
		nestedString(payload, "response", "taskId"),
		nestedString(payload, "response", "id"),
		nestedString(payload, "response", "uid"),
	} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func detectPollURL(response *http.Response, payload map[string]any, baseURL string) string {
	location := strings.TrimSpace(response.Header.Get("Location"))
	if location != "" {
		return normalizePollURL(baseURL, location)
	}
	for _, candidate := range []string{
		stringValue(payload["pollUrl"]),
		stringValue(payload["href"]),
		stringValue(payload["url"]),
		stringValue(payload["location"]),
		nestedString(payload, "response", "pollUrl"),
		nestedString(payload, "response", "href"),
		nestedString(payload, "response", "url"),
		nestedString(payload, "response", "location"),
		nestedString(payload, "response", "relativeNotifierEndpoint"),
	} {
		if candidate == "" {
			continue
		}
		if normalized := normalizePollURL(baseURL, candidate); normalized != "" {
			return normalized
		}
	}
	return ""
}

func normalizePollURL(baseURL string, candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
	}
	parsedCandidate, err := url.Parse(candidate)
	if err != nil {
		return ""
	}
	if parsedCandidate.IsAbs() {
		return parsedCandidate.String()
	}
	if strings.TrimSpace(baseURL) == "" {
		return ""
	}
	parsedBase, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return ""
	}
	if strings.HasPrefix(candidate, "/") {
		basePath := strings.TrimRight(parsedBase.Path, "/")
		parsedBase.Path = basePath + candidate
		parsedBase.RawPath = ""
		parsedBase.RawQuery = ""
		parsedBase.Fragment = ""
		return parsedBase.String()
	}
	return parsedBase.ResolveReference(parsedCandidate).String()
}

func detectTaskCollectionStatus(payload map[string]any) string {
	items := taskCollectionItems(payload)
	if len(items) == 0 {
		return ""
	}

	allCompleted := true
	for _, item := range items {
		completed, hasCompleted := boolValue(item["completed"])
		level := strings.ToUpper(stringValue(item["level"]))
		if hasCompleted && !completed {
			allCompleted = false
		}
		if completed && (level == "ERROR" || level == "FATAL") {
			return "failed"
		}
	}
	if allCompleted {
		return "succeeded"
	}
	return "polling"
}

func detectTaskCollectionError(payload map[string]any) string {
	for _, item := range taskCollectionItems(payload) {
		level := strings.ToUpper(stringValue(item["level"]))
		if level == "ERROR" || level == "FATAL" {
			return stringValue(item["message"])
		}
	}
	return ""
}

func taskCollectionItems(payload map[string]any) []map[string]any {
	rawItems, ok := payload["items"].([]any)
	if !ok {
		return nil
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items
}

func nestedString(value map[string]any, keys ...string) string {
	current := any(value)
	for _, key := range keys {
		mapped, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = mapped[key]
	}
	return stringValue(current)
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func boolValue(value any) (bool, bool) {
	boolean, ok := value.(bool)
	return boolean, ok
}

func normalizeStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ok", "success", "succeeded", "complete", "completed", "done":
		return "succeeded"
	case "failed", "failure", "error", "aborted":
		return "failed"
	case "pending", "queued":
		return "pending"
	case "running", "processing", "in_progress", "in progress", "active":
		return "polling"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func isSucceededState(value string) bool {
	return normalizeStatus(value) == "succeeded"
}

func isFailedState(value string) bool {
	return normalizeStatus(value) == "failed"
}

func isPendingState(value string) bool {
	state := normalizeStatus(value)
	return state == "pending" || state == "polling"
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func intPtr(value int) *int {
	return &value
}
