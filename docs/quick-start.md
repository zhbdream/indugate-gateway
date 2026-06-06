# InduGate 快速开始

本指南帮助你在 5 分钟内启动 InduGate 工业智能体协议网关，并完成第一次设备连接与数据读取。

## 方式一：Docker 一键启动（推荐）

**前置条件**：Docker 20+ 与 Docker Compose v2

```bash
# 克隆项目
git clone https://gitee.com/zhbdream/indugate-gateway.git
cd indugate-gateway

# 一条命令启动（构建镜像 + 启动服务）
docker compose up -d --build

# 查看启动日志
docker compose logs -f
```

启动完成后访问：

| 服务 | 地址 |
|------|------|
| **Web 管理面板** | http://localhost:8080 |
| REST API | http://localhost:8080/api/v1 |
| 健康检查 | http://localhost:8080/health |
| MCP 服务发现 | http://localhost:8080/mcp/.well-known/mcp.json |

> Docker 镜像内置 SQLite 数据库，并自动启动 OPC UA / Modbus / MQTT 三个模拟器。

### 停止服务

```bash
docker compose down
```

---

## 方式二：本地开发

### 前置条件

- Go 1.23+
- Node.js 18+

### 1. 启动后端

```bash
# 安装 Go 依赖
make deps

# 创建数据目录
mkdir -p data

# 启动网关（端口 8080）
make run
```

### 2. 启动前端（开发模式）

```bash
cd web
npm install
npm run dev
```

前端开发服务器：http://localhost:3000（API 自动代理到 8080）

### 3. 生产模式（前后端一体）

```bash
cd web && npm install && npm run build && cd ..
make run
# 访问 http://localhost:8080
```

---

## 5 分钟体验流程

### 步骤 1：打开 Web 面板

浏览器访问 http://localhost:8080 ，进入 **模拟器** 页面，确认三个模拟器均为「运行中」。

若使用本地开发且未自动启动，点击各模拟器的 **启动** 按钮。

### 步骤 2：添加 Modbus 设备

进入 **设备管理** → **添加设备**：

| 字段 | 值 |
|------|-----|
| 名称 | Modbus Demo |
| 协议 | Modbus TCP |
| 地址 | 127.0.0.1:502 |
| 配置 | `{"unit_id":1}` |

> Docker 部署时，若 Gateway 在容器内，Modbus 地址填 `127.0.0.1:502` 即可（模拟器与网关在同一个容器）。

### 步骤 3：连接设备

在设备列表点击 **连接**，状态变为「已连接」。

### 步骤 4：查看实时数据

点击 **数据** 进入设备详情 → **浏览节点** → 选择 `holding:0` 查看温度寄存器实时值。

开启 **自动刷新** 可每 2 秒更新数据。

### 步骤 5：MQTT 体验（可选）

添加 MQTT 设备：

| 字段 | 值 |
|------|-----|
| 名称 | MQTT Demo |
| 协议 | MQTT |
| 地址 | tcp://127.0.0.1:1883 |
| 配置 | `{"client_id":"web-demo","qos":1,"topics":["factory/device1/telemetry"]}` |

连接后浏览节点，读取 topic `factory/device1/telemetry`。

---

## MCP Agent 接入

InduGate 内置 MCP Server，Agent 可通过标准 JSON-RPC 调用工具：

```bash
# 服务发现
curl http://localhost:8080/mcp/.well-known/mcp.json

# 列出工具
curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

可用工具：`list_devices`、`read_data`、`write_data`、`subscribe_data`

---

## 完整部署栈（可选）

如需 PostgreSQL + InfluxDB + 外部 Mosquitto：

```bash
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

---

## 常见问题

**Q: 端口 8080 被占用？**

```bash
# 修改 docker-compose.yml 端口映射
ports:
  - "9090:8080"
```

**Q: 设备连接失败？**

1. 确认对应协议模拟器已启动
2. 检查设备地址和 config 配置
3. 查看日志：`docker compose logs -f`

**Q: Web 页面空白？**

确认 `web/dist` 目录存在。Docker 镜像已内置前端；本地开发需先 `cd web && npm run build`。

---

## 下一步

- 阅读 [README.md](../README.md) 了解完整功能
- 查看 [OPC UA 测试指南](./opcua-test-guide.md)
- 参考 [examples/mcp-client](../examples/mcp-client/) 接入 AI Agent
