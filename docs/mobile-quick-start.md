# InduGate 移动伴侣快速开始

移动伴侣为**独立 uni-app 开源仓库**，与本网关通过 REST API 对接。

| 仓库 | 地址 |
|------|------|
| Gitee | https://gitee.com/zhbdream/indugate-uniapp |
| GitHub | https://github.com/zhbdream/indugate-uniapp |

克隆后阅读仓库内 `README.md` 获取完整说明。

## 最短路径

1. 启动网关（与 Web 相同）：

   ```bash
   git clone https://gitee.com/zhbdream/indugate-gateway.git
   cd indugate-gateway
   docker compose up -d --build
   ```

2. 克隆移动伴侣并运行：

   ```bash
   git clone https://gitee.com/zhbdream/indugate-uniapp.git
   ```

3. 用 **HBuilderX** 打开 `indugate-uniapp` 目录 → 运行到浏览器。

4. 登录页服务器地址填 `http://localhost:8080`，测试连接后进入应用。

## 真机调试

| 环境 | 服务器地址示例 |
|------|----------------|
| H5 本机 | `http://localhost:8080` |
| 手机同 WiFi | `http://<电脑局域网IP>:8080` |

确保防火墙放行 8080，且网关监听 `0.0.0.0`。

## 认证说明

- 网关默认 **未启用 JWT**：移动端显示「开放访问」，可直接进入。
- 启用 JWT 后：在登录页输入与 Web 相同的账号密码。

## 功能边界

移动端覆盖巡检场景（概览、告警确认、设备读写），**不包含**：用户管理、操作审计、模拟器、告警规则编辑。请使用 Web 管理面板。
