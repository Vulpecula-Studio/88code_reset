# Windows 10 Docker Compose 多 Key 部署指南

本指南将帮助你在 Windows 10 上使用 Docker Compose 部署 88code Reset 工具，并配置多个 API Key 统一管理。

## 前置要求

### 1. 安装 Docker Desktop for Windows

1. 下载 [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop/)
2. 安装并启动 Docker Desktop
3. 确保 Docker Desktop 正在运行（系统托盘会显示 Docker 图标）
4. 打开 PowerShell 或 CMD，验证安装：
   ```powershell
   docker --version
   docker-compose --version
   ```

### 2. 准备 API Keys

准备好你的所有 88code API Keys，格式如下：
- `sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`
- 可以有多个，用逗号分隔

---

## 快速开始（3 步部署）

### 步骤 1：创建配置文件

在项目根目录（`d:\88code_reset`）创建 `.env` 文件：

**方式 A：使用记事本创建**
```powershell
# 在项目目录打开 PowerShell
cd d:\88code_reset
notepad .env
```

**方式 B：复制示例文件**
```powershell
cd d:\88code_reset
copy .env.example .env
notepad .env
```

### 步骤 2：配置多个 API Keys

在 `.env` 文件中填入以下内容（**替换为你的实际 API Keys**）：

```env
# ============================================
# 多账号配置（推荐使用 API_KEYS）
# ============================================
# 方式1：使用 API_KEYS（支持多个，逗号分隔）
API_KEYS=sk-your-first-key-here,sk-your-second-key-here,sk-your-third-key-here

# 方式2：使用单个 API_KEY（仅单账号时使用）
# API_KEY=sk-your-single-key-here

# ============================================
# 时区配置
# ============================================
TIMEZONE=Asia/Shanghai

# ============================================
# 额度判断配置（可选）
# ============================================
# 上限模式：当额度 > 83% 时跳过18:50重置（保留到23:55）
CREDIT_THRESHOLD_MAX=83

# 下限模式：当额度 < 50% 时才执行18:50重置（取消注释启用）
# CREDIT_THRESHOLD_MIN=50

# ============================================
# 重置时间配置
# ============================================
# 是否启用18:50重置（默认关闭，只在23:55重置）
# 可选值：true, false, 1, 0, yes, no
ENABLE_FIRST_RESET=false

# ============================================
# 目标套餐配置（可选）
# ============================================
# 留空 = 所有 MONTHLY 套餐
# 指定套餐：FREE,PRO,PLUS（逗号分隔）
PLANS=
```

### 步骤 3：启动服务

#### 3.1 先测试配置（推荐）

```powershell
# 在项目目录
cd d:\88code_reset

# 运行测试模式，验证 API Keys 和配置
docker-compose run --rm reset-test
```

**测试成功的标志：**
- ✅ API 连接测试通过
- ✅ 获取到订阅列表
- ✅ 找到目标订阅
- ✅ 显示所有账号信息

#### 3.2 正式启动调度器

```powershell
# 后台启动服务
docker-compose up -d

# 查看日志（实时）
docker-compose logs -f

# 查看日志（最近100行）
docker-compose logs --tail=100
```

---

## 配置说明

### 多 Key 配置详解

#### 方式 1：API_KEYS（推荐 - 多账号）
```env
API_KEYS=key1,key2,key3,key4
```
- 支持任意数量的 API Keys
- 用英文逗号分隔
- 自动去重
- 并发调度所有账号

#### 方式 2：API_KEY（单账号）
```env
API_KEY=your-single-key
```
- 仅适用于单个账号
- 如果同时设置了 API_KEYS，会合并使用

### 重置时间配置

项目支持两个重置时间点：

| 时间点 | 默认状态 | 配置项 | 说明 |
|--------|---------|--------|------|
| 18:50 | 关闭 | `ENABLE_FIRST_RESET=true` | 第一次重置，可选 |
| 23:55 | 开启 | 无需配置 | 第二次重置，必选 |

**推荐配置：**
- 保守策略：只启用 23:55（`ENABLE_FIRST_RESET=false`）
- 激进策略：启用两次重置（`ENABLE_FIRST_RESET=true`）

### 额度判断模式

#### 上限模式（默认）
```env
CREDIT_THRESHOLD_MAX=83
```
- 当额度 > 83% 时，跳过 18:50 重置
- 保留重置机会到 23:55
- 适合额度充足的情况

#### 下限模式
```env
CREDIT_THRESHOLD_MIN=50
# 注释掉上限配置
# CREDIT_THRESHOLD_MAX=83
```
- 当额度 < 50% 时，才执行 18:50 重置
- 适合额度紧张的情况

### 目标套餐配置

```env
# 所有 MONTHLY 套餐（默认）
PLANS=

# 仅 FREE 套餐
PLANS=FREE

# 多个套餐
PLANS=FREE,PRO,PLUS
```

---

## Docker Compose 命令速查

### 基础操作

```powershell
# 启动服务（后台运行）
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看服务状态
docker-compose ps
```

### 日志查看

```powershell
# 实时查看日志
docker-compose logs -f

# 查看最近 100 行日志
docker-compose logs --tail=100

# 查看特定服务日志
docker-compose logs -f reset-scheduler
```

### 测试和调试

```powershell
# 运行测试模式
docker-compose run --rm reset-test

# 查看账号列表
docker-compose run --rm reset-scheduler -mode=list

# 进入容器调试
docker-compose exec reset-scheduler sh
```

### 更新和重建

```powershell
# 拉取最新代码后重建
docker-compose build --no-cache

# 重建并启动
docker-compose up -d --build

# 清理旧容器和镜像
docker-compose down --rmi all
```

---

## 数据持久化

项目会在本地创建以下目录：

```
d:\88code_reset\
├── data\              # 数据目录
│   ├── account.json   # 账号信息
│   ├── status.json    # 执行状态
│   └── accounts\      # 多账号数据
└── logs\              # 日志目录
    └── app.log        # 应用日志
```

**Windows 路径说明：**
- Docker Compose 会自动将 `./data` 映射到 `d:\88code_reset\data`
- 日志和数据会持久化保存，即使容器重启也不会丢失

---

## 常见问题

### Q1: 如何验证多个 API Key 都生效了？

**方法 1：查看测试输出**
```powershell
docker-compose run --rm reset-test
```
会显示所有账号的订阅信息。

**方法 2：查看账号列表**
```powershell
docker-compose run --rm reset-scheduler -mode=list
```
会列出所有已同步的账号。

**方法 3：查看日志**
```powershell
docker-compose logs | findstr "活跃账号"
```
会显示当前活跃的账号数量。

### Q2: 如何添加或删除 API Key？

1. 编辑 `.env` 文件，修改 `API_KEYS`
2. 重启服务：
   ```powershell
   docker-compose restart
   ```

### Q3: 如何查看某个账号的重置记录？

查看日志文件：
```powershell
# 查看所有重置记录
type logs\app.log | findstr "重置"

# 查看今天的日志
docker-compose logs --since=24h
```

### Q4: Docker Desktop 启动失败怎么办？

1. 确保 Windows 10 版本 >= 1903
2. 启用 WSL 2（推荐）或 Hyper-V
3. 在 BIOS 中启用虚拟化（VT-x/AMD-V）
4. 重启电脑后再试

### Q5: 容器无法访问网络？

检查 Docker Desktop 网络设置：
1. 打开 Docker Desktop
2. Settings → Resources → Network
3. 确保网络配置正确

### Q6: 如何修改重置时间？

重置时间是硬编码的（18:50 和 23:55），如需修改需要：
1. 修改源代码 `internal/scheduler/scheduler.go`
2. 重新构建镜像

### Q7: Windows 防火墙阻止 Docker？

允许 Docker Desktop 通过防火墙：
1. 控制面板 → Windows Defender 防火墙
2. 允许应用通过防火墙
3. 勾选 Docker Desktop

---

## 监控和维护

### 定期检查

建议每周检查一次：

```powershell
# 1. 查看服务状态
docker-compose ps

# 2. 查看最近的重置记录
docker-compose logs --tail=50 | findstr "重置"

# 3. 查看账号状态
docker-compose run --rm reset-scheduler -mode=list

# 4. 检查磁盘空间
dir data
dir logs
```

### 日志清理

日志会自动轮转（最多保留 3 个文件，每个 10MB），但可以手动清理：

```powershell
# 清理旧日志
del logs\*.log.old

# 或者重启服务清空日志
docker-compose down
del logs\app.log
docker-compose up -d
```

---

## 完整示例

### 示例 1：3 个账号，保守策略

`.env` 配置：
```env
API_KEYS=sk-key1,sk-key2,sk-key3
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MAX=83
ENABLE_FIRST_RESET=false
PLANS=
```

启动：
```powershell
docker-compose up -d
docker-compose logs -f
```

### 示例 2：5 个账号，激进策略

`.env` 配置：
```env
API_KEYS=sk-key1,sk-key2,sk-key3,sk-key4,sk-key5
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MIN=50
ENABLE_FIRST_RESET=true
PLANS=FREE,PRO
```

启动：
```powershell
docker-compose up -d
docker-compose logs -f
```

---

## 故障排查

### 检查清单

1. ✅ Docker Desktop 正在运行
2. ✅ `.env` 文件存在且配置正确
3. ✅ API Keys 格式正确（sk-开头）
4. ✅ 网络连接正常
5. ✅ 防火墙允许 Docker

### 获取详细日志

```powershell
# 查看容器详细信息
docker-compose ps -a

# 查看容器日志（包括错误）
docker-compose logs --tail=200

# 查看 Docker 系统日志
docker system events
```

### 完全重置

如果遇到无法解决的问题：

```powershell
# 1. 停止并删除所有容器
docker-compose down -v

# 2. 删除镜像
docker-compose down --rmi all

# 3. 清理数据（可选，会丢失历史记录）
# rmdir /s data
# rmdir /s logs

# 4. 重新构建和启动
docker-compose build --no-cache
docker-compose up -d
```

---

## 技术支持

- **项目地址**：https://github.com/Vulpecula-Studio/88code_reset
- **问题反馈**：提交 GitHub Issue
- **查看文档**：README.md

---

## 附录：Windows 特定注意事项

### 路径格式
- Windows 使用反斜杠 `\`，但 Docker 使用正斜杠 `/`
- Docker Compose 会自动转换路径

### 文件权限
- Windows 不需要设置 Linux 文件权限
- Docker Desktop 会自动处理

### 换行符
- 如果手动编辑 `.env`，使用 LF（Unix）而非 CRLF（Windows）
- 推荐使用 VS Code 或 Notepad++ 编辑

### PowerShell vs CMD
- 推荐使用 PowerShell（功能更强大）
- 所有命令在 CMD 中也可以运行

---

**祝你使用愉快！🎉**