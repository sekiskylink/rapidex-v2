package dhis2

import "time"

type SubmissionInput struct {
	BaseURL           string
	Method            string
	URLSuffix         string
	Headers           map[string]string
	URLParams         map[string]string
	PayloadBody       string
	PayloadFormat     string
	SubmissionBinding string
	UseAsync          bool
}

type SubmissionResult struct {
	HTTPStatus           *int
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ResponseSummary      map[string]any
	ErrorMessage         string
	RemoteJobID          string
	PollURL              string
	RemoteStatus         string
	RemoteResponse       map[string]any
	Async                bool
	Terminal             bool
	Succeeded            bool
}

type PollResult struct {
	StatusCode           *int
	RemoteStatus         string
	TerminalState        string
	ResponseBody         string
	ResponseContentType  string
	ResponseBodyFiltered bool
	ResponseSummary      map[string]any
	ErrorMessage         string
	NextPollAt           *time.Time
	RemoteResponse       map[string]any
}
