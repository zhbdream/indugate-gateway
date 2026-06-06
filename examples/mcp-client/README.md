# MCP Client Example

## Python（推荐）

```bash
# 确保 Gateway 已启动且设备已连接
python examples/mcp-client/mcp_tools.py --list-tools
python examples/mcp-client/mcp_tools.py --read 1 "ns=1;s=Temperature"
```

## curl

```bash
curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

See [docs/opcua-test-guide.md](../../docs/opcua-test-guide.md) for full workflow.
