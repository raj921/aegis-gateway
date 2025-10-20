# Aegis Gateway - Design Document

## Overview

Aegis Gateway is a reverse-proxy service that enforces least-privilege policies on agent-to-tool communications. It provides policy-as-code governance, comprehensive audit telemetry, and a clean architecture for extensibility.

## Architecture Decisions

### 1. Layered Architecture

```
Presentation Layer (HTTP Gateway)
        ↓
Business Logic (Policy Engine)
        ↓
Adapter Layer (Tool Integrations)
        ↓
Infrastructure (Telemetry, Logging)
```

**Rationale:**
- Clear separation of concerns
- Each layer is independently testable
- Easy to swap implementations (e.g., different tool adapters)
- Follows SOLID principles

### 2. Policy Engine Design

**Key Components:**
- **Policy Manager**: Loads, validates, and manages policy files
- **Hot-Reload**: Uses `fsnotify` for file system watching
- **Evaluation Logic**: Stateless decision-making

**Why YAML over JSON:**
- More human-readable
- Better for configuration files
- Supports comments (future enhancement)
- Industry standard for K8s, Terraform, etc.

**Policy Structure:**
```yaml
version: 1          # Versioning for migration support
agents:
  - id: string      # Agent identifier
    allow:
      - tool: string
        actions: []
        conditions: {}
```

**Conditions Framework:**
Extensible condition checking in `checkConditions()`:
- Type-specific validations (amount, currency, path)
- Easy to add new condition types
- Clear error messages on violation

### 3. Adapter Pattern

**Why Separate Adapters:**
- Tools can be deployed independently
- Different scaling needs per tool
- Technology diversity (Go, Python, Node.js)
- Fault isolation

**Current Implementation:**
- HTTP-based adapters (payments, files)
- In-process for simplicity
- Can be external services via config

**Future:**
- gRPC adapters
- Message queue integrations
- Plugin system

### 4. Telemetry Strategy

**OpenTelemetry Choice:**
- Vendor-neutral standard
- Rich ecosystem (Jaeger, Prometheus, Grafana)
- Distributed tracing ready
- Spans capture full request lifecycle

**Span Attributes:**
```
agent.id          → Who made the request
tool.name         → Target tool
tool.action       → Specific action
decision.allow    → Policy result
policy.version    → Which policy version
params.hash       → SHA-256 (PII-safe)
latency.ms        → Performance tracking
trace.id          → Request correlation
```

**Audit Logging:**
- Structured JSON for machine parsing
- Dual output: stdout + file
- Retention-ready format
- Compliant with audit standards

### 5. Security Design

**Threat Model:**
1. **Malicious Agent**: Attempts unauthorized actions
   - Mitigation: Policy enforcement, deny-by-default
2. **Policy Bypass**: Tries to circumvent checks
   - Mitigation: Centralized enforcement point
3. **PII Leakage**: Sensitive data in logs
   - Mitigation: SHA-256 hashing of request bodies
4. **Policy Tampering**: Modifies policy files
   - Mitigation: File system permissions, validation
5. **Tool Impersonation**: Fake tool responses
   - Mitigation: Authenticated adapter URLs (future)

**Security Features:**
- Request validation at gateway
- Schema enforcement
- Safe error messages (no stack traces)
- Timeout protection (10s on adapter calls)
- No secrets in codebase

### 6. Scalability Considerations

**Stateless Gateway:**
- No shared state between requests
- Horizontal scaling via load balancer
- Policy loaded in-memory (fast reads)

**Performance Optimizations:**
- Policy map lookup: O(1) for agent ID
- Minimal allocations in hot path
- Read-write mutex for policy updates

**Bottlenecks & Solutions:**
- Hot-reload watches single directory
  - Solution: Watch recursively if needed
- Policy evaluation is synchronous
  - Solution: Pre-compiled policies (future)

**Load Testing Targets:**
- 1000 req/s per gateway instance
- <10ms policy evaluation
- <100ms end-to-end latency

### 7. Trade-offs

| Decision | Pros | Cons | Rationale |
|----------|------|------|-----------|
| In-process adapters | Simple deployment | Not independently scalable | Good for demo, easy to separate |
| YAML policies | Human-readable | No schema autocomplete | Can add JSON Schema |
| Hot-reload vs versioned releases | Zero downtime | Risk of bad policies | Validation catches most issues |
| HTTP adapters vs gRPC | Universal, easy debug | Slower than gRPC | HTTP sufficient for current scale |
| Stdout tracing | Simple setup | Not production-grade | Easy to swap for OTLP exporter |

## Extension Points

### 1. Adding New Policy Conditions

**Location:** `internal/policy/policy.go`

```go
case "new_condition":
    expected := condValue.(string)
    actual := params["field"].(string)
    if actual != expected {
        return "Violation message"
    }
```

### 2. Adding New Tools

**Steps:**
1. Create adapter in `internal/adapters/newtool/`
2. Implement handler functions
3. Register in `cmd/aegis/main.go`
4. Add policies

**Example:**
```go
// internal/adapters/database/database.go
func (a *Adapter) HandleQuery(w http.ResponseWriter, r *http.Request) {
    // query logic
}
```

### 3. Call-Chain Tracking

**Future Enhancement:**

```yaml
conditions:
  max_chain_depth: 3
  allowed_parents: [supervisor-agent]
```

Parse `X-Parent-Agent` header and validate ancestry.

### 4. Approval Gates

**Design:**

```yaml
actions: [large_payment]
approval_required: true
approvers: [manager-agent]
```

Flow:
1. Request → Soft-deny with approval ID
2. Manager calls `/approve/{id}`
3. Original request re-evaluated

## Near-Term Roadmap

### Phase 1: Robustness (1 week)
- [ ] Unit tests for policy engine
- [ ] Integration tests for gateway
- [ ] Error handling improvements
- [ ] Metrics (request rate, policy hits)

### Phase 2: Production Features (2 weeks)
- [ ] Database-backed audit logs
- [ ] Policy versioning & rollback
- [ ] Rate limiting per agent
- [ ] Circuit breakers for adapters

### Phase 3: Advanced Governance (4 weeks)
- [ ] Admin UI (React)
- [ ] Policy simulation mode
- [ ] Approval workflows
- [ ] Call-chain ancestry rules

## Testing Strategy

### Unit Tests
- Policy evaluation logic
- Condition checking
- Parameter hashing

### Integration Tests
- End-to-end request flow
- Hot-reload behavior
- Error scenarios

### Performance Tests
- Load testing (wrk, k6)
- Policy evaluation benchmarks
- Memory profiling

## Deployment Considerations

### Development
- Single binary
- File-based policies
- Stdout logging

### Staging
- Docker Compose
- Centralized logging
- Health checks

### Production
- Kubernetes deployment
- ConfigMaps for policies
- Prometheus metrics
- Jaeger tracing
- Log aggregation (ELK)

### High Availability
- Multiple gateway replicas
- Load balancer (nginx, HAProxy)
- Policy sync via ConfigMap
- Adapter health checks

## Monitoring & Alerting

### Key Metrics
- `aegis_policy_decisions_total{allow="true|false"}`
- `aegis_request_latency_ms`
- `aegis_adapter_errors_total{tool="..."}`
- `aegis_policy_reload_total{status="success|failure"}`

### Alerts
- High policy violation rate (>10%)
- Slow adapter responses (>500ms)
- Policy reload failures
- Request error rate (>5%)

## Compliance & Audit

### Audit Requirements
- Who (agent.id)
- What (tool.action)
- When (timestamp)
- Why (decision.allow, reason)
- Result (response status)

### Retention Policy
- Logs: 90 days
- Traces: 30 days
- Metrics: 1 year

### GDPR Considerations
- No PII in logs (SHA-256 hashing)
- Right to deletion (purge by agent ID)
- Data minimization (only essential attributes)

## Conclusion

Aegis Gateway provides a solid foundation for policy-driven governance of agent-tool interactions. The architecture prioritizes security, observability, and extensibility while maintaining simplicity in the core design.

Key strengths:
- ✅ Production-ready patterns
- ✅ Clean separation of concerns
- ✅ Security by default
- ✅ Observable at every layer
- ✅ Easy to extend and maintain

The modular design ensures that as requirements evolve, the gateway can adapt without major refactoring.
