# InduGate 快速开始

本指南帮助你在 5 分钟内启动 InduGate，并完成第一次设备连接与数据读取。

## 选择启动方式

| 方式 | 适合谁 | 命令 | Web 访问 |
|------|--------|------|----------|
| **Docker（推荐）** | 想最快体验、不想配环境 | `docker compose up -d --build` | http://localhost:8080 |
| **本地开发** | 改代码、调试前后端 | 后端 + 前端两个终端 | http://localhost:3000 |
| **本地一体** | 本地模拟生产、只开一个进程 | `npm run build` 后 `go run` | http://localhost:8080 |

---

## 方式一：Docker 一键启动（推荐）

**前置条件**：Docker 20+ 与 Docker Compose v2

```bash
git clone https://gitee.com/zhbdream/indugate-gateway.git
cd indugate-gateway
docker compose up -d --build
docker compose logs -f    # 查看日志，Ctrl+C 退出
```

| 服务 | 地址 |
|------|------|
| **Web 管理面板** | http://localhost:8080 |
| REST API | http://localhost:8080/api/v1 |
| Swagger | http://localhost:8080/swagger/index.html |
| 健康检查 | http://localhost:8080/health |
| MCP 发现 | http://localhost:8080/mcp/.well-known/mcp.json |

> Docker 镜像内置 SQLite、Web UI，并通过环境变量**自动启动** OPC UA / Modbus / MQTT 模拟器。

**验证：**

```powershell
curl http://localhost:8080/health
# 期望: {"status":"ok",...}
```

**停止：**

```bash
docker compose down
```

---

## 方式二：本地开发

### 前置条件

| 工具 | 版本要求 |
|------|----------|
| Go | 1.24+ |
| Node.js | **18+**（Vite 6 不支持 Node 16） |

**国内下载 Go 依赖**（若 `go mod download` 超时）：

```powershell
$env:GOPROXY="https://goproxy.cn,direct"
go mod download
```

> Windows 默认没有 `make`，请直接用下方命令，或使用 `.\scripts\dev.ps1`。

### 步骤 1：启动后端（终端 1）

```powershell
cd E:\project\InduGate          # 换成你的项目路径
$env:GOPROXY="https://goproxy.cn,direct"
if (-not (Test-Path data)) { mkdir data }
go run ./cmd/gateway
```

**启动成功的标志：**

```
INFO  gateway/main.go  gateway server starting  {"addr": "0.0.0.0:8080"}
```

此时终端会**占用前台**，不要关闭。

**可能出现的 WARN（正常）：**

```
WARN  web static directory not found, UI disabled  {"dir": "./web/dist"}
```

表示还没构建前端，`:8080` 暂时没有 Web 页面，但 **API 可用**。要看界面请执行步骤 2，或见「方式三」。

**验证 API：**

```powershell
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/devices
curl http://localhost:8080/swagger/index.html
```

### 步骤 2：启动前端（终端 2）

```powershell
node -v    # 必须 v18 / v20 / v22，不能是 v16
cd web
npm install
npm run dev
```

浏览器访问 **http://localhost:3000**（API 自动代理到 8080）。

快捷脚本：

```powershell
.\scripts\dev.ps1 backend    # 终端 1
.\scripts\dev.ps1 frontend   # 终端 2
```

### Linux / macOS

```bash
export GOPROXY=https://goproxy.cn,direct
make deps && mkdir -p data && make run    # 终端 1
cd web && npm install && npm run dev      # 终端 2
```

---

## 方式三：本地一体（单端口 8080）

适合本地不想开两个终端，前后端都走 8080：

```powershell
cd web
npm install
npm run build
cd ..
go run ./cmd/gateway
```

访问 http://localhost:8080 ，日志应出现 `web UI enabled`。

---

## 5 分钟体验流程

### 步骤 1：确认模拟器

- **Docker**：打开 http://localhost:8080 → **模拟器**，三个应已运行
- **本地开发**：http://localhost:3000 → **模拟器**，手动点 **启动**

### 步骤 2：添加 Modbus 设备

**设备管理** → **添加设备**：

| 字段 | 值 |
|------|-----|
| 名称 | Modbus Demo |
| 协议 | Modbus TCP |
| 地址 | 127.0.0.1:502 |
| 配置 | `{"unit_id":1}` |

### 步骤 3：连接并读数据

1. 点击 **连接**，状态变为「已连接」
2. 进入 **数据** → **浏览节点** → 选 `holding:0` 查看实时值

### 步骤 4：MQTT（可选）

| 字段 | 值 |
|------|-----|
| 协议 | MQTT |
| 地址 | tcp://127.0.0.1:1883 |
| 配置 | `{"client_id":"web-demo","qos":1,"topics":["factory/device1/telemetry"]}` |

---

## MCP Agent 接入

```bash
curl http://localhost:8080/mcp/.well-known/mcp.json

curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Python 示例：

```bash
python examples/mcp-client/mcp_tools.py --list-tools
```

---

## 常见问题

### Q: `make` 不是内部或外部命令（Windows）

Windows 没有 `make`，用：

```powershell
go run ./cmd/gateway
```

或 `.\scripts\dev.ps1 backend`。

### Q: `go mod download` 连接 proxy.golang.org 超时

```powershell
$env:GOPROXY="https://goproxy.cn,direct"
go mod download
```

### Q: `npm run dev` 报 `crypto.getRandomValues is not a function`

Node 版本过低（常见 v16）。升级到 **Node 18+** 后重试：

```powershell
node -v
cd web && npm install && npm run dev
```

### Q: 8080 端口被占用 `bind: Only one usage of each socket address`

已有 Gateway 在运行，或上次未退出。

```powershell
netstat -ano | findstr ":8080"
taskkill /PID <上一步看到的PID> /F
go run ./cmd/gateway
```

或直接使用已在运行的实例：`curl http://localhost:8080/health`

### Q: 启动后 WARN `web static directory not found`

本地未执行 `npm run build`，8080 无 Web 页面。**API 仍正常**。

- 开发：另开终端 `cd web && npm run dev` → 访问 **:3000**
- 一体：先 `cd web && npm run build` 再 `go run ./cmd/gateway` → 访问 **:8080**

### Q: Docker 下设备连接失败

1. 模拟器页面确认已启动
2. 检查地址与 config JSON
3. `docker compose logs -f` 查看日志

### Q: 读取点位 URL 怎么写

路径参数（常规，需 URL 编码）：

```
GET /api/v1/devices/1/data/ns=1%3Bs=Temperature
```

含 `/` 的 nodeId 用查询参数：

```
GET /api/v1/devices/1/data?node=your/node/id
```

---

## 完整部署栈（可选）

PostgreSQL + InfluxDB + Mosquitto：

```bash
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

---

## 下一步

- [README.md](../README.md) — 完整功能与配置
- [architecture.md](./architecture.md) — 架构说明
- [opcua-test-guide.md](./opcua-test-guide.md) — OPC UA 测试
- [examples/mcp-client](../examples/mcp-client/) — MCP 接入示例
