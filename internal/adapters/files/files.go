package files

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ReadRequest struct {
	Path string `json:"path"`
}

type ReadResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WriteResponse struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type Adapter struct {
	mu    sync.RWMutex
	files map[string]string
}

func NewAdapter() *Adapter {
	a := &Adapter{
		files: make(map[string]string),
	}
	a.files["/hr-docs/employee-handbook.pdf"] = "Employee handbook content..."
	a.files["/hr-docs/benefits.pdf"] = "Benefits information..."
	a.files["/legal/contract.docx"] = "Legal contract content..."
	return a
}

func (a *Adapter) HandleRead(w http.ResponseWriter, r *http.Request) {
	var req ReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"InvalidRequest","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, `{"error":"InvalidRequest","message":"Path is required"}`, http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	content, exists := a.files[req.Path]
	a.mu.RUnlock()

	if !exists {
		http.Error(w, `{"error":"NotFound","message":"File not found"}`, http.StatusNotFound)
		return
	}

	resp := ReadResponse{
		Path:    req.Path,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *Adapter) HandleWrite(w http.ResponseWriter, r *http.Request) {
	var req WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"InvalidRequest","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, `{"error":"InvalidRequest","message":"Path is required"}`, http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, `{"error":"InvalidRequest","message":"Content is required"}`, http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	a.files[req.Path] = req.Content
	a.mu.Unlock()

	resp := WriteResponse{
		Path:   req.Path,
		Status: "written",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *Adapter) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (a *Adapter) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/read", a.HandleRead)
	mux.HandleFunc("/write", a.HandleWrite)
	mux.HandleFunc("/health", a.HandleHealth)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("Files adapter listening on %s\n", addr)
	return server.ListenAndServe()
}
