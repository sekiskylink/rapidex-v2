package dhis2

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func interpretSubmission(response *http.Response, body []byte, useAsync bool) SubmissionResult {
	statusCode := response.StatusCode
	result := SubmissionResult{
		HTTPStatus:   intPtr(statusCode),
		ResponseBody: strings.TrimSpace(string(body)),
	}

	payload := decodeBody(body)
	result.RemoteResponse = payload
	result.RemoteStatus = detectRemoteStatus(payload)
	result.RemoteJobID = detectRemoteJobID(payload)
	result.PollURL = detectPollURL(response, payload)

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
	if remoteStatus == "" {
		remoteStatus = "polling"
	}

	result := PollResult{
		StatusCode:     intPtr(statusCode),
		ResponseBody:   strings.TrimSpace(string(body)),
		RemoteStatus:   remoteStatus,
		RemoteResponse: payload,
	}

	switch {
	case isFailedState(remoteStatus) || statusCode >= http.StatusBadRequest:
		result.TerminalState = "failed"
		result.ErrorMessage = firstText(
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

func decodeBody(body []byte) map[string]any {
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return map[string]any{
			"raw": string(body),
		}
	}
	if payload == nil {
		return map[string]any{}
	}
	return payload
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

func detectPollURL(response *http.Response, payload map[string]any) string {
	location := strings.TrimSpace(response.Header.Get("Location"))
	if location != "" {
		return location
	}
	for _, candidate := range []string{
		stringValue(payload["pollUrl"]),
		stringValue(payload["href"]),
		stringValue(payload["url"]),
		nestedString(payload, "response", "pollUrl"),
		nestedString(payload, "response", "href"),
		nestedString(payload, "response", "url"),
	} {
		if candidate == "" {
			continue
		}
		if _, err := url.Parse(candidate); err == nil {
			return candidate
		}
	}
	return ""
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
