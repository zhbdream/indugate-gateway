# InduGate 架构说明

InduGate（Industrial Agent Gateway）是面向 **AI Agent ↔ 工业设备** 的协议网关：南向对接多种工业协议，北向通过 **MCP（Model Context Protocol）** 暴露统一能力。

## 分层架构

```
┌─────────────────────────────────────────────────────────┐
│  接入层    HTTP API / Web UI / MCP Server                 │
├─────────────────────────────────────────────────────────┤
│  业务层    设备管理 · 告警 · 历史数据 · 权限 · 审计      │
├─────────────────────────────────────────────────────────┤
│  协议层    OPC UA · Modbus · MQTT · S7 · BACnet         │
├─────────────────────────────────────────────────────────┤
│  数据层    SQLite / PostgreSQL · 可选 InfluxDB           │
└─────────────────────────────────────────────────────────┘
```

## 项目目录

```
InduGate/
├── cmd/gateway/           # 程序入口
├── internal/
│   ├── api/               # HTTP 路由、Handler、中间件
│   ├── mcp/               # MCP 协议实现（Tools / Resources / SSE）
│   ├── protocol/          # 工业协议驱动（按协议分子包）
│   ├── simulator/         # 内置设备模拟器
│   ├── service/           # 业务服务（设备、告警、历史、权限等）
│   ├── model/             # 数据模型（GORM）
│   ├── config/            # 配置加载（Viper）
│   ├── storage/           # 数据库初始化
│   └── metrics/           # Prometheus 指标
├── pkg/logger/            # 日志封装（Zap）
├── web/                   # Vue 3 管理面板
├── configs/               # 默认配置文件
├── deployments/           # Docker、Grafana 等部署资产
├── docs/                  # 用户与开发者文档
└── examples/              # 集成示例
```

## 核心数据流

### 1. Agent 通过 MCP 读设备

```
AI Agent → POST /mcp/message (tools/call read_data)
         → MCP Server → DeviceService → DriverManager
         → protocol/opcua (或其他驱动) → 工业设备
```

### 2. Web 管理设备

```
Browser → REST /api/v1/devices → DeviceService → GORM → SQLite/PostgreSQL
连接设备 → DriverManager.Connect() → 对应协议驱动建立会话
```

### 3. 数据历史与告警

```
读/订阅数据 → HistoryRecorder → SQLite 历史表
                              → 可选 InfluxDB
                              → AlertService 规则评估
```

## 协议驱动约定

每种协议在 `internal/protocol/<name>/` 下实现：

| 能力 | 说明 |
|------|------|
| Connect / Disconnect | 建立/释放与设备连接 |
| Browse | 浏览点位/节点/Topic |
| Read / Write | 读写数据 |
| Subscribe | 订阅变化（协议支持时） |

`internal/service/driver_manager.go` 按设备 `protocol` 字段路由到具体驱动，并维护「设备 ID → 活跃连接」映射。

## MCP 工具

| 工具 | 作用 |
|------|------|
| `list_devices` | 列出设备及连接状态 |
| `get_device_info` | 设备详情 |
| `read_data` | 读取点位 |
| `write_data` | 写入点位 |
| `subscribe_data` | 创建订阅（事件通过 REST 轮询） |

`viewer` 角色仅可使用只读类工具；写操作在 MCP 层与 REST RBAC 双层拦截。

## 配置

主配置：`configs/config.yaml`，环境变量前缀 `INDUGATE_`（如 `INDUGATE_SERVER_PORT`）。

认证默认关闭，便于开箱试用；生产环境请启用 `auth.enabled` 并修改默认密码，见 [quick-start.md](./quick-start.md)。

## 扩展新协议驱动

1. 在 `internal/protocol/<proto>/` 实现驱动（参考 `modbus` 或 `opcua`）
2. 在 `internal/model/device.go` 增加 `Protocol` 常量
3. 在 `DriverManager` 注册 Connect/Browse/Read/Write/Subscribe
4. 补充集成测试与 `docs/` 下的协议说明
5. 更新 MCP `tools.go` 中的协议枚举与说明

详见 [CONTRIBUTING.md](../CONTRIBUTING.md)。

## 部署模式

| 模式 | 文件 | 说明 |
|------|------|------|
| 单机试用 | 根目录 `docker-compose.yml` | SQLite + 内置模拟器 |
| 完整栈 | `deployments/docker/docker-compose.yml` | PostgreSQL + InfluxDB + Mosquitto |
| 监控 | `deployments/grafana/` | Prometheus + Grafana 模板 |

## 相关文档

- [快速开始](./quick-start.md)
- [OPC UA 测试指南](./opcua-test-guide.md)
- [示例代码](../examples/)
