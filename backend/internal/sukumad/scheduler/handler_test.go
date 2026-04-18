package scheduler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerListJobsAndRunNow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewService(NewRepository())
	job, err := service.CreateScheduledJob(nil, CreateInput{
		Code:         "nightly-sync",
		Name:         "Nightly Sync",
		JobCategory:  JobCategoryIntegration,
		JobType:      "dhis2.sync",
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config:       map[string]any{"serverCode": "dhis2"},
	})
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}

	handler := NewHandler(service)
	router := gin.New()
	router.GET("/scheduler/jobs", handler.ListJobs)
	router.POST("/scheduler/jobs/:id/run-now", handler.RunNow)

	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, httptest.NewRequest(http.MethodGet, "/scheduler/jobs?page=1&pageSize=25", nil))
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listW.Code, listW.Body.String())
	}

	var listPayload map[string]any
	if err := json.Unmarshal(listW.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list payload: %v", err)
	}
	if listPayload["totalCount"].(float64) != 1 {
		t.Fatalf("unexpected list payload: %+v", listPayload)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/scheduler/jobs/1/run-now", nil)
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	router.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without principal, got %d body=%s", runW.Code, runW.Body.String())
	}

	if job.ID != 1 {
		t.Fatalf("expected seeded job ID 1, got %d", job.ID)
	}
}

func TestHandlerCreateJobValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(NewService(NewRepository()))
	router := gin.New()
	router.POST("/scheduler/jobs", handler.CreateJob)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/scheduler/jobs", strings.NewReader(`{"code":"","name":"Bad"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without principal, got %d", w.Code)
	}
}
