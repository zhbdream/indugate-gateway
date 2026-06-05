# InduGate Grafana 监控

## 前置条件

1. 在 `configs/config.yaml` 中启用指标：

```yaml
metrics:
  enabled: true
```

2. 启动 InduGate 后访问 `http://localhost:8080/metrics` 确认有指标输出。

## Prometheus

使用 `prometheus.yml` 抓取 InduGate 指标。Docker 环境下目标地址为 `host.docker.internal:8080`；本地 Prometheus 可改为 `localhost:8080`。

## Grafana 仪表盘

1. 添加 Prometheus 数据源
2. 导入 `dashboard-indugate.json`
3. 面板包含：设备总数、已连接设备、活跃告警、HTTP 请求速率

## 可选：Docker Compose 监控栈

```bash
docker compose -f deployments/grafana/docker-compose.monitoring.yml up -d
```

- Grafana: http://localhost:3001 (admin/admin)
- Prometheus: http://localhost:9090
