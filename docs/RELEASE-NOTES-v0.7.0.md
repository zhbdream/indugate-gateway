# InduGate v0.7.0 — First Open Source Release

**InduGate** (Industrial Agent Gateway) connects AI agents to industrial equipment via the **Model Context Protocol (MCP)**.

## Highlights

- **MCP Server** with tools: `list_devices`, `read_data`, `write_data`, `subscribe_data`, `get_device_info`
- **5 protocol drivers**: OPC UA, Modbus TCP, MQTT, Siemens S7, BACnet/IP
- **Built-in simulators**: OPC UA, Modbus, MQTT — no hardware required
- **Web UI**: device management, alerts, dashboard, simulators
- **One-command deploy**: `docker compose up -d`

## Quick Start

```bash
git clone https://gitee.com/zhbdream/indugate-gateway.git
cd indugate-gateway
docker compose up -d --build
```

Open http://localhost:8080

## MCP Example

```bash
python examples/mcp-client/mcp_tools.py --list-tools
```

## Documentation

- [Quick Start](docs/quick-start.md)
- [Architecture](docs/architecture.md)
- [OPC UA Test Guide](docs/opcua-test-guide.md)
- [Contributing](CONTRIBUTING.md)

## Security Note

Authentication is **disabled by default** for easy evaluation. For production, enable `auth.enabled` in `configs/config.yaml` and change default credentials.

## Full Changelog

See [CHANGELOG.md](../CHANGELOG.md)
