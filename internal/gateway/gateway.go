package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aegis-gateway/internal/policy"
	"aegis-gateway/pkg/telemetry"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
)

// main gateway struct
type Gateway struct {
	policyManager *policy.Manager
	router        *mux.Router
	adapters      map[string]string  // tool name -> URL
	watcher       *fsnotify.Watcher
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
}

func NewGateway(policyDir string, adapters map[string]string) (*Gateway, error) {
	pm, err := policy.NewManager(policyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy manager: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := watcher.Add(policyDir); err != nil {
		return nil, fmt.Errorf("failed to watch policy directory: %w", err)
	}

	g := &Gateway{
		policyManager: pm,
		router:        mux.NewRouter(),
		adapters:      adapters,
		watcher:       watcher,
	}

	g.setupRoutes()
	go g.watchPolicies()

	return g, nil
}

func (g *Gateway) setupRoutes() {
	// main tool execution endpoint
	g.router.HandleFunc("/tools/{tool}/{action}", g.handleToolRequest).Methods("POST")
	
	// admin endpoints
	g.router.HandleFunc("/health", g.handle_health).Methods("GET")
	g.router.HandleFunc("/policies/reload", g.handle_reload).Methods("POST")
}

func (g *Gateway) handle_health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (g *Gateway) handle_reload(w http.ResponseWriter, r *http.Request) {
	err := g.policyManager.Reload()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"ReloadFailed","message":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
}

// watch for policy file changes and auto-reload
func (g *Gateway) watchPolicies() {
	for {
		select {
		case event, ok := <-g.watcher.Events:
			if !ok {
				return
			}
			// reload on write or create
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Printf("Policy file changed: %s, reloading...\n", event.Name)
				err := g.policyManager.Reload()
				if err != nil {
					fmt.Printf("ERROR: failed to reload policies: %v\n", err)
				} else {
					fmt.Println("Policies reloaded successfully")
				}
			}
		case err, ok := <-g.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("ERROR: watcher error: %v\n", err)
		}
	}
}

// main handler for tool execution requests
func (g *Gateway) handleToolRequest(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	ctx, span := telemetry.StartSpan(ctx, "gateway.handleToolRequest")
	defer span.End()

	// extract tool and action from URL
	vars := mux.Vars(r)
	toolName := vars["tool"]
	actionName := vars["action"]

	agentID := r.Header.Get("X-Agent-ID")
	parentAgent := r.Header.Get("X-Parent-Agent")

	// agent ID is required
	if agentID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "MissingHeader",
			Reason: "X-Agent-ID header is required",
		})
		return
	}

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "InvalidRequest",
			Reason: "Failed to read request body",
		})
		return
	}

	var requestParams map[string]interface{}
	if err := json.Unmarshal(requestBody, &requestParams); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "InvalidRequest",
			Reason: "Request body must be valid JSON",
		})
		return
	}

	paramsHash := policy.HashParams(requestParams)

	// evaluate policy
	decision := g.policyManager.Evaluate(agentID, toolName, actionName, requestParams)
	latencyMs := float64(time.Since(startTime).Microseconds()) / 1000.0

	// add telemetry attributes
	telemetry.AddSpanAttributes(span, map[string]interface{}{
		"agent.id":        agentID,
		"tool.name":       toolName,
		"tool.action":     actionName,
		"decision.allow":  decision.Allow,
		"policy.version":  decision.Version,
		"params.hash":     paramsHash,
		"latency.ms":      latencyMs,
		"parent.agent":    parentAgent,
	})

	telemetry.LogDecision(ctx, agentID, toolName, actionName, decision.Reason, paramsHash, parentAgent, decision.Allow, decision.Version, latencyMs)

	// check if policy allows this
	if !decision.Allow {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "PolicyViolation",
			Reason: decision.Reason,
		})
		return
	}

	// find the adapter for this tool
	adapterURL, ok := g.adapters[toolName]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "AdapterNotFound",
			Reason: fmt.Sprintf("No adapter configured for tool: %s", toolName),
		})
		return
	}

	// forward request to adapter
	targetURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(adapterURL, "/"), actionName)
	adapterResp, err := g.forward_to_adapter(ctx, targetURL, requestBody)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "AdapterError",
			Reason: err.Error(),
		})
		return
	}
	defer adapterResp.Body.Close()

	responseBody, err := io.ReadAll(adapterResp.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:  "AdapterError",
			Reason: "Failed to read adapter response",
		})
		return
	}

	// return adapter response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(adapterResp.StatusCode)
	w.Write(responseBody)
}

func (g *Gateway) forward_to_adapter(ctx context.Context, url string, body []byte) (*http.Response, error) {
	ctx, span := telemetry.StartSpan(ctx, "gateway.forward_to_adapter")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// 10 second timeout for adapter calls
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return httpClient.Do(req)
}

func (g *Gateway) Start(addr string) error {
	fmt.Printf("Gateway listening on %s\n", addr)
	return http.ListenAndServe(addr, g.router)
}

func (g *Gateway) Close() error {
	return g.watcher.Close()
}
