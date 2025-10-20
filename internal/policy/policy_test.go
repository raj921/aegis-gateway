package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyValidation(t *testing.T) {
	tests := []struct {
		name      string
		policy    Policy
		wantError bool
	}{
		{
			name: "valid policy",
			policy: Policy{
				Version: 1,
				Agents: []Agent{
					{
						ID: "test-agent",
						Allow: []Permission{
							{
								Tool:    "payments",
								Actions: []string{"create"},
							},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid version",
			policy: Policy{
				Version: 0,
				Agents: []Agent{
					{ID: "test-agent"},
				},
			},
			wantError: true,
		},
		{
			name: "no agents",
			policy: Policy{
				Version: 1,
				Agents:  []Agent{},
			},
			wantError: true,
		},
		{
			name: "empty agent ID",
			policy: Policy{
				Version: 1,
				Agents: []Agent{
					{ID: ""},
				},
			},
			wantError: true,
		},
	}

	m := &Manager{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.check_policy_valid(&tt.policy)
			if (err != nil) != tt.wantError {
				t.Errorf("check_policy_valid() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestPolicyEvaluation(t *testing.T) {
	// create temp directory for test policies
	tmpDir := t.TempDir()
	
	// create a test policy file
	policyContent := `version: 1
agents:
  - id: finance-agent
    allow:
      - tool: payments
        actions: [create, refund]
        conditions:
          max_amount: 5000
          currencies: [USD, EUR]
  - id: hr-agent
    allow:
      - tool: files
        actions: [read]
        conditions:
          folder_prefix: "/hr-docs/"
`
	
	policyPath := filepath.Join(tmpDir, "test-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write test policy: %v", err)
	}

	m, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name      string
		agentID   string
		tool      string
		action    string
		params    map[string]interface{}
		wantAllow bool
		wantReason string
	}{
		{
			name:    "allowed payment within limits",
			agentID: "finance-agent",
			tool:    "payments",
			action:  "create",
			params: map[string]interface{}{
				"amount":   1000.0,
				"currency": "USD",
			},
			wantAllow: true,
		},
		{
			name:    "blocked payment exceeds limit",
			agentID: "finance-agent",
			tool:    "payments",
			action:  "create",
			params: map[string]interface{}{
				"amount":   10000.0,
				"currency": "USD",
			},
			wantAllow: false,
		},
		{
			name:    "blocked invalid currency",
			agentID: "finance-agent",
			tool:    "payments",
			action:  "create",
			params: map[string]interface{}{
				"amount":   1000.0,
				"currency": "GBP",
			},
			wantAllow: false,
		},
		{
			name:    "allowed file read in hr-docs",
			agentID: "hr-agent",
			tool:    "files",
			action:  "read",
			params: map[string]interface{}{
				"path": "/hr-docs/handbook.pdf",
			},
			wantAllow: true,
		},
		{
			name:    "blocked file read outside hr-docs",
			agentID: "hr-agent",
			tool:    "files",
			action:  "read",
			params: map[string]interface{}{
				"path": "/legal/contract.pdf",
			},
			wantAllow: false,
		},
		{
			name:    "blocked unknown agent",
			agentID: "unknown-agent",
			tool:    "payments",
			action:  "create",
			params: map[string]interface{}{
				"amount":   1000.0,
				"currency": "USD",
			},
			wantAllow: false,
		},
		{
			name:    "blocked unknown action",
			agentID: "finance-agent",
			tool:    "payments",
			action:  "delete",
			params: map[string]interface{}{
				"amount":   1000.0,
				"currency": "USD",
			},
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := m.Evaluate(tt.agentID, tt.tool, tt.action, tt.params)
			if decision.Allow != tt.wantAllow {
				t.Errorf("Evaluate() Allow = %v, want %v. Reason: %s", decision.Allow, tt.wantAllow, decision.Reason)
			}
		})
	}
}

func TestHashParams(t *testing.T) {
	params1 := map[string]interface{}{
		"amount":   1000.0,
		"currency": "USD",
	}
	
	params2 := map[string]interface{}{
		"amount":   1000.0,
		"currency": "USD",
	}
	
	params3 := map[string]interface{}{
		"amount":   2000.0,
		"currency": "USD",
	}

	hash1 := HashParams(params1)
	hash2 := HashParams(params2)
	hash3 := HashParams(params3)

	// same params should produce same hash
	if hash1 != hash2 {
		t.Errorf("Same params produced different hashes: %s != %s", hash1, hash2)
	}

	// different params should produce different hashes
	if hash1 == hash3 {
		t.Errorf("Different params produced same hash: %s", hash1)
	}

	// hash should be 64 characters (SHA-256 hex)
	if len(hash1) != 64 {
		t.Errorf("Hash length = %d, want 64", len(hash1))
	}
}

func TestPolicyReload(t *testing.T) {
	tmpDir := t.TempDir()
	
	// create initial policy
	initialPolicy := `version: 1
agents:
  - id: test-agent
    allow:
      - tool: payments
        actions: [create]
`
	
	policyPath := filepath.Join(tmpDir, "test-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(initialPolicy), 0644); err != nil {
		t.Fatalf("Failed to write initial policy: %v", err)
	}

	m, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// test initial policy
	decision := m.Evaluate("test-agent", "payments", "create", map[string]interface{}{})
	if !decision.Allow {
		t.Errorf("Initial policy should allow action")
	}

	// update policy to remove permission
	updatedPolicy := `version: 2
agents:
  - id: test-agent
    allow:
      - tool: files
        actions: [read]
`
	
	if err := os.WriteFile(policyPath, []byte(updatedPolicy), 0644); err != nil {
		t.Fatalf("Failed to write updated policy: %v", err)
	}

	// reload policies
	if err := m.Reload(); err != nil {
		t.Fatalf("Failed to reload policies: %v", err)
	}

	// test updated policy
	decision = m.Evaluate("test-agent", "payments", "create", map[string]interface{}{})
	if decision.Allow {
		t.Errorf("Updated policy should deny action")
	}
}

func TestConditionChecks(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name       string
		conditions map[string]interface{}
		params     map[string]interface{}
		wantReason string
	}{
		{
			name: "max_amount pass",
			conditions: map[string]interface{}{
				"max_amount": 5000,
			},
			params: map[string]interface{}{
				"amount": 1000.0,
			},
			wantReason: "",
		},
		{
			name: "max_amount fail",
			conditions: map[string]interface{}{
				"max_amount": 5000,
			},
			params: map[string]interface{}{
				"amount": 10000.0,
			},
			wantReason: "Amount 10000.00 exceeds max_amount=5000.00",
		},
		{
			name: "currencies pass",
			conditions: map[string]interface{}{
				"currencies": []interface{}{"USD", "EUR"},
			},
			params: map[string]interface{}{
				"currency": "USD",
			},
			wantReason: "",
		},
		{
			name: "currencies fail",
			conditions: map[string]interface{}{
				"currencies": []interface{}{"USD", "EUR"},
			},
			params: map[string]interface{}{
				"currency": "GBP",
			},
			wantReason: "Currency GBP not in allowed list",
		},
		{
			name: "folder_prefix pass",
			conditions: map[string]interface{}{
				"folder_prefix": "/hr-docs/",
			},
			params: map[string]interface{}{
				"path": "/hr-docs/handbook.pdf",
			},
			wantReason: "",
		},
		{
			name: "folder_prefix fail",
			conditions: map[string]interface{}{
				"folder_prefix": "/hr-docs/",
			},
			params: map[string]interface{}{
				"path": "/legal/contract.pdf",
			},
			wantReason: "Path /legal/contract.pdf does not match required prefix /hr-docs/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := m.check_conditions(tt.conditions, tt.params)
			if reason != tt.wantReason {
				t.Errorf("check_conditions() = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}
