# Contributing to InduGate

感谢参与 InduGate 开源建设！

## 开始之前

- 阅读 [docs/architecture.md](docs/architecture.md) 了解项目结构
- 阅读 [docs/quick-start.md](docs/quick-start.md) 在本地跑通服务

## 开发环境

```bash
make deps
mkdir -p data
make run

# 前端（另开终端）
cd web && npm install && npm run dev
# http://localhost:3000
```

Windows 用户若 `go` 不在 PATH，请使用 Go 安装目录下的完整路径，或配置环境变量。

## 代码规范

- **Go**：`gofmt` / `go fmt ./...`（`make fmt`）
- **提交信息**：[Conventional Commits](https://www.conventionalcommits.org/)（`feat:` / `fix:` / `docs:` 等）
- **注释**：为 `package` 和对外导出的类型补充简短英文或中文包注释；避免对显而易见的代码堆砌行注释
- **测试**：新功能尽量附带 `*_test.go`；提交前执行 `go test ./...`

## 如何新增协议驱动

1. 在 `internal/protocol/<name>/` 创建驱动包，实现连接、浏览、读写、订阅
2. 在 `internal/model/device.go` 添加 `Protocol` 常量
3. 在 `internal/service/driver_manager.go` 注册该协议
4. （可选）在 `internal/simulator/<name>/` 添加模拟器
5. 补充集成测试与文档
6. 更新 MCP `internal/mcp/tools.go` 中的协议说明

参考实现：`internal/protocol/opcua/`、`internal/protocol/modbus/`。

## Pull Request 流程

1. Fork 仓库并创建功能分支
2. 实现改动并补充测试
3. 若变更 API 或行为，更新 `docs/` 或 `CHANGELOG.md`
4. 确认通过：
   ```bash
   go test ./...
   cd web && npm run build
   ```
5. 提交 PR，说明改动动机与测试方式

## 报告 Issue

请提供：

- InduGate 版本或 commit
- 复现步骤
- 期望与实际行为
- 相关日志（可脱敏）

## 发布维护（维护者）

- 版本记录写入 `CHANGELOG.md`
- GitHub Release 正文可参考 `docs/RELEASE-NOTES-v0.7.0.md` 模板
- 发布前确认 `configs/` 无真实密钥，默认 `auth.enabled: false`
