package dhis2

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
)

type Service struct {
	client *Client
}

func NewService(httpClient *http.Client, limiter interface {
	Wait(context.Context, string) error
}) *Service {
	return &Service{client: NewClient(httpClient, limiter)}
}

func (s *Service) Submit(ctx context.Context, input delivery.DispatchInput) (delivery.DispatchResult, error) {
	response, body, err := s.client.Submit(ctx, destinationKeyFromServer(input.Server), SubmissionInput{
		BaseURL:     input.Server.BaseURL,
		Method:      input.Server.HTTPMethod,
		URLSuffix:   input.URLSuffix,
		Headers:     cloneMap(input.Server.Headers),
		URLParams:   cloneMap(input.Server.URLParams),
		PayloadBody: input.PayloadBody,
		UseAsync:    input.Server.UseAsync,
	})
	if err != nil {
		return delivery.DispatchResult{}, err
	}

	interpreted := interpretSubmission(response, body, input.Server.UseAsync, input.Server.BaseURL)
	policy := delivery.ResolveResponseFilter(input.Server.Code)
	if !delivery.ShouldAllowContentType(policy, interpreted.ResponseContentType) {
		interpreted.ResponseBodyFiltered = true
		interpreted.ResponseSummary = summarizeBody(interpreted.ResponseContentType, body)
		interpreted.ResponseBody = ""
		interpreted.RemoteResponse = filteredRemoteResponse(interpreted.ResponseContentType, interpreted.HTTPStatus, interpreted.ResponseSummary)
	}
	return delivery.DispatchResult{
		HTTPStatus:           interpreted.HTTPStatus,
		ResponseBody:         interpreted.ResponseBody,
		ResponseContentType:  delivery.NormalizeContentType(interpreted.ResponseContentType),
		ResponseBodyFiltered: interpreted.ResponseBodyFiltered,
		ResponseSummary:      interpreted.ResponseSummary,
		ErrorMessage:         interpreted.ErrorMessage,
		RemoteJobID:          interpreted.RemoteJobID,
		PollURL:              interpreted.PollURL,
		RemoteStatus:         interpreted.RemoteStatus,
		RemoteResponse:       interpreted.RemoteResponse,
		Async:                interpreted.Async,
		Terminal:             interpreted.Terminal,
		Succeeded:            interpreted.Succeeded,
	}, nil
}

func (s *Service) Poll(ctx context.Context, task asyncjobs.Record) (asyncjobs.RemotePollResult, error) {
	response, body, err := s.client.Poll(ctx, destinationKeyFromTask(task), task.PollURL, nil)
	if err != nil {
		return asyncjobs.RemotePollResult{}, err
	}

	interpreted := interpretPollResponse(response, body)
	policy := delivery.ResolveResponseFilter(task.DestinationCode)
	if !delivery.ShouldAllowContentType(policy, interpreted.ResponseContentType) {
		interpreted.ResponseBodyFiltered = true
		interpreted.ResponseSummary = summarizeBody(interpreted.ResponseContentType, body)
		interpreted.ResponseBody = ""
		interpreted.RemoteResponse = filteredRemoteResponse(interpreted.ResponseContentType, interpreted.StatusCode, interpreted.ResponseSummary)
	}
	return asyncjobs.RemotePollResult{
		StatusCode:           interpreted.StatusCode,
		RemoteStatus:         interpreted.RemoteStatus,
		TerminalState:        strings.TrimSpace(interpreted.TerminalState),
		ResponseBody:         interpreted.ResponseBody,
		ResponseContentType:  delivery.NormalizeContentType(interpreted.ResponseContentType),
		ResponseBodyFiltered: interpreted.ResponseBodyFiltered,
		ErrorMessage:         interpreted.ErrorMessage,
		NextPollAt:           interpreted.NextPollAt,
		RemoteResponse:       interpreted.RemoteResponse,
	}, nil
}

func cloneMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func destinationKeyFromServer(server delivery.ServerSnapshot) string {
	if code := strings.TrimSpace(server.Code); code != "" {
		return strings.ToLower(code)
	}
	if baseURL := strings.TrimSpace(server.BaseURL); baseURL != "" {
		if parsed, err := url.Parse(baseURL); err == nil && parsed.Host != "" {
			return strings.ToLower(parsed.Host)
		}
	}
	return strings.ToLower(strings.TrimSpace(server.Name))
}

func destinationKeyFromPollURL(pollURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(pollURL))
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host)
	}
	return "default"
}

func destinationKeyFromTask(task asyncjobs.Record) string {
	if code := strings.TrimSpace(task.DestinationCode); code != "" {
		return strings.ToLower(code)
	}
	return destinationKeyFromPollURL(task.PollURL)
}

func filteredRemoteResponse(contentType string, statusCode *int, summary map[string]any) map[string]any {
	safe := map[string]any{
		"filtered": true,
		"summary":  summary,
	}
	if normalized := delivery.NormalizeContentType(contentType); normalized != "" {
		safe["contentType"] = normalized
	}
	if statusCode != nil {
		safe["httpStatus"] = *statusCode
	}
	return safe
}
