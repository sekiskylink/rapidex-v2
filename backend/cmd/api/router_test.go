package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestHealthEndpoint(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db := sqlx.NewDb(sqlDB, "sqlmock")
	mock.ExpectPing()

	r := newRouter(AppDeps{DB: db, Version: "0.1.0"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status field to be ok, got %q", body["status"])
	}
	if body["db"] != "up" {
		t.Fatalf("expected db field to be up, got %q", body["db"])
	}
	if body["version"] != "0.1.0" {
		t.Fatalf("expected version field to be 0.1.0, got %q", body["version"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestVersionEndpointReturnsBuildMetadata(t *testing.T) {
	r := newRouter(AppDeps{
		Version:   "1.2.3",
		Commit:    "a1b2c3d",
		BuildDate: "2026-03-03T00:00:00Z",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body["version"] != "1.2.3" {
		t.Fatalf("expected version to be 1.2.3, got %q", body["version"])
	}
	if body["commit"] != "a1b2c3d" {
		t.Fatalf("expected commit to be a1b2c3d, got %q", body["commit"])
	}
	if body["buildDate"] != "2026-03-03T00:00:00Z" {
		t.Fatalf("expected buildDate to be 2026-03-03T00:00:00Z, got %q", body["buildDate"])
	}
}
