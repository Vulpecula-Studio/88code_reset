# 88code 订阅自动重置工具

自动化管理你的 88code.org 订阅积分额度。

## 功能特性

- 🔄 每天自动重置积分额度（18:50 和 23:55）
- 🛡️ PAYGO 保护机制，确保安全
- 🌏 支持自定义时区
- 📊 完整的日志记录和状态监控
- 🔒 智能检查和间隔控制

## 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `API_KEY` | ✅ 是 | 无 | 你的 88code.org API Key |
| `TARGET_PLANS` | ❌ 否 | `FREE` | 目标套餐类型（FREE/PRO/PLUS） |
| `TZ` | ❌ 否 | `Asia/Shanghai` | 时区设置 |

## 获取 API Key

1. 访问 [88code.org](https://88code.org)
2. 登录你的账号
3. 前往设置页面获取 API Key

## 使用说明

部署完成后，程序会自动运行：
- 每天 18:50 和 23:55 自动检查并重置积分
- 每小时检查一次订阅状态
- 所有操作日志可在 Zeabur 的日志面板查看

## 项目链接

- GitHub: https://github.com/Vulpecula-Studio/88code_reset
- 文档: https://github.com/Vulpecula-Studio/88code_reset#readme
- Issues: https://github.com/Vulpecula-Studio/88code_reset/issues

## 许可证

MIT License
