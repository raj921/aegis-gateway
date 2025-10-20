package files

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleRead_Success(t *testing.T) {
	adapter := NewAdapter()

	// first write a file
	writeBody := WriteRequest{
		Path:    "/test.txt",
		Content: "Hello, World!",
	}
	writeBodyBytes, _ := json.Marshal(writeBody)
	writeReq := httptest.NewRequest("POST", "/write", bytes.NewReader(writeBodyBytes))
	writeW := httptest.NewRecorder()
	adapter.HandleWrite(writeW, writeReq)

	// now read it
	readBody := ReadRequest{
		Path: "/test.txt",
	}
	readBodyBytes, _ := json.Marshal(readBody)

	req := httptest.NewRequest("POST", "/read", bytes.NewReader(readBodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleRead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ReadResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Path != "/test.txt" {
		t.Errorf("Expected path /test.txt, got %s", resp.Path)
	}
	if resp.Content != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got %s", resp.Content)
	}
}

func TestHandleRead_FileNotFound(t *testing.T) {
	adapter := NewAdapter()

	body := ReadRequest{
		Path: "/nonexistent.txt",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/read", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleRead(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleRead_InvalidJSON(t *testing.T) {
	adapter := NewAdapter()

	req := httptest.NewRequest("POST", "/read", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	adapter.HandleRead(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleRead_MissingPath(t *testing.T) {
	adapter := NewAdapter()

	body := ReadRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/read", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleRead(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleWrite_Success(t *testing.T) {
	adapter := NewAdapter()

	body := WriteRequest{
		Path:    "/test.txt",
		Content: "Test content",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/write", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleWrite(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp WriteResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Path != "/test.txt" {
		t.Errorf("Expected path /test.txt, got %s", resp.Path)
	}
	if resp.Status != "written" {
		t.Errorf("Expected status written, got %s", resp.Status)
	}
}

func TestHandleWrite_InvalidJSON(t *testing.T) {
	adapter := NewAdapter()

	req := httptest.NewRequest("POST", "/write", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	adapter.HandleWrite(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleWrite_MissingPath(t *testing.T) {
	adapter := NewAdapter()

	body := WriteRequest{
		Content: "Test content",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/write", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleWrite(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleWrite_MissingContent(t *testing.T) {
	adapter := NewAdapter()

	body := WriteRequest{
		Path: "/test.txt",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/write", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleWrite(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleWrite_Overwrite(t *testing.T) {
	adapter := NewAdapter()

	// write initial content
	body1 := WriteRequest{
		Path:    "/test.txt",
		Content: "Initial content",
	}
	bodyBytes1, _ := json.Marshal(body1)
	req1 := httptest.NewRequest("POST", "/write", bytes.NewReader(bodyBytes1))
	w1 := httptest.NewRecorder()
	adapter.HandleWrite(w1, req1)

	// overwrite with new content
	body2 := WriteRequest{
		Path:    "/test.txt",
		Content: "Updated content",
	}
	bodyBytes2, _ := json.Marshal(body2)
	req2 := httptest.NewRequest("POST", "/write", bytes.NewReader(bodyBytes2))
	w2 := httptest.NewRecorder()
	adapter.HandleWrite(w2, req2)

	// read and verify
	readBody := ReadRequest{
		Path: "/test.txt",
	}
	readBodyBytes, _ := json.Marshal(readBody)
	readReq := httptest.NewRequest("POST", "/read", bytes.NewReader(readBodyBytes))
	readW := httptest.NewRecorder()
	adapter.HandleRead(readW, readReq)

	var resp ReadResponse
	json.NewDecoder(readW.Body).Decode(&resp)

	if resp.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got %s", resp.Content)
	}
}

func TestHealthEndpoint(t *testing.T) {
	adapter := NewAdapter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	adapter.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %s", resp["status"])
	}
}
