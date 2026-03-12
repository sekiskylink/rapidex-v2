package async

import (
	"context"
	"time"
)

type StaticPoller struct {
	Result RemotePollResult
	Err    error
}

func (p StaticPoller) Poll(context.Context, Record) (RemotePollResult, error) {
	if p.Result.NextPollAt == nil && p.Result.TerminalState == "" {
		next := time.Now().UTC().Add(time.Minute)
		p.Result.NextPollAt = &next
	}
	return p.Result, p.Err
}
