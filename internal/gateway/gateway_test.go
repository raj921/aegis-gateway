package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aegis-gateway/pkg/telemetry"
)

func setupTestGateway(t *testing.T) (*Gateway, string) {
	tmpDir := t.TempDir()
	
	// initialize telemetry for tests
	logPath := filepath.Join(tmpDir, "test-audit.log")
	if err := telemetry.InitTelemetry("aegis-test", logPath); err != nil {
		t.Fatalf("Failed to initialize telemetry: %v", err)
	}
	
	// create test policy
	policyContent := `version: 1
agents:
  - id: test-agent
    allow:
      - tool: payments
        actions: [create]
        conditions:
          max_amount: 5000
`
	
	policyPath := filepath.Join(tmpDir, "test-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write test policy: %v", err)
	}

	// create mock adapter server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"payment_id": "test-123",
			"status":     "created",
		})
	}))

	adapters := map[string]string{
		"payments": mockServer.URL,
	}

	gw, err := NewGateway(tmpDir, adapters)
	if err != nil {
		t.Fatalf("Failed to create gateway: %v", err)
	}

	return gw, mockServer.URL
}

func TestHandleToolRequest_MissingAgentID(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	body := map[string]interface{}{
		"amount":   1000.0,
		"currency": "USD",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "MissingHeader" {
		t.Errorf("Expected MissingHeader error, got %s", resp.Error)
	}
}

func TestHandleToolRequest_InvalidJSON(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	req := httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "InvalidRequest" {
		t.Errorf("Expected InvalidRequest error, got %s", resp.Error)
	}
}

func TestHandleToolRequest_PolicyViolation(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	body := map[string]interface{}{
		"amount":   10000.0, // exceeds max_amount of 5000
		"currency": "USD",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader(bodyBytes))
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "PolicyViolation" {
		t.Errorf("Expected PolicyViolation error, got %s", resp.Error)
	}
}

func TestHandleToolRequest_Success(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	body := map[string]interface{}{
		"amount":   1000.0,
		"currency": "USD",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader(bodyBytes))
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["payment_id"] != "test-123" {
		t.Errorf("Expected payment_id test-123, got %v", resp["payment_id"])
	}
}

func TestHandleToolRequest_UnknownAdapter(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	body := map[string]interface{}{
		"path": "/test.txt",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tools/unknown-tool/action", bytes.NewReader(bodyBytes))
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	// Policy check happens first, so we get 403 for unknown tool
	// This is correct behavior - policy enforcement before adapter lookup
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "PolicyViolation" {
		t.Errorf("Expected PolicyViolation error, got %s", resp.Error)
	}
}

func TestHealthEndpoint(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	gw.handle_health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %s", resp["status"])
	}
}

func TestReloadEndpoint(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	req := httptest.NewRequest("POST", "/policies/reload", nil)
	w := httptest.NewRecorder()

	gw.handle_reload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "reloaded" {
		t.Errorf("Expected status reloaded, got %s", resp["status"])
	}
}

func TestPolicyHotReload(t *testing.T) {
	tmpDir := t.TempDir()
	
	// create initial policy
	initialPolicy := `version: 1
agents:
  - id: test-agent
    allow:
      - tool: payments
        actions: [create]
        conditions:
          max_amount: 5000
`
	
	policyPath := filepath.Join(tmpDir, "test-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(initialPolicy), 0644); err != nil {
		t.Fatalf("Failed to write initial policy: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"payment_id": "test-123",
			"status":     "created",
		})
	}))
	defer mockServer.Close()

	adapters := map[string]string{
		"payments": mockServer.URL,
	}

	gw, err := NewGateway(tmpDir, adapters)
	if err != nil {
		t.Fatalf("Failed to create gateway: %v", err)
	}
	defer gw.Close()

	// test with initial policy - should allow 3000
	body := map[string]interface{}{
		"amount":   3000.0,
		"currency": "USD",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader(bodyBytes))
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Initial policy: Expected status 200, got %d", w.Code)
	}

	// update policy to lower max_amount
	updatedPolicy := `version: 2
agents:
  - id: test-agent
    allow:
      - tool: payments
        actions: [create]
        conditions:
          max_amount: 2000
`
	
	if err := os.WriteFile(policyPath, []byte(updatedPolicy), 0644); err != nil {
		t.Fatalf("Failed to write updated policy: %v", err)
	}

	// give the file watcher time to detect the change
	time.Sleep(100 * time.Millisecond)

	// manually reload for test (in real scenario, watcher does this)
	gw.policyManager.Reload()

	// test with updated policy - should now deny 3000
	req = httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader(bodyBytes))
	req.Header.Set("X-Agent-ID", "test-agent")
	w = httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Updated policy: Expected status 403, got %d", w.Code)
	}
}

func TestParentAgentHeader(t *testing.T) {
	gw, _ := setupTestGateway(t)
	defer gw.Close()

	body := map[string]interface{}{
		"amount":   1000.0,
		"currency": "USD",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tools/payments/create", bytes.NewReader(bodyBytes))
	req.Header.Set("X-Agent-ID", "test-agent")
	req.Header.Set("X-Parent-Agent", "parent-agent")
	w := httptest.NewRecorder()

	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	// Parent agent header is captured in telemetry
}
