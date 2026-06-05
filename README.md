# InduGate - 工业智能体协议网关

> AI 智能体与工业设备之间的翻译官 — 让任何 Agent 都能通过标准 MCP 协议对接真实工业设备

## 功能特性

- **多协议支持**：OPC UA、Modbus TCP、MQTT、S7（规划中）
- **MCP 协议接入**：标准 Model Context Protocol，Agent 开箱即用
- **设备模拟器**：内置模拟器，零硬件依赖即可开发测试
- **Web 管理面板**：设备配置、数据监控、协议测试（前端待开发）
- **容器化部署**：Docker / Docker Compose 一键启动

## 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.22+ |
| Web 框架 | Gin |
| 配置管理 | Viper |
| 日志 | Zap |
| ORM | GORM |
| 数据库 | SQLite（开发）/ PostgreSQL（生产） |

## 快速开始

### 前置条件

- Go 1.22+
- Docker & Docker Compose（可选）

### 本地开发

```bash
# 安装依赖
make deps

# 启动服务（默认 SQLite，端口 8080）
make run

# 或
go run cmd/gateway/main.go
```

### Docker 部署

```bash
# 构建并启动全部服务（Gateway + PostgreSQL + InfluxDB + Mosquitto）
make docker-up

# 查看日志
make docker-logs

# 停止服务
make docker-down
```

## 访问地址

| 服务 | 地址 |
|------|------|
| 健康检查 | http://localhost:8080/health |
| REST API | http://localhost:8080/api/v1 |
| MCP 服务发现 | http://localhost:8080/mcp/.well-known/mcp.json |
| InfluxDB | http://localhost:8086 |
| MQTT Broker | mqtt://localhost:1883 |

## API 概览

```
GET    /api/v1/devices              # 设备列表
POST   /api/v1/devices              # 创建设备
GET    /api/v1/devices/{id}         # 设备详情
PUT    /api/v1/devices/{id}         # 更新设备
DELETE /api/v1/devices/{id}         # 删除设备
POST   /api/v1/devices/{id}/connect # 连接设备
POST   /api/v1/devices/{id}/disconnect # 断开设备

GET    /api/v1/simulators           # 模拟器列表
POST   /api/v1/simulators/{type}/start  # 启动模拟器
POST   /api/v1/simulators/{type}/stop   # 停止模拟器
```

## 项目结构

```
InduGate/
├── cmd/gateway/          # 应用入口
├── internal/
│   ├── api/              # HTTP API（路由、处理器、中间件）
│   ├── config/           # Viper 配置管理
│   ├── mcp/              # MCP 协议实现
│   ├── model/            # 数据模型
│   ├── protocol/         # 协议驱动（OPC UA / Modbus / MQTT / S7）
│   ├── service/          # 业务服务
│   ├── simulator/        # 设备模拟器
│   └── storage/          # 数据存储（GORM）
├── pkg/logger/           # Zap 日志封装
├── configs/              # 配置文件
├── deployments/docker/   # Docker 部署配置
├── web/                  # 前端项目（待开发）
└── docs/                 # 文档
```

## 配置说明

主配置文件：`configs/config.yaml`

支持环境变量覆盖，前缀为 `INDUGATE_`，例如：

```bash
export INDUGATE_SERVER_PORT=9090
export INDUGATE_DATABASE_DRIVER=postgres
export INDUGATE_DATABASE_DSN="host=localhost user=indugate password=xxx dbname=indugate port=5432 sslmode=disable"
```

## 开发命令

```bash
make help        # 查看所有命令
make build       # 编译二进制
make test        # 运行测试
make fmt         # 格式化代码
make lint        # 代码检查
make clean       # 清理构建产物
```

## License

MIT
