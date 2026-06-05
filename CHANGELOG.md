# Changelog

All notable changes to InduGate are documented in this file.

## [0.6.1] - 2026-06-05

### Changed
- MCP tools/resources respect device ACL when `auth.device_acl_enabled` is on

## [0.6.0] - 2026-06-05

### Added
- Device-level permission isolation (`auth.device_acl_enabled`)
- User device assignment API (`GET/PUT /api/v1/users/:id/devices`)
- BACnet COV subscription mode (`cov_enabled` in device config, fallback to poll)
- Web UI: assign devices to operator/viewer users when ACL is enabled

## [0.5.0] - 2026-06-05

### Added
- Operation audit log: middleware records mutating API calls when `auth.enabled`
- Audit log query API (`GET /api/v1/audit/logs`, admin only) and Web UI (`/audit`)
- Audit retention cleanup (`audit.retention_days`)
- Grafana dashboard template and optional Prometheus/Grafana compose stack
- Frontend role-aware UI: hide write actions for `viewer` role

### Changed
- Header shows current username and role after JWT login

## [0.4.0] - 2026-06-05

### Added
- RBAC enforcement: admin / operator / viewer API permissions
- User management API and Web UI (`/users`, admin only)
- Prometheus metrics endpoint (`/metrics`, optional `metrics.enabled`)

### Changed
- Static API token authenticates as admin role when RBAC is enabled

## [0.3.0] - 2026-06-05

### Added
- JWT authentication with default admin bootstrap and login API
- Web login page with route guard (`/login`)
- BACnet/IP driver (UDP Read/Write Present_Value, polling subscribe)
- User model with admin/operator/viewer roles (RBAC foundation)

### Changed
- Auth middleware accepts static API token or JWT

## [0.2.2] - 2026-06-05

### Added
- Siemens S7 driver (gos7): DB/M/I/Q read-write, browse catalog, polling subscribe
- Web UI API Token settings (localStorage + axios Bearer header)

## [0.2.1] - 2026-06-05

### Added
- Alert change_rate condition with per-node sample tracking
- Alert deduplication (one active event per rule until acknowledged)
- Alert notifications via Webhook and MQTT (`alerts.*` config)
- History retention auto-cleanup (`history.retention_days`)
- Optional API Bearer token auth (`auth.enabled`)

### Changed
- Alert evaluation triggers external notifications asynchronously

## [0.2.0] - 2026-06-05

### Added
- Alert engine with rules CRUD, event listing, and acknowledge API
- Dashboard stats API (`GET /api/v1/dashboard/stats`)
- Optional InfluxDB telemetry writer (config: `influxdb.enabled`)
- History CSV export (`GET /api/v1/devices/{id}/data/history/export`)
- Web dashboard and alerts management pages with ECharts trend charts
- OPC UA driver integration test with built-in simulator
- Runnable Python integration example (`examples/python-integration/example.py`)

### Changed
- Read/subscribe data paths now record history, evaluate alerts, and optionally write to InfluxDB

## [0.1.0] - 2026-06-04

### Added
- Initial release: OPC UA, Modbus TCP, MQTT drivers
- Device management REST API and MCP Server
- Protocol simulators and Vue 3 web panel
- SQLite data history, Swagger docs, Docker one-command deploy
