package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHealthEndpointDoesNotLeakSensitiveData(t *testing.T) {
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

	bodyText := strings.ToLower(w.Body.String())
	for _, forbidden := range []string{"password", "secret", "token", "jwt", "env", "dsn", "authorization"} {
		if strings.Contains(bodyText, forbidden) {
			t.Fatalf("health response leaked forbidden term %q: %s", forbidden, w.Body.String())
		}
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["status"]; !ok {
		t.Fatal("expected health status field")
	}
	if _, ok := body["db"]; !ok {
		t.Fatal("expected health db field")
	}
	if _, ok := body["version"]; !ok {
		t.Fatal("expected health version field")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestOpenAPISpecEndpointServesYAML(t *testing.T) {
	r := newRouter(AppDeps{})

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "application/yaml") {
		t.Fatalf("expected yaml content type, got %q", contentType)
	}

	body := w.Body.String()
	for _, expected := range []string{"openapi: 3.0.3", "title: SukumadPro API", "/auth/login:", "/observability/trace:"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected response to contain %q", expected)
		}
	}
}

func TestDocsEndpointServesScalarUI(t *testing.T) {
	r := newRouter(AppDeps{})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected html content type, got %q", contentType)
	}

	body := w.Body.String()
	for _, expected := range []string{"@scalar/api-reference", `data-url="/openapi.yaml"`, "<title>SukumadPro API Docs</title>"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected response to contain %q", expected)
		}
	}
}
