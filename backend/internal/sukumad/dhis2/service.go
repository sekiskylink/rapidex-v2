package dhis2

import (
	"context"
	"net/http"
	"strings"

	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
)

type Service struct {
	client *Client
}

func NewService(httpClient *http.Client) *Service {
	return &Service{client: NewClient(httpClient)}
}

func (s *Service) Submit(ctx context.Context, input delivery.DispatchInput) (delivery.DispatchResult, error) {
	response, body, err := s.client.Submit(ctx, SubmissionInput{
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

	interpreted := interpretSubmission(response, body, input.Server.UseAsync)
	return delivery.DispatchResult{
		HTTPStatus:     interpreted.HTTPStatus,
		ResponseBody:   interpreted.ResponseBody,
		ErrorMessage:   interpreted.ErrorMessage,
		RemoteJobID:    interpreted.RemoteJobID,
		PollURL:        interpreted.PollURL,
		RemoteStatus:   interpreted.RemoteStatus,
		RemoteResponse: interpreted.RemoteResponse,
		Async:          interpreted.Async,
		Terminal:       interpreted.Terminal,
		Succeeded:      interpreted.Succeeded,
	}, nil
}

func (s *Service) Poll(ctx context.Context, task asyncjobs.Record) (asyncjobs.RemotePollResult, error) {
	response, body, err := s.client.Poll(ctx, task.PollURL, nil)
	if err != nil {
		return asyncjobs.RemotePollResult{}, err
	}

	interpreted := interpretPollResponse(response, body)
	return asyncjobs.RemotePollResult{
		StatusCode:     interpreted.StatusCode,
		RemoteStatus:   interpreted.RemoteStatus,
		TerminalState:  strings.TrimSpace(interpreted.TerminalState),
		ResponseBody:   interpreted.ResponseBody,
		ErrorMessage:   interpreted.ErrorMessage,
		NextPollAt:     interpreted.NextPollAt,
		RemoteResponse: interpreted.RemoteResponse,
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
