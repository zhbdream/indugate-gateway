# MCP Client Example

InduGate 内置 MCP Server，支持标准 JSON-RPC 2.0 协议。

## 端点

- 服务发现: `GET /mcp/.well-known/mcp.json`
- JSON-RPC: `POST /mcp/message`

## 快速测试

```bash
# 初始化
curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {"name": "example", "version": "1.0.0"}
    }
  }'

# 列出工具
curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'

# 列出设备
curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "list_devices",
      "arguments": {}
    }
  }'
```

## Cursor / Claude Desktop 配置

```json
{
  "mcpServers": {
    "indugate": {
      "url": "http://localhost:8080/mcp/message"
    }
  }
}
```
