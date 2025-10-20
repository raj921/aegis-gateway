package payments

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCreate_Success(t *testing.T) {
	adapter := NewAdapter()

	body := CreateRequest{
		Amount:   1000.0,
		Currency: "USD",
		VendorID: "V123",
		Memo:     "Test payment",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/create", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleCreate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp CreateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Amount != 1000.0 {
		t.Errorf("Expected amount 1000.0, got %f", resp.Amount)
	}
	if resp.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", resp.Currency)
	}
	if resp.Status != "created" {
		t.Errorf("Expected status created, got %s", resp.Status)
	}
	if resp.PaymentID == "" {
		t.Error("Expected non-empty payment ID")
	}
}

func TestHandleCreate_InvalidJSON(t *testing.T) {
	adapter := NewAdapter()

	req := httptest.NewRequest("POST", "/create", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	adapter.HandleCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreate_MissingFields(t *testing.T) {
	adapter := NewAdapter()

	tests := []struct {
		name string
		body CreateRequest
	}{
		{
			name: "missing amount",
			body: CreateRequest{
				Currency: "USD",
				VendorID: "V123",
			},
		},
		{
			name: "missing currency",
			body: CreateRequest{
				Amount:   1000.0,
				VendorID: "V123",
			},
		},
		{
			name: "missing vendor_id",
			body: CreateRequest{
				Amount:   1000.0,
				Currency: "USD",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/create", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			adapter.HandleCreate(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleRefund_Success(t *testing.T) {
	adapter := NewAdapter()

	// first create a payment
	createBody := CreateRequest{
		Amount:   1000.0,
		Currency: "USD",
		VendorID: "V123",
	}
	createBodyBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest("POST", "/create", bytes.NewReader(createBodyBytes))
	createW := httptest.NewRecorder()
	adapter.HandleCreate(createW, createReq)

	var createResp CreateResponse
	json.NewDecoder(createW.Body).Decode(&createResp)

	// now refund it
	refundBody := RefundRequest{
		PaymentID: createResp.PaymentID,
		Reason:    "Test refund",
	}
	refundBodyBytes, _ := json.Marshal(refundBody)

	req := httptest.NewRequest("POST", "/refund", bytes.NewReader(refundBodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleRefund(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp RefundResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.PaymentID != createResp.PaymentID {
		t.Errorf("Expected payment_id %s, got %s", createResp.PaymentID, resp.PaymentID)
	}
	if resp.Status != "refunded" {
		t.Errorf("Expected status refunded, got %s", resp.Status)
	}
	if resp.RefundID == "" {
		t.Error("Expected non-empty refund ID")
	}
}

func TestHandleRefund_InvalidJSON(t *testing.T) {
	adapter := NewAdapter()

	req := httptest.NewRequest("POST", "/refund", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	adapter.HandleRefund(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleRefund_MissingPaymentID(t *testing.T) {
	adapter := NewAdapter()

	body := RefundRequest{
		Reason: "Test refund",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/refund", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	adapter.HandleRefund(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
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
