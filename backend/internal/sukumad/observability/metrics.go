package observability

type MetricsSnapshot struct {
	Workers    int `json:"workers"`
	RateLimits int `json:"rateLimits"`
}
