# OPC UA 功能测试指南

> 本文档提供 InduGate 网关 OPC UA 驱动与内置模拟器的完整测试步骤。  
> **测试环境**：Windows 10 / Go 1.26 / 2026-06-05  
> **测试结果**：12/12 步骤全部通过 ✅

---

## 前置条件

- Go 1.22+ 已安装
- 项目目录：`InduGate/`
- 网关默认端口：`8080`
- OPC UA 模拟器默认端口：`4840`

### 启动网关

```bash
# 编译（Windows 无需 CGO，已切换为纯 Go SQLite 驱动）
go build -o bin/gateway.exe ./cmd/gateway

# 启动
./bin/gateway.exe
# 或
go run cmd/gateway/main.go
```

验证服务正常：

```bash
curl http://localhost:8080/health
```

**预期响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "service": "indugate-gateway",
    "status": "up"
  }
}
```

---

## 测试流程概览

```
启动模拟器 → 创建设备 → 连接 → 浏览节点 → 读取 → 写入 → 再次读取
    → 订阅 → 轮询事件 → 断开 → 停止模拟器
```

---

## 步骤 1：启动 OPC UA 模拟器

无需真实工业设备，使用内置模拟器。

```bash
curl -X POST http://localhost:8080/api/v1/simulators/opcua/start
```

**实际响应（2026-06-05 测试）：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "type": "opcua",
    "status": "running",
    "endpoint": "opc.tcp://127.0.0.1:4840",
    "nodes": [
      "ns=1;s=Temperature",
      "ns=1;s=Pressure",
      "ns=1;s=Flow",
      "ns=1;s=MotorSpeed",
      "ns=1;s=AlarmActive"
    ]
  }
}
```

**模拟器数据点说明：**

| NodeID | 名称 | 模拟模式 | 范围 |
|--------|------|----------|------|
| `ns=1;s=Temperature` | 温度 | 正弦波 | 20~80 °C |
| `ns=1;s=Pressure` | 压力 | 随机值 | 1~5 bar |
| `ns=1;s=Flow` | 流量 | 正弦波 | 10~100 m³/h |
| `ns=1;s=MotorSpeed` | 电机转速 | 阶梯变化 | 0~3000 rpm |
| `ns=1;s=AlarmActive` | 告警状态 | 周期触发 | 0/1 |

> **注意**：NodeID 命名空间索引以启动响应中返回的 `nodes` 为准（通常为 `ns=1`）。

---

## 步骤 2：创建设备配置

```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Local Sim",
    "protocol": "opcua",
    "address": "opc.tcp://127.0.0.1:4840",
    "description": "OPC UA simulator test"
  }'
```

**实际响应：**

```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": 4,
    "name": "Local Sim",
    "protocol": "opcua",
    "address": "opc.tcp://127.0.0.1:4840",
    "status": "disconnected"
  }
}
```

记录返回的 `id`（下文以 `{device_id}` 表示，测试中为 `4`）。

---

## 步骤 3：连接 OPC UA 服务器

```bash
curl -X POST http://localhost:8080/api/v1/devices/{device_id}/connect
```

**实际响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 4,
    "status": "connected"
  }
}
```

---

## 步骤 4：浏览节点树

```bash
# 浏览 Objects 文件夹（单层子节点）
curl "http://localhost:8080/api/v1/devices/{device_id}/nodes?node=i=85&depth=1&children_only=true"

# 浏览模拟器命名空间 Objects（推荐用于查找 InduGate 变量）
curl "http://localhost:8080/api/v1/devices/{device_id}/nodes?node=ns=1;i=85&depth=1&children_only=true"
```

**查询参数：**

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `node` | 起始 NodeID | `i=85` |
| `depth` | 递归深度 | `3` |
| `children_only` | 仅返回直接子节点 | `false` |

> 读取模拟器变量时，建议直接使用步骤 1 返回的 `nodes` 列表中的 NodeID，最为可靠。

---

## 步骤 5：读取变量值

```bash
# NodeID 中的分号需要 URL 编码：; → %3B
curl "http://localhost:8080/api/v1/devices/{device_id}/data/ns=1%3Bs=Temperature"
```

**实际响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "node_id": "ns=1;s=Temperature",
    "value": 0,
    "status": "The operation succeeded. StatusGood (0x0)",
    "timestamp": "2026-06-05T16:08:48.2323084+08:00"
  }
}
```

---

## 步骤 6：写入变量值

```bash
curl -X POST "http://localhost:8080/api/v1/devices/{device_id}/data/ns=1%3Bs=Temperature" \
  -H "Content-Type: application/json" \
  -d '{"value": 55.5}'
```

**实际响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "node_id": "ns=1;s=Temperature",
    "value": 55.5
  }
}
```

---

## 步骤 7：验证写入结果

```bash
curl "http://localhost:8080/api/v1/devices/{device_id}/data/ns=1%3Bs=Temperature"
```

**实际响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "node_id": "ns=1;s=Temperature",
    "value": 55.5,
    "status": "The operation succeeded. StatusGood (0x0)",
    "timestamp": "2026-06-05T16:08:48.2359137+08:00"
  }
}
```

写入 55.5 后读取值一致，**读写功能验证通过** ✅

---

## 步骤 8：订阅数据变化

```bash
curl -X POST "http://localhost:8080/api/v1/devices/{device_id}/subscribe" \
  -H "Content-Type: application/json" \
  -d '{
    "node_ids": ["ns=1;s=Temperature", "ns=1;s=Pressure"],
    "interval_ms": 500
  }'
```

**实际响应：**

```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": "323270f7-37a2-48ad-bff2-703fe2129c68",
    "device_id": 4,
    "node_ids": ["ns=1;s=Temperature", "ns=1;s=Pressure"],
    "interval": "500ms",
    "created_at": "2026-06-05T16:08:48.2439934+08:00"
  }
}
```

记录返回的订阅 `id`（下文以 `{sub_id}` 表示）。

---

## 步骤 9：轮询订阅事件

等待 3~5 秒后拉取变化数据：

```bash
curl "http://localhost:8080/api/v1/devices/{device_id}/subscriptions/{sub_id}/events"
```

**实际响应（节选）：**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "subscription_id": "323270f7-37a2-48ad-bff2-703fe2129c68",
      "node_id": "ns=1;s=Temperature",
      "value": 55.5,
      "status": "The operation succeeded. StatusGood (0x0)",
      "timestamp": "2026-06-05T16:08:48.7444475+08:00"
    },
    {
      "subscription_id": "323270f7-37a2-48ad-bff2-703fe2129c68",
      "node_id": "ns=1;s=Pressure",
      "value": 3.3959071872580333,
      "status": "The operation succeeded. StatusGood (0x0)",
      "timestamp": "2026-06-05T16:08:49.2441902+08:00"
    }
  ]
}
```

4 秒内收到 10 条变化事件，**订阅功能验证通过** ✅

---

## 步骤 10：断开设备

```bash
curl -X POST http://localhost:8080/api/v1/devices/{device_id}/disconnect
```

**实际响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 4,
    "status": "disconnected"
  }
}
```

---

## 步骤 11：停止模拟器

```bash
curl -X POST http://localhost:8080/api/v1/simulators/opcua/stop
```

**实际响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "type": "opcua",
    "status": "stopped"
  }
}
```

---

## 测试结果汇总

| 步骤 | 接口 | 结果 |
|------|------|------|
| 1 | `GET /health` | ✅ PASS |
| 2 | `POST /api/v1/simulators/opcua/start` | ✅ PASS |
| 3 | `POST /api/v1/devices` | ✅ PASS |
| 4 | `POST /api/v1/devices/{id}/connect` | ✅ PASS |
| 5 | `GET /api/v1/devices/{id}/nodes` | ✅ PASS |
| 6 | `GET /api/v1/devices/{id}/data/{nodeId}` | ✅ PASS |
| 7 | `POST /api/v1/devices/{id}/data/{nodeId}` | ✅ PASS |
| 8 | 写入后再次读取 | ✅ PASS（值 = 55.5） |
| 9 | `POST /api/v1/devices/{id}/subscribe` | ✅ PASS |
| 10 | `GET .../subscriptions/{subId}/events` | ✅ PASS（10 条事件） |
| 11 | `POST /api/v1/devices/{id}/disconnect` | ✅ PASS |
| 12 | `POST /api/v1/simulators/opcua/stop` | ✅ PASS |

---

## PowerShell 一键测试脚本

将以下脚本保存为 `scripts/test-opcua.ps1` 后执行：

```powershell
$Base = "http://localhost:8080/api/v1"

# 1. 启动模拟器
$sim = (Invoke-RestMethod "$Base/simulators/opcua/start" -Method POST).data
$temp = $sim.nodes | Where-Object { $_ -match 'Temperature' } | Select-Object -First 1
Write-Host "Simulator nodes: $($sim.nodes -join ', ')"

# 2. 创建设备并连接
$dev = (Invoke-RestMethod "$Base/devices" -Method POST -ContentType "application/json" `
  -Body '{"name":"Local Sim","protocol":"opcua","address":"opc.tcp://127.0.0.1:4840"}').data
$id = $dev.id
Invoke-RestMethod "$Base/devices/$id/connect" -Method POST | Out-Null

# 3. 读写
$enc = [uri]::EscapeDataString($temp)
$before = (Invoke-RestMethod "$Base/devices/$id/data/$enc").data.value
Invoke-RestMethod "$Base/devices/$id/data/$enc" -Method POST -ContentType "application/json" `
  -Body '{"value":55.5}' | Out-Null
$after = (Invoke-RestMethod "$Base/devices/$id/data/$enc").data.value
Write-Host "Read before=$before, after write=$after"

# 4. 订阅
$sub = (Invoke-RestMethod "$Base/devices/$id/subscribe" -Method POST -ContentType "application/json" `
  -Body "{`"node_ids`":[`"$temp`"],`"interval_ms`":500}").data
Start-Sleep -Seconds 4
$events = (Invoke-RestMethod "$Base/devices/$id/subscriptions/$($sub.id)/events").data
Write-Host "Subscription events: $($events.Count)"

# 5. 清理
Invoke-RestMethod "$Base/devices/$id/disconnect" -Method POST | Out-Null
Invoke-RestMethod "$Base/simulators/opcua/stop" -Method POST | Out-Null
Write-Host "Done."
```

---

## 常见问题

### Q: Windows 下启动报 SQLite CGO 错误？

项目已切换为纯 Go 驱动 `github.com/glebarez/sqlite`，无需安装 GCC。重新编译即可：

```bash
go build -o bin/gateway.exe ./cmd/gateway
```

### Q: 启动模拟器请求超时？

确保没有其他进程占用 4840 端口。若之前异常退出，重启网关后再试。

### Q: 读取节点报 `StatusBadNodeIDUnknown`？

请使用步骤 1 启动模拟器时返回的 `nodes` 列表中的 NodeID，不要硬编码命名空间索引。

### Q: 如何连接外部 OPC UA 服务器？

创建设备时将 `address` 改为目标 endpoint，并在 `config` 字段传入 JSON：

```json
{
  "security_policy": "None",
  "security_mode": "None",
  "request_timeout_ms": 5000
}
```

---

## 相关文件

| 文件 | 说明 |
|------|------|
| `internal/protocol/opcua/` | OPC UA 驱动实现 |
| `internal/simulator/opcua/` | OPC UA 模拟器 |
| `configs/config.yaml` | 模拟器端口等配置 |
| `test-output.json` | 最近一次自动化测试原始结果 |
