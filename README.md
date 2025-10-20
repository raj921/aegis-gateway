# Aegis Gateway



**Quick Verification**: `bash scripts/verify-completion.sh`

## Features

- **Policy-as-Code**: YAML-based policies with hot-reload capability
- **Reverse Proxy**: Secure routing between agents and tools
- **Policy Enforcement**: Fine-grained condition checks (amount limits, path prefixes, etc.)
- **Observability**: OpenTelemetry spans + structured JSON audit logs
- **Security**: Request validation, PII-safe logging (SHA-256 hashing), graceful error handling
- **Extensibility**: Clean architecture for adding new tools and policy conditions

## Architecture

```
┌──────────┐          ┌─────────────────┐          ┌──────────────┐
│  Agent   │─────────▶│  Aegis Gateway  │─────────▶│   Tools      │
│          │          │                 │          │              │
│          │          │  ┌───────────┐  │          │  ┌─────────┐ │
│          │          │  │  Policy   │  │          │  │ Payments│ │
│          │          │  │  Engine   │  │          │  └─────────┘ │
│          │          │  └───────────┘  │          │              │
│          │          │                 │          │  ┌─────────┐ │
│          │          │  ┌───────────┐  │          │  │  Files  │ │
│          │          │  │Telemetry  │  │          │  └─────────┘ │
│          │          │  └───────────┘  │          │              │
└──────────┘          └─────────────────┘          └──────────────┘
     │                        │
     │                        │
     └────────────────────────┴────────▶ OpenTelemetry / Audit Logs
```

## Project Structure

```
aegis-gateway/
├── cmd/aegis/              # Main application entry point
├── internal/
│   ├── gateway/           # HTTP gateway & routing
│   ├── policy/            # Policy engine & evaluation
│   └── adapters/          # Tool adapters (payments, files)
│       ├── payments/
│       └── files/
├── pkg/telemetry/         # OpenTelemetry & audit logging
├── policies/              # YAML policy files
├── scripts/               # Demo & test scripts
├── deploy/                # Docker & deployment configs
└── logs/                  # Audit log output
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose (optional, for containerized setup)

### Local Setup (One Command)

```bash
# Install dependencies
go mod tidy

# Run the gateway
go run cmd/aegis/main.go
```

The gateway will start on `http://localhost:8080` with:
- Gateway: `:8080`
- Payments adapter: `:8081`
- Files adapter: `:8082`

### Docker Setup

```bash
cd deploy
docker-compose up --build
```

This starts:
- Aegis Gateway
- OpenTelemetry Collector
- Jaeger UI (http://localhost:16686)

## Testing

### Run Unit Tests

```bash
make test
# or
go test -v ./...
```

Test coverage includes:
- Policy evaluation logic (allow/deny scenarios)
- Condition checking (amount limits, path prefixes, currencies)
- Gateway request handling and routing
- Adapter operations (payments, files)
- Hot reload functionality
- Telemetry and audit logging

### Run Demo

```bash
make demo
# or
chmod +x scripts/demo.sh
./scripts/demo.sh
```

This demonstrates:
1. **Blocked high-value payment** (exceeds `max_amount=5000`)
2. **Allowed payment** within limits
3. **Allowed HR file read** inside `/hr-docs/`
4. **Blocked HR file read** outside `/hr-docs/`

### Manual Test Cases

#### 1. Blocked High-Value Payment

```bash
curl -v -H "X-Agent-ID: finance-agent" \
  -X POST http://localhost:8080/tools/payments/create \
  -d '{"amount":50000,"currency":"USD","vendor_id":"V99"}'
```

Expected: `403 Forbidden`
```json
{
  "error": "PolicyViolation",
  "reason": "Amount 50000.00 exceeds max_amount=5000.00"
}
```

#### 2. Allowed Payment

```bash
curl -H "X-Agent-ID: finance-agent" \
  -X POST http://localhost:8080/tools/payments/create \
  -d '{"amount":1000,"currency":"USD","vendor_id":"V42","memo":"Office supplies"}'
```

Expected: `200 OK`
```json
{
  "payment_id": "uuid-here",
  "amount": 1000,
  "currency": "USD",
  "status": "created"
}
```

#### 3. Allowed HR File Read

```bash
curl -H "X-Agent-ID: hr-agent" \
  -X POST http://localhost:8080/tools/files/read \
  -d '{"path":"/hr-docs/employee-handbook.pdf"}'
```

Expected: `200 OK`
```json
{
  "path": "/hr-docs/employee-handbook.pdf",
  "content": "Employee handbook content..."
}
```

#### 4. Blocked HR File Read

```bash
curl -H "X-Agent-ID: hr-agent" \
  -X POST http://localhost:8080/tools/files/read \
  -d '{"path":"/legal/contract.docx"}'
```

Expected: `403 Forbidden`
```json
{
  "error": "PolicyViolation",
  "reason": "Path /legal/contract.docx does not match required prefix /hr-docs/"
}
```

## Policy Hot-Reload

Policies automatically reload when files change. Test it:

```bash
chmod +x scripts/test-hot-reload.sh
./scripts/test-hot-reload.sh
```

Or manually trigger reload:
```bash
curl -X POST http://localhost:8080/policies/reload
```

## Policy Configuration

### Example Policy

```yaml
version: 1
agents:
  - id: finance-agent
    allow:
      - tool: payments
        actions: [create, refund]
        conditions:
          max_amount: 5000
          currencies: [USD, EUR]
```

### Supported Conditions

- **`max_amount`**: Maximum payment amount (float)
- **`currencies`**: Allowed currency codes (array of strings)
- **`folder_prefix`**: Required path prefix (string)

Add new conditions in `internal/policy/policy.go:checkConditions()`

### Hot Reload

Policies automatically reload when files change. Test it:

```bash
# Terminal 1: Start the gateway
make run

# Terminal 2: Make a request that's currently allowed
curl -H "X-Agent-ID: finance-agent" \
  -X POST http://localhost:8080/tools/payments/create \
  -d '{"amount":3000,"currency":"USD","vendor_id":"V1"}'

# Terminal 3: Edit policies/example.yaml
# Change max_amount from 5000 to 2000

# Terminal 2: Make the same request again (now blocked)
curl -H "X-Agent-ID: finance-agent" \
  -X POST http://localhost:8080/tools/payments/create \
  -d '{"amount":3000,"currency":"USD","vendor_id":"V1"}'
# Returns: 403 PolicyViolation
```

Or use the automated test:
```bash
make hot-reload-test
```

## Telemetry & Audit Logs

### OpenTelemetry Spans

Every request creates spans with attributes:
- `agent.id`
- `tool.name`
- `tool.action`
- `decision.allow` (bool)
- `policy.version`
- `params.hash` (SHA-256)
- `latency.ms`
- `trace.id`

### JSON Audit Logs

Logs written to `stdout` and `logs/aegis.log`:

```json
{
  "timestamp": "2024-10-18T23:10:42Z",
  "trace_id": "abc123...",
  "agent_id": "finance-agent",
  "tool": "payments",
  "action": "create",
  "decision_allow": false,
  "reason": "Amount 50000.00 exceeds max_amount=5000.00",
  "policy_version": 1,
  "params_hash": "sha256...",
  "latency_ms": 2.34
}
```

**Security**: Request bodies are hashed (SHA-256), not logged in plain text.

## API Reference

### Gateway Endpoint

```
POST /tools/:tool/:action
```

**Headers:**
- `X-Agent-ID` (required): Agent identifier
- `X-Parent-Agent` (optional): Parent agent in call chain

**Request Body:** JSON (tool-specific)

**Responses:**
- `200 OK`: Tool response (passthrough)
- `403 Forbidden`: Policy violation
- `400 Bad Request`: Invalid request
- `502 Bad Gateway`: Tool adapter error

### Payments Tool

**Create Payment:**
```
POST /tools/payments/create
Body: {"amount": 1000, "currency": "USD", "vendor_id": "V42", "memo": "optional"}
```

**Refund Payment:**
```
POST /tools/payments/refund
Body: {"payment_id": "uuid", "reason": "optional"}
```

### Files Tool

**Read File:**
```
POST /tools/files/read
Body: {"path": "/hr-docs/file.pdf"}
```

**Write File:**
```
POST /tools/files/write
Body: {"path": "/tmp/output.txt", "content": "data"}
```

## Design Decisions

### 1. Stateless Gateway
- No session storage; all decisions based on policy + request
- Horizontal scaling ready

### 2. Policy Hot-Reload
- Uses `fsnotify` for file system watching
- Invalid policies fail gracefully without crashing
- Allows zero-downtime policy updates

### 3. Adapter Pattern
- Tools run as separate HTTP services (or in-process)
- Easy to add new tools without gateway changes
- Gateway forwards requests after policy approval

### 4. Security-First
- Request body hashing prevents PII leakage in logs
- Schema validation on all inputs
- Safe error messages (no internal details exposed)

### 5. Observability
- OpenTelemetry for distributed tracing
- Structured JSON logs for analysis
- Complete audit trail for compliance

## Extending the Gateway

### Adding a New Tool

1. Create adapter in `internal/adapters/newtool/`
2. Implement HTTP handlers
3. Register in `cmd/aegis/main.go`:
   ```go
   adapters["newtool"] = "http://localhost:8083"
   ```
4. Add policy rules in `policies/*.yaml`

### Adding New Policy Conditions

Edit `internal/policy/policy.go:checkConditions()`:

```go
case "your_condition":
    // your validation logic
    if violates {
        return "violation reason"
    }
```

## Testing

Run the gateway and execute:

```bash
# Functional tests
./scripts/demo.sh

# Hot-reload test
./scripts/test-hot-reload.sh

# Check audit logs
tail -f logs/aegis.log
```

## Security Considerations

- ✅ Request validation at gateway layer
- ✅ PII-safe logging (SHA-256 hashing)
- ✅ No secrets in repo
- ✅ Graceful error handling
- ✅ Timeout protection on adapter calls
- ✅ Least-privilege enforcement

## Production Readiness Checklist

- [x] Policy enforcement with hot-reload
- [x] OpenTelemetry integration
- [x] Structured audit logging
- [x] Clean architecture (extensible)
- [x] Docker deployment
- [x] Comprehensive documentation
- [x] Demo scripts
- [x] Security best practices

## Future Enhancements

- Call-chain ancestry tracking (`X-Parent-Agent` evaluation)
- Admin UI for policy management
- Approval gates for high-risk actions
- Rate limiting per agent
- Policy versioning & rollback
- Database-backed audit storage

## License

MIT

