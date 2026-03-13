package delivery

import (
	"mime"
	"strings"
	"time"

	"basepro/backend/internal/config"
)

type SubmissionWindowPolicy struct {
	Configured bool
	StartHour  int
	EndHour    int
	Source     string
}

type ResponseFilterPolicy struct {
	Allowed      []string
	AllowUnknown bool
}

func ResolveSubmissionWindow(serverCode string) SubmissionWindowPolicy {
	cfg := config.Get()
	key := strings.ToLower(strings.TrimSpace(serverCode))
	if destination, ok := cfg.Sukumad.SubmissionWindow.Destinations[key]; ok {
		return SubmissionWindowPolicy{
			Configured: destination.StartHour != 0 || destination.EndHour != 0,
			StartHour:  destination.StartHour,
			EndHour:    destination.EndHour,
			Source:     "destination",
		}
	}
	return SubmissionWindowPolicy{
		Configured: cfg.Sukumad.SubmissionWindow.Default.StartHour != 0 || cfg.Sukumad.SubmissionWindow.Default.EndHour != 0,
		StartHour:  cfg.Sukumad.SubmissionWindow.Default.StartHour,
		EndHour:    cfg.Sukumad.SubmissionWindow.Default.EndHour,
		Source:     "global",
	}
}

func (p SubmissionWindowPolicy) Evaluate(now time.Time) (bool, *time.Time) {
	if !p.Configured {
		return true, nil
	}
	currentHour := now.UTC().Hour()
	if p.StartHour < p.EndHour {
		if currentHour >= p.StartHour && currentHour < p.EndHour {
			return true, nil
		}
		next := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), p.StartHour, 0, 0, 0, time.UTC)
		if currentHour >= p.EndHour {
			next = next.Add(24 * time.Hour)
		}
		return false, &next
	}
	if currentHour >= p.StartHour || currentHour < p.EndHour {
		return true, nil
	}
	next := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), p.StartHour, 0, 0, 0, time.UTC)
	return false, &next
}

func ResolveMaxRetries(serverCode string) int {
	cfg := config.Get()
	key := strings.ToLower(strings.TrimSpace(serverCode))
	if destination, ok := cfg.Sukumad.Retry.Destinations[key]; ok {
		return destination.MaxRetries
	}
	return cfg.Sukumad.Retry.Default.MaxRetries
}

func ResolveResponseFilter(serverCode string) ResponseFilterPolicy {
	cfg := config.Get()
	key := strings.ToLower(strings.TrimSpace(serverCode))
	if destination, ok := cfg.Sukumad.ResponseFilter.Destinations[key]; ok {
		return ResponseFilterPolicy{Allowed: append([]string{}, destination.AllowedContentTypes...), AllowUnknown: destination.AllowUnknown}
	}
	return ResponseFilterPolicy{Allowed: append([]string{}, cfg.Sukumad.ResponseFilter.Default.AllowedContentTypes...), AllowUnknown: cfg.Sukumad.ResponseFilter.Default.AllowUnknown}
}

func ShouldAllowContentType(policy ResponseFilterPolicy, contentType string) bool {
	normalized := NormalizeContentType(contentType)
	if normalized == "" {
		return policy.AllowUnknown
	}
	for _, allowed := range policy.Allowed {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		switch {
		case allowed == normalized:
			return true
		case strings.HasSuffix(allowed, "/*") && strings.HasPrefix(normalized, strings.TrimSuffix(allowed, "*")):
			return true
		case strings.HasPrefix(allowed, "application/*+") && strings.HasPrefix(normalized, "application/") && strings.HasSuffix(normalized, strings.TrimPrefix(allowed, "application/*")):
			return true
		}
	}
	return false
}

func NormalizeContentType(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(value))
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}
