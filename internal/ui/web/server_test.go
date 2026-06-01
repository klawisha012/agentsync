package web_test

import (
	"agentsync/internal/domain"
	"agentsync/internal/ui/web"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAPIHandlers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentsync-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Тестируем GET /api/status
	req, err := http.NewRequest("GET", "/api/status", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(web.TestApiStatusHandlerWrapper(tmpDir))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var agents []struct {
		Name        string   `json:"name"`
		Detected    bool     `json:"detected"`
		ConfigPaths []string `json:"config_paths"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &agents); err != nil {
		t.Errorf("handler returned invalid JSON: %v", err)
	}

	// Тестируем GET /api/components
	reqComp, err := http.NewRequest("GET", "/api/components", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rrComp := httptest.NewRecorder()
	handlerComp := http.HandlerFunc(web.TestApiComponentsHandlerWrapper(tmpDir))
	handlerComp.ServeHTTP(rrComp, reqComp)

	if status := rrComp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var components []domain.AgentComponent
	if err := json.Unmarshal(rrComp.Body.Bytes(), &components); err != nil {
		t.Errorf("handler returned invalid JSON for components: %v", err)
	}
}
