# InduGate - 工业智能体协议网关

> AI 智能体与工业设备之间的翻译官 — 让任何 Agent 都能通过标准 MCP 协议对接真实工业设备

[![Docker](https://img.shields.io/badge/docker-compose-up-blue)](docker-compose.yml)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8)](go.mod)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D)](web/)

## 功能特性

- **多协议支持**：OPC UA、Modbus TCP、MQTT（S7 规划中）
- **MCP 协议接入**：标准 Model Context Protocol，Agent 开箱即用
- **设备模拟器**：内置 OPC UA / Modbus / MQTT 模拟器，零硬件依赖
- **Web 管理面板**：设备管理、连接控制、实时数据、模拟器管理
- **一键部署**：Docker Compose 一条命令启动完整系统

## 快速开始

```bash
git clone https://github.com/your-org/InduGate.git
cd InduGate
docker compose up -d --build
```

打开浏览器访问 **http://localhost:8080**

详细步骤见 [快速开始文档](docs/quick-start.md)

## 访问地址

| 服务 | 地址 |
|------|------|
| Web 管理面板 | http://localhost:8080 |
| REST API | http://localhost:8080/api/v1 |
| Swagger API 文档 | http://localhost:8080/swagger/index.html |
| 健康检查 | http://localhost:8080/health |
| MCP 服务发现 | http://localhost:8080/mcp/.well-known/mcp.json |

## Web 管理面板

基于 Vue 3 + Element Plus 构建，提供：

- **设备管理** — 增删改查、连接/断开
- **数据监控** — 浏览节点、实时读取、写入、订阅
- **模拟器控制** — 一键启停 OPC UA / Modbus / MQTT 模拟器

### 本地开发

```bash
# 终端 1：后端
make deps && mkdir -p data && make run

# 终端 2：前端
cd web && npm install && npm run dev
# 访问 http://localhost:3000
```

## MCP 工具

| 工具 | 说明 |
|------|------|
| `list_devices` | 列出所有设备 |
| `get_device_info` | 获取设备详情 |
| `read_data` | 读取数据点 |
| `write_data` | 写入数据点 |
| `subscribe_data` | 订阅数据变化 |

```bash
curl -X POST http://localhost:8080/mcp/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## API 概览

```
# 设备管理
GET/POST        /api/v1/devices
GET/PUT/DELETE  /api/v1/devices/{id}
POST            /api/v1/devices/{id}/connect
POST            /api/v1/devices/{id}/disconnect

# 数据操作
GET             /api/v1/devices/{id}/nodes
GET             /api/v1/devices/{id}/data/{nodeId}
POST            /api/v1/devices/{id}/data/{nodeId}
POST            /api/v1/devices/{id}/subscribe

# 历史数据
GET             /api/v1/devices/{id}/data/history

# 模拟器
GET             /api/v1/simulators
POST            /api/v1/simulators/{type}/start
POST            /api/v1/simulators/{type}/stop

# 告警与仪表盘
GET/POST        /api/v1/alerts/rules
GET             /api/v1/alerts/events
POST            /api/v1/alerts/events/{id}/acknowledge
GET             /api/v1/dashboard/stats
```

## 技术栈

| 组件 | 选型 |
|------|------|
| 后端 | Go 1.23+ / Gin / GORM / Zap / Viper |
| 前端 | Vue 3 / Vite / Element Plus / TypeScript |
| 数据库 | SQLite（默认）/ PostgreSQL（生产） |
| 部署 | Docker / Docker Compose |

## 项目结构

```
InduGate/
├── cmd/gateway/           # 应用入口
├── internal/
│   ├── api/               # HTTP API + 静态文件服务
│   ├── mcp/               # MCP Server
│   ├── protocol/          # 协议驱动
│   ├── simulator/         # 设备模拟器
│   └── service/           # 业务服务
├── web/                   # Vue 3 前端
├── configs/               # 配置文件
├── deployments/docker/    # Docker 构建文件
├── docker-compose.yml     # 一键启动（推荐）
└── docs/                  # 文档
```

## Docker 部署

### 一键启动（推荐）

```bash
docker compose up -d --build    # 启动
docker compose logs -f          # 日志
docker compose down             # 停止
```

包含：Gateway + Web UI + SQLite + 内置模拟器

### 完整栈（PostgreSQL + InfluxDB + Mosquitto）

```bash
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

## 配置

主配置文件：`configs/config.yaml`

环境变量前缀 `INDUGATE_`：

```bash
export INDUGATE_SERVER_PORT=9090
export INDUGATE_DATABASE_DRIVER=sqlite
export INDUGATE_SIMULATOR_MODBUS_AUTO_START=true
```

### 告警通知

```yaml
alerts:
  webhook_url: "https://hooks.example.com/alerts"
  mqtt_enabled: true
  mqtt_broker: "tcp://localhost:1883"
  mqtt_topic: "indugate/alerts"
```

### 历史数据保留

```yaml
history:
  retention_days: 30          # 超过 30 天的 SQLite 历史自动清理
  cleanup_interval_hours: 24
```

### API 认证（可选）

生产环境可启用 Bearer Token 保护 `/api/v1` 与 `/mcp`：

```yaml
auth:
  enabled: true
  api_token: "your-secret-token"
```

请求头：`Authorization: Bearer your-secret-token`

Web 面板右上角 **API Token** 可配置静态 Token；JWT 登录使用 `/login` 页面。

### JWT 登录

```yaml
auth:
  enabled: true
  jwt_secret: "change-me-in-production"
  jwt_expire_hours: 24
  default_admin_user: "admin"
  default_admin_password: "admin123"
```

- `POST /api/v1/auth/login` — 获取 JWT
- `GET /api/v1/auth/me` — 当前用户信息

### RBAC 角色

| 角色 | 权限 |
|------|------|
| admin | 全部 API + 用户管理 |
| operator | 设备/数据/告警/模拟器读写 |
| viewer | 只读 GET 请求 |

### 设备级权限

```yaml
auth:
  enabled: true
  device_acl_enabled: true
```

启用后，非 admin 用户仅能访问被分配的设备（`PUT /api/v1/users/:id/devices`）。Web 用户管理页可配置。

BACnet 设备可启用 COV 订阅：

```json
{"device_id": 1234, "cov_enabled": true, "cov_lifetime_sec": 300}
```

### Prometheus

```yaml
metrics:
  enabled: true
```

访问 `GET /metrics` 获取 `indugate_devices_total`、`indugate_active_alerts` 等指标。

Grafana 仪表盘模板见 `deployments/grafana/`。

### 操作审计

启用认证后自动记录写操作（`audit.enabled`，默认 true）：

```yaml
audit:
  enabled: true
  retention_days: 90
```

- `GET /api/v1/audit/logs` — 查询审计日志（admin）
- Web 面板 `/audit` — 操作审计页

## 开发命令

```bash
make help        # 查看所有命令
make build       # 编译后端
make run         # 本地运行
make test        # 运行测试
make docker-up   # Docker 启动（完整栈）
```

## License

MIT
