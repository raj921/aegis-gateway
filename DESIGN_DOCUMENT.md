# Aegis Gateway - Design Document

## Executive Summary

Aegis Gateway is a production-grade reverse proxy that enforces least-privilege policies on agent-to-tool calls with comprehensive audit telemetry. It provides a secure, scalable, and observable layer between AI agents and external tools, ensuring that all interactions are policy-compliant and fully auditable.

## Architecture Overview

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        AI Agents                              │
│  (finance-agent, hr-agent, analytics-agent, etc.)            │
└───────────────────────┬──────────────────────────────────────┘
                        │
                        │ HTTP POST /tools/:tool/:action
                        │ Headers: X-Agent-ID, X-Parent-Agent
                        │
┌───────────────────────▼──────────────────────────────────────┐
│                    Aegis Gateway                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  1. Request Validation                                │   │
│  │     - Parse headers (X-Agent-ID, X-Parent-Agent)     │   │
│  │     - Validate JSON body                             │   │
│  │     - Hash params (SHA-256) for PII-safe logging     │   │
│  └──────────────────────────────────────────────────────┘   │
│                        │                                      │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │  2. Policy Evaluation                                │   │
│  │     - Load agent permissions from YAML               │   │
│  │     - Check tool/action authorization                │   │
│  │     - Evaluate conditions (amount, path, etc.)       │   │
│  └──────────────────────────────────────────────────────┘   │
│                        │                                      │
│                   ┌────┴────┐                                │
│                   │  Allow? │                                │
│                   └────┬────┘                                │
│               Yes ◄────┘────► No                             │
│                   │              │                            │
│  ┌────────────────▼───┐    ┌────▼─────────────────────┐    │
│  │  3. Forward to     │    │  4. Return 403           │    │
│  │     Adapter        │    │     PolicyViolation      │    │
│  │  - HTTP POST       │    │  - Log decision          │    │
│  │  - 10s timeout     │    │  - Emit OTel span        │    │
│  └────────────────────┘    └──────────────────────────┘    │
│           │                                                   │
│  ┌────────▼────────────────────────────────────────────┐   │
│  │  5. Telemetry & Audit Logging                       │   │
│  │     - OpenTelemetry spans with attributes           │   │
│  │     - Structured JSON logs to stdout & file         │   │
│  │     - Trace ID, decision, latency, params hash      │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────────────┬──────────────────────────────────────┘
                        │
        ┌───────────────┴───────────────┐
        │                               │
┌───────▼────────┐            ┌─────────▼────────┐
│  Payments Tool │            │   Files Tool     │
│  - /create     │            │   - /read        │
│  - /refund     │            │   - /write       │
└────────────────┘            └──────────────────┘
```

### Component Breakdown

#### 1. Gateway Layer (`internal/gateway`)
- **Responsibilities:**
  - HTTP request routing and handling
  - Header validation (X-Agent-ID, X-Parent-Agent)
  - Request/response proxying
  - Integration with policy engine and telemetry
  
- **Key Features:**
  - Gorilla Mux for routing
  - File system watcher for hot-reload
  - Graceful error handling
  - 10-second timeout for adapter calls

#### 2. Policy Engine (`internal/policy`)
- **Responsibilities:**
  - Load and parse YAML policy files
  - Evaluate agent permissions
  - Check conditions (max_amount, currencies, folder_prefix)
  - Provide clear denial reasons
  
- **Key Features:**
  - Thread-safe policy management (sync.RWMutex)
  - Hot-reload capability
  - Invalid policy file handling (doesn't crash)
  - Extensible condition system

#### 3. Tool Adapters (`internal/adapters`)
- **Payments Adapter:**
  - Create payments with amount/currency validation
  - Refund payments with reason tracking
  - In-memory storage for demo purposes
  
- **Files Adapter:**
  - Read files with path validation
  - Write files with content validation
  - In-memory file system for demo purposes

#### 4. Telemetry Layer (`pkg/telemetry`)
- **OpenTelemetry Integration:**
  - Span creation with context propagation
  - Rich attributes (agent.id, tool.name, decision.allow, etc.)
  - Stdout exporter for local development
  - Ready for OTLP collector integration
  
- **Audit Logging:**
  - Structured JSON logs
  - SHA-256 hashing of request params (PII-safe)
  - Trace ID correlation
  - Dual output (stdout + file)

## Design Decisions & Trade-offs

### 1. Policy-First Architecture
**Decision:** Evaluate policy before checking adapter availability.

**Rationale:**
- Security-first approach
- Fail fast on unauthorized requests
- Consistent error handling
- Reduces attack surface

**Trade-off:** Unknown tools return 403 (PolicyViolation) instead of 404 (NotFound), but this is acceptable as it doesn't leak information about available tools.

### 2. In-Memory Adapters
**Decision:** Use in-memory storage for mock tools.

**Rationale:**
- Simplifies demo and testing
- No external dependencies
- Fast and deterministic
- Easy to reset state

**Trade-off:** Data doesn't persist across restarts, but this is intentional for a demo/test environment.

### 3. Hot Reload with File Watcher
**Decision:** Use fsnotify for automatic policy reloading.

**Rationale:**
- Zero-downtime policy updates
- Developer-friendly workflow
- Production-ready pattern
- Graceful error handling

**Trade-off:** Adds complexity and a goroutine, but the benefits far outweigh the cost.

### 4. SHA-256 Param Hashing
**Decision:** Hash request parameters before logging.

**Rationale:**
- PII protection
- Compliance-friendly
- Deterministic (same params = same hash)
- Enables correlation without exposing sensitive data

**Trade-off:** Can't debug with actual values, but this is the right security posture.

### 5. Structured JSON Logging
**Decision:** Use JSON format for all audit logs.

**Rationale:**
- Machine-readable
- Easy to parse and analyze
- Standard format for log aggregation
- Rich metadata support

**Trade-off:** Less human-readable than plain text, but tools like `jq` make it easy to work with.

## Security Considerations

### 1. Input Validation
- All request bodies validated as JSON
- Required fields checked before processing
- Type validation for numeric and string fields
- Path traversal prevention in file operations

### 2. PII Protection
- Request parameters hashed with SHA-256
- No raw sensitive data in logs
- Trace IDs for correlation without exposure
- Safe error messages (no stack traces to clients)

### 3. Least Privilege
- Agents only access explicitly allowed tools/actions
- Fine-grained condition checks
- Default-deny policy model
- Clear audit trail of all decisions

### 4. Timeout Protection
- 10-second timeout on adapter calls
- Prevents resource exhaustion
- Graceful error handling
- Client receives clear timeout errors

## Scalability & Performance

### 1. Stateless Design
- Gateway holds no session state
- Policy loaded from files (can be externalized)
- Horizontal scaling ready
- Load balancer friendly

### 2. Efficient Locking
- RWMutex for policy reads (high concurrency)
- Minimal lock duration
- Thread-safe adapter operations
- No global locks

### 3. Resource Management
- HTTP client pooling
- Bounded timeouts
- Graceful shutdown support
- Memory-efficient data structures

### 4. Extension Points
- Pluggable adapters (interface-based)
- Custom condition evaluators
- Alternative policy stores (DB, etcd, etc.)
- Multiple telemetry exporters

## Observability

### 1. OpenTelemetry Spans
```
gateway.handleToolRequest
├── Attributes:
│   ├── agent.id: "finance-agent"
│   ├── tool.name: "payments"
│   ├── tool.action: "create"
│   ├── decision.allow: true
│   ├── policy.version: 1
│   ├── params.hash: "abc123..."
│   ├── latency.ms: 12.5
│   └── parent.agent: "supervisor-agent"
└── gateway.forward_to_adapter
    └── HTTP POST to adapter
```

### 2. Audit Log Format
```json
{
  "timestamp": "2025-10-20T08:00:00Z",
  "trace_id": "abc123...",
  "agent_id": "finance-agent",
  "tool": "payments",
  "action": "create",
  "decision_allow": true,
  "reason": "Policy allows this action",
  "policy_version": 1,
  "params_hash": "def456...",
  "latency_ms": 12.5,
  "parent_agent": "supervisor-agent"
}
```

### 3. Metrics (Future)
- Request rate per agent
- Policy evaluation latency
- Adapter response times
- Denial rate by reason
- Hot reload success/failure

## Testing Strategy

### 1. Unit Tests
- Policy evaluation logic
- Condition checking
- Adapter request/response handling
- Telemetry attribute setting

### 2. Integration Tests
- End-to-end request flow
- Policy hot reload
- Error handling paths
- Header propagation

### 3. Demo Scripts
- Allowed and blocked scenarios
- Multiple agents and tools
- Condition boundary testing
- Audit log verification

## Deployment

### 1. Local Development
```bash
make deps    # Install dependencies
make run     # Start gateway + adapters
make demo    # Run demo scenarios
make test    # Run all tests
```

### 2. Docker Compose
```bash
make docker-up      # Start all services
make docker-logs    # View logs
make docker-down    # Stop services
```

Services:
- Aegis Gateway (port 8080)
- OTel Collector (port 4318)
- Jaeger UI (port 16686)

### 3. Production Considerations
- Use external policy store (DB, S3, ConfigMap)
- Configure OTLP exporter to collector
- Set up log aggregation (ELK, Loki)
- Enable TLS for all connections
- Implement rate limiting
- Add health checks and readiness probes
- Use secrets management for sensitive config

## Future Enhancements

### Near-Term (Next Sprint)
1. **Call Chain Tracking**
   - Validate X-Parent-Agent against policy
   - Prevent circular agent calls
   - Depth limits for call chains

2. **Approval Gates**
   - Soft-deny for risky actions
   - Manual approval workflow
   - Time-bound approvals

3. **Rate Limiting**
   - Per-agent request limits
   - Token bucket algorithm
   - Configurable windows

### Medium-Term (Next Quarter)
1. **Admin UI**
   - View agents and policies
   - Last 50 decisions dashboard
   - Real-time policy validation
   - Approval queue management

2. **Advanced Conditions**
   - Time-based restrictions
   - Geo-fencing
   - Risk scoring
   - Dynamic limits based on history

3. **Multi-Tenancy**
   - Namespace isolation
   - Per-tenant policies
   - Resource quotas

### Long-Term (Future)
1. **ML-Based Anomaly Detection**
   - Unusual access patterns
   - Behavioral analysis
   - Automated threat response

2. **Policy Testing Framework**
   - Unit tests for policies
   - Simulation mode
   - Impact analysis

3. **Federation**
   - Multi-gateway deployment
   - Cross-gateway policy sync
   - Distributed tracing

## Conclusion

Aegis Gateway provides a robust, secure, and observable foundation for managing agent-to-tool interactions. Its clean architecture, comprehensive testing, and production-ready features make it suitable for both development and production environments. The design prioritizes security, scalability, and developer experience while maintaining flexibility for future enhancements.
