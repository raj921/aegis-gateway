package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Policy stuff - main structure for YAML files
type Policy struct {
	Version int      `yaml:"version"`
	Agents  []Agent  `yaml:"agents"`
}

type Agent struct {
	ID    string       `yaml:"id"`
	Allow []Permission `yaml:"allow"`
}

type Permission struct {
	Tool       string                 `yaml:"tool"`
	Actions    []string               `yaml:"actions"`
	Conditions map[string]interface{} `yaml:"conditions"`
}

// result of policy check
type Decision struct {
	Allow   bool
	Reason  string
	Version int
}

type Manager struct {
	mu       sync.RWMutex
	policies map[string]Policy
	dir      string
}

func NewManager(dir string) (*Manager, error) {
	m := &Manager{
		policies: make(map[string]Policy),
		dir:      dir,
	}
	err := m.load_policies()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load_policies() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return fmt.Errorf("failed to read policies directory: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// clear old policies and load fresh ones
	newPolicies := make(map[string]Policy)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		policyPath := filepath.Join(m.dir, entry.Name())
		fileData, err := os.ReadFile(policyPath)
		if err != nil {
			fmt.Printf("ERROR: failed to read policy file %s: %v\n", policyPath, err)
			continue
		}

		var pol Policy
		if err := yaml.Unmarshal(fileData, &pol); err != nil {
			fmt.Printf("ERROR: failed to parse policy file %s: %v\n", policyPath, err)
			continue
		}

		// validate before adding
		if err := m.check_policy_valid(&pol); err != nil {
			fmt.Printf("ERROR: invalid policy file %s: %v\n", policyPath, err)
			continue
		}

		newPolicies[entry.Name()] = pol
	}

	m.policies = newPolicies
	return nil
}

func (m *Manager) check_policy_valid(p *Policy) error {
	// basic validation
	if p.Version < 1 {
		return fmt.Errorf("policy version must be >= 1")
	}
	if len(p.Agents) == 0 {
		return fmt.Errorf("policy must have at least one agent")
	}
	for _, agent := range p.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent ID cannot be empty")
		}
	}
	return nil
}

// reload all policies from disk
func (m *Manager) Reload() error {
	return m.load_policies()
}

// check if agent can do this action
func (m *Manager) Evaluate(agentID, tool, action string, params map[string]interface{}) Decision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// loop through all loaded policies
	for _, policy := range m.policies {
		for _, agent := range policy.Agents {
			if agent.ID != agentID {
				continue
			}

			// found the agent, check permissions
			for _, perm := range agent.Allow {
				if perm.Tool != tool {
					continue
				}

				// check if action is in allowed list
				actionAllowed := false
				for _, a := range perm.Actions {
					if a == action {
						actionAllowed = true
						break
					}
				}
				if !actionAllowed {
					continue
				}

				// check conditions (amount, currency, path, etc)
				if reason := m.check_conditions(perm.Conditions, params); reason != "" {
					return Decision{
						Allow:   false,
						Reason:  reason,
						Version: policy.Version,
					}
				}

				// all checks passed!
				return Decision{
					Allow:   true,
					Reason:  "Policy allows this action",
					Version: policy.Version,
				}
			}
		}
	}

	// no matching policy found
	return Decision{
		Allow:  false,
		Reason: fmt.Sprintf("No policy found for agent=%s, tool=%s, action=%s", agentID, tool, action),
	}
}

func (m *Manager) check_conditions(conditions map[string]interface{}, params map[string]interface{}) string {
	// iterate through each condition and validate
	for condName, condVal := range conditions {
		switch condName {
		case "max_amount":
			var maxAmt float64
			// handle different number types from yaml
			switch v := condVal.(type) {
			case float64:
				maxAmt = v
			case int:
				maxAmt = float64(v)
			default:
				fmt.Printf("WARNING: invalid max_amount type in policy: %T\n", condVal)
				continue
			}
			
			amt, ok := params["amount"].(float64)
			if !ok {
				return "Invalid amount parameter"
			}
			if amt > maxAmt {
				return fmt.Sprintf("Amount %.2f exceeds max_amount=%.2f", amt, maxAmt)
			}

		case "currencies":
			allowedCurrs, ok := condVal.([]interface{})
			if !ok {
				fmt.Printf("WARNING: invalid currencies type in policy: %T\n", condVal)
				continue
			}
			curr, ok := params["currency"].(string)
			if !ok {
				return "Invalid currency parameter"
			}
			
			// check if currency is in the allowed list
			currencyFound := false
			for _, c := range allowedCurrs {
				cStr, ok := c.(string)
				if !ok {
					continue
				}
				if cStr == curr {
					currencyFound = true
					break
				}
			}
			if !currencyFound {
				return fmt.Sprintf("Currency %s not in allowed list", curr)
			}

		case "folder_prefix":
			pfx, ok := condVal.(string)
			if !ok {
				fmt.Printf("WARNING: invalid folder_prefix type in policy: %T\n", condVal)
				continue
			}
			pth, ok := params["path"].(string)
			if !ok {
				return "Invalid path parameter"
			}
			// check if path starts with required prefix
			if !strings.HasPrefix(pth, pfx) {
				return fmt.Sprintf("Path %s does not match required prefix %s", pth, pfx)
			}
		}
	}
	return ""
}

// hash the request params for logging (PII safe)
func HashParams(params map[string]interface{}) string {
	data, _ := json.Marshal(params)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
