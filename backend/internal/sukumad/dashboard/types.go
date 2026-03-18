package dashboard

import "time"

type Snapshot struct {
	GeneratedAt  time.Time      `json:"generatedAt"`
	Health       Health         `json:"health"`
	KPIs         KPIs           `json:"kpis"`
	Trends       Trends         `json:"trends"`
	Attention    Attention      `json:"attention"`
	Workers      WorkersSummary `json:"workers"`
	RecentEvents []EventSummary `json:"recentEvents"`
}

type Health struct {
	Status  string   `json:"status"`
	Signals []string `json:"signals"`
}

type KPIs struct {
	RequestsToday            int `json:"requestsToday"`
	PendingRequests          int `json:"pendingRequests"`
	PendingDeliveries        int `json:"pendingDeliveries"`
	RunningDeliveries        int `json:"runningDeliveries"`
	FailedDeliveriesLastHour int `json:"failedDeliveriesLastHour"`
	PollingJobs              int `json:"pollingJobs"`
	IngestBacklog            int `json:"ingestBacklog"`
	HealthyWorkers           int `json:"healthyWorkers"`
	UnhealthyWorkers         int `json:"unhealthyWorkers"`
}

type Trends struct {
	RequestsByHour     []TimeCountPoint   `json:"requestsByHour"`
	DeliveriesByStatus []StatusCountPoint `json:"deliveriesByStatus"`
	JobsByState        []StatusCountPoint `json:"jobsByState"`
	FailuresByServer   []ServerCountPoint `json:"failuresByServer"`
}

type TimeCountPoint struct {
	BucketStart time.Time `json:"bucketStart"`
	Count       int       `json:"count"`
}

type StatusCountPoint struct {
	BucketStart time.Time `json:"bucketStart"`
	Status      string    `json:"status"`
	Count       int       `json:"count"`
}

type ServerCountPoint struct {
	ServerID   int64  `json:"serverId"`
	ServerName string `json:"serverName"`
	Count      int    `json:"count"`
}

type Attention struct {
	FailedDeliveries       DeliveryAttentionList `json:"failedDeliveries"`
	StaleRunningDeliveries DeliveryAttentionList `json:"staleRunningDeliveries"`
	StuckJobs              JobAttentionList      `json:"stuckJobs"`
	RecentIngestFailures   IngestAttentionList   `json:"recentIngestFailures"`
	UnhealthyWorkers       WorkerAttentionList   `json:"unhealthyWorkers"`
}

type DeliveryAttentionList struct {
	Total int                     `json:"total"`
	Items []DeliveryAttentionItem `json:"items"`
}

type JobAttentionList struct {
	Total int                `json:"total"`
	Items []JobAttentionItem `json:"items"`
}

type IngestAttentionList struct {
	Total int                   `json:"total"`
	Items []IngestAttentionItem `json:"items"`
}

type WorkerAttentionList struct {
	Total int                   `json:"total"`
	Items []WorkerAttentionItem `json:"items"`
}

type DeliveryAttentionItem struct {
	ID             int64      `json:"id"`
	UID            string     `json:"uid"`
	RequestID      int64      `json:"requestId"`
	RequestUID     string     `json:"requestUid"`
	ServerID       int64      `json:"serverId"`
	ServerName     string     `json:"serverName"`
	CorrelationID  string     `json:"correlationId"`
	Status         string     `json:"status"`
	ErrorMessage   string     `json:"errorMessage"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	NextEligibleAt *time.Time `json:"nextEligibleAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type JobAttentionItem struct {
	ID            int64      `json:"id"`
	UID           string     `json:"uid"`
	DeliveryID    int64      `json:"deliveryId"`
	DeliveryUID   string     `json:"deliveryUid"`
	RequestID     int64      `json:"requestId"`
	RequestUID    string     `json:"requestUid"`
	CorrelationID string     `json:"correlationId"`
	RemoteJobID   string     `json:"remoteJobId"`
	RemoteStatus  string     `json:"remoteStatus"`
	CurrentState  string     `json:"currentState"`
	NextPollAt    *time.Time `json:"nextPollAt,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type IngestAttentionItem struct {
	ID               int64      `json:"id"`
	UID              string     `json:"uid"`
	OriginalName     string     `json:"originalName"`
	CurrentPath      string     `json:"currentPath"`
	Status           string     `json:"status"`
	LastErrorCode    string     `json:"lastErrorCode"`
	LastErrorMessage string     `json:"lastErrorMessage"`
	RequestID        *int64     `json:"requestId,omitempty"`
	FailedAt         *time.Time `json:"failedAt,omitempty"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type WorkerAttentionItem struct {
	ID              int64      `json:"id"`
	UID             string     `json:"uid"`
	WorkerType      string     `json:"workerType"`
	WorkerName      string     `json:"workerName"`
	Status          string     `json:"status"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt,omitempty"`
	StartedAt       time.Time  `json:"startedAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type WorkersSummary struct {
	HeartbeatFreshnessSeconds int                   `json:"heartbeatFreshnessSeconds"`
	Items                     []WorkerAttentionItem `json:"items"`
}

type EventSummary struct {
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
	Severity      string    `json:"severity"`
	EntityType    string    `json:"entityType"`
	EntityID      int64     `json:"entityId,omitempty"`
	EntityUID     string    `json:"entityUid,omitempty"`
	Summary       string    `json:"summary"`
	CorrelationID string    `json:"correlationId,omitempty"`
	RequestID     *int64    `json:"requestId,omitempty"`
	DeliveryID    *int64    `json:"deliveryId,omitempty"`
	JobID         *int64    `json:"jobId,omitempty"`
	WorkerID      *int64    `json:"workerId,omitempty"`
}

type SourceEvent struct {
	Type          string
	Timestamp     time.Time
	Severity      string
	Message       string
	CorrelationID string
	RequestID     *int64
	RequestUID    string
	DeliveryID    *int64
	DeliveryUID   string
	JobID         *int64
	JobUID        string
	WorkerID      *int64
	WorkerUID     string
	Payload       map[string]any
}

type StreamEvent struct {
	Type          string         `json:"type"`
	Timestamp     time.Time      `json:"timestamp"`
	Severity      string         `json:"severity"`
	EntityType    string         `json:"entityType"`
	EntityID      int64          `json:"entityId,omitempty"`
	EntityUID     string         `json:"entityUid,omitempty"`
	Summary       string         `json:"summary"`
	CorrelationID string         `json:"correlationId,omitempty"`
	RequestID     *int64         `json:"requestId,omitempty"`
	DeliveryID    *int64         `json:"deliveryId,omitempty"`
	JobID         *int64         `json:"jobId,omitempty"`
	ServerID      *int64         `json:"serverId,omitempty"`
	WorkerID      *int64         `json:"workerId,omitempty"`
	Patch         *StreamPatch   `json:"patch,omitempty"`
	Invalidations []string       `json:"invalidations,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
}

type StreamPatch struct {
	KPI   string `json:"kpi"`
	Op    string `json:"op"`
	Value int    `json:"value"`
}
