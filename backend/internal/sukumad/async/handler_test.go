package async

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerListAndPolls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewService(NewRepository())
	task, err := service.CreateTask(nil, CreateInput{
		DeliveryAttemptID: 9,
		RemoteJobID:       "remote-9",
		RemoteStatus:      StatePending,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := service.RecordPoll(nil, RecordPollInput{
		AsyncTaskID:  task.ID,
		RemoteStatus: StatePolling,
		ResponseBody: `{"state":"processing"}`,
	}); err != nil {
		t.Fatalf("record poll: %v", err)
	}

	handler := NewHandler(service)
	router := gin.New()
	router.GET("/jobs", handler.List)
	router.GET("/jobs/:id/polls", handler.ListPolls)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/jobs?page=1&pageSize=25", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	pollW := httptest.NewRecorder()
	router.ServeHTTP(pollW, httptest.NewRequest(http.MethodGet, "/jobs/1/polls?page=1&pageSize=25", nil))
	if pollW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", pollW.Code, pollW.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(pollW.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode polls: %v", err)
	}
	if payload["totalCount"].(float64) != 1 {
		t.Fatalf("unexpected polls payload: %+v", payload)
	}
}
