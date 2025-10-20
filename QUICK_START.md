# Quick Start Guide

## 1. Install & Build

```bash
# Install dependencies
make deps

# Build the binary
make build
```

## 2. Run the Gateway

```bash
# Option 1: Run directly
make run

# Option 2: Use the binary
./bin/aegis-gateway

# Option 3: Docker
make docker-up
```

The gateway starts on:
- Gateway: `http://localhost:8080`
- Payments adapter: `http://localhost:8081`
- Files adapter: `http://localhost:8082`

## 3. Run Demo

```bash
# Run all 4 test cases
make demo
```

Expected output:
- ✓ Test 1: Blocked high-value payment
- ✓ Test 2: Allowed payment within limits
- ✓ Test 3: Allowed HR file read
- ✓ Test 4: Blocked HR file read

## 4. Test Hot-Reload

```bash
make hot-reload-test
```

This will:
1. Test with `max_amount=5000` (blocks $7500)
2. Update policy to `max_amount=10000`
3. Reload and verify $7500 is now allowed
4. Restore original policy

## 5. View Audit Logs

```bash
tail -f logs/aegis.log
```

Each log entry includes:
- `trace_id`: Request correlation
- `agent_id`: Who made the request
- `decision_allow`: Policy result (true/false)
- `reason`: Human-readable explanation
- `params_hash`: SHA-256 of request body
- `latency_ms`: Performance metric

## 6. Manual API Testing

### Health Check
```bash
curl http://localhost:8080/health
```

### Blocked Payment
```bash
curl -H "X-Agent-ID: finance-agent" \
  -X POST http://localhost:8080/tools/payments/create \
  -d '{"amount":50000,"currency":"USD","vendor_id":"V99"}'
```

### Allowed Payment
```bash
curl -H "X-Agent-ID: finance-agent" \
  -X POST http://localhost:8080/tools/payments/create \
  -d '{"amount":1000,"currency":"USD","vendor_id":"V42"}'
```

### Manual Policy Reload
```bash
# Edit policies/finance-policy.yaml, then:
curl -X POST http://localhost:8080/policies/reload
```

## 7. Docker with Observability

```bash
cd deploy
docker-compose up --build
```

Access:
- Gateway: http://localhost:8080
- Jaeger UI: http://localhost:16686
- View distributed traces with full span details

## 8. Cleanup

```bash
make clean
# or
make docker-down
```

## Common Commands

| Command | Description |
|---------|-------------|
| `make run` | Start the gateway locally |
| `make build` | Build the binary |
| `make demo` | Run demo script |
| `make hot-reload-test` | Test policy hot-reload |
| `make docker-up` | Start with Docker |
| `make docker-logs` | View container logs |
| `make clean` | Clean build artifacts |

## Troubleshooting

**Port already in use:**
```bash
lsof -ti:8080 | xargs kill -9
```

**Policies not loading:**
- Check `policies/*.yaml` syntax
- View logs for validation errors
- Use `curl -X POST http://localhost:8080/policies/reload`

**Adapters not responding:**
- Verify ports 8081 and 8082 are free
- Check logs for adapter startup messages

## Next Steps

1. **Add new policy conditions** - Edit `internal/policy/policy.go`
2. **Create new tool adapters** - Follow patterns in `internal/adapters/`
3. **Extend telemetry** - Add custom spans and metrics
4. **Deploy to production** - Use provided Dockerfile and K8s configs

See [README.md](README.md) for comprehensive documentation.
