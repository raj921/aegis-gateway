package payments

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CreateRequest struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	VendorID  string  `json:"vendor_id"`
	Memo      string  `json:"memo,omitempty"`
}

type CreateResponse struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Status    string  `json:"status"`
}

type RefundRequest struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason,omitempty"`
}

type RefundResponse struct {
	RefundID  string `json:"refund_id"`
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
}

type Adapter struct {
	mu       sync.RWMutex
	payments map[string]CreateResponse
	refunds  map[string]RefundResponse
}

func NewAdapter() *Adapter {
	return &Adapter{
		payments: make(map[string]CreateResponse),
		refunds:  make(map[string]RefundResponse),
	}
}

func (a *Adapter) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"InvalidRequest","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, `{"error":"InvalidRequest","message":"Amount must be positive"}`, http.StatusBadRequest)
		return
	}
	if req.Currency == "" {
		http.Error(w, `{"error":"InvalidRequest","message":"Currency is required"}`, http.StatusBadRequest)
		return
	}
	if req.VendorID == "" {
		http.Error(w, `{"error":"InvalidRequest","message":"VendorID is required"}`, http.StatusBadRequest)
		return
	}

	resp := CreateResponse{
		PaymentID: uuid.New().String(),
		Amount:    req.Amount,
		Currency:  req.Currency,
		Status:    "created",
	}

	a.mu.Lock()
	a.payments[resp.PaymentID] = resp
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *Adapter) HandleRefund(w http.ResponseWriter, r *http.Request) {
	var req RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"InvalidRequest","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if req.PaymentID == "" {
		http.Error(w, `{"error":"InvalidRequest","message":"PaymentID is required"}`, http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	_, exists := a.payments[req.PaymentID]
	a.mu.RUnlock()

	if !exists {
		http.Error(w, `{"error":"NotFound","message":"Payment not found"}`, http.StatusNotFound)
		return
	}

	resp := RefundResponse{
		RefundID:  uuid.New().String(),
		PaymentID: req.PaymentID,
		Status:    "refunded",
	}

	a.mu.Lock()
	a.refunds[resp.RefundID] = resp
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *Adapter) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (a *Adapter) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/create", a.HandleCreate)
	mux.HandleFunc("/refund", a.HandleRefund)
	mux.HandleFunc("/health", a.HandleHealth)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("Payments adapter listening on %s\n", addr)
	return server.ListenAndServe()
}
