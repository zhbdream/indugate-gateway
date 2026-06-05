# Python Integration Example

Run the example against a running InduGate instance:

```bash
# Start gateway
go run cmd/gateway/main.go

# Run example (requires Python 3.8+)
python examples/python-integration/example.py
```

Environment variables:

- `INDUGATE_URL` — base URL (default: `http://localhost:8080`)

The script demonstrates REST API calls (devices, dashboard stats) and MCP `list_devices`.
