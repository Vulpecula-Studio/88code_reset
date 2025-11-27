# Windows 10 快速启动指南

## 🚀 3 分钟快速部署

### 步骤 1：复制配置模板

复制以下内容，创建 `d:\88code_reset\.env` 文件：

```env
# ============================================
# 88code Reset 多账号配置模板
# 适用于 Windows Docker Compose 部署
# ============================================

# 📌 多账号配置（必须配置）
# 替换为你的实际 API Keys，用英文逗号分隔
API_KEYS=sk-your-first-key-here,sk-your-second-key-here,sk-your-third-key-here

# 🌍 时区配置
TIMEZONE=Asia/Shanghai

# 💰 额度判断配置
# 上限模式：当额度 > 83% 时跳过18:50重置（保留到23:55）
CREDIT_THRESHOLD_MAX=83

# 下限模式：当额度 < 50% 时才执行18:50重置（取消注释启用）
# CREDIT_THRESHOLD_MIN=50

# ⏰ 重置时间配置
# 是否启用18:50重置（默认关闭，只在23:55重置）
ENABLE_FIRST_RESET=false

# 🎯 目标套餐配置（可选）
# 留空 = 所有 MONTHLY 套餐
# 指定套餐：FREE,PRO,PLUS（逗号分隔）
PLANS=
```

### 步骤 2：复制启动脚本

创建 `d:\88code_reset\start.bat` 文件：

```batch
@echo off
echo ========================================
echo 88code Reset 多账号快速启动脚本
echo ========================================
echo.

REM 检查 Docker Desktop 是否运行
docker --version >nul 2>&1
if errorlevel 1 (
    echo ❌ 错误: Docker Desktop 未安装或未运行
    echo 请先安装并启动 Docker Desktop for Windows
    pause
    exit /b 1
)

echo ✅ Docker Desktop 已就绪
echo.

REM 检查 .env 文件是否存在
if not exist ".env" (
    echo ❌ 错误: .env 配置文件不存在
    echo 请先创建 .env 文件并配置 API Keys
    echo.
    echo 参考配置:
    echo API_KEYS=sk-key1,sk-key2,sk-key3
    echo TIMEZONE=Asia/Shanghai
    pause
    exit /b 1
)

echo ✅ 配置文件已找到
echo.

REM 显示当前配置
echo 📋 当前配置:
type .env | findstr /V "^#" | findstr /V "^$"
echo.

echo ========================================
echo 开始部署...
echo ========================================
echo.

REM 构建并启动服务
echo 🏗️  构建 Docker 镜像...
docker-compose build

echo 🚀 启动服务...
docker-compose up -d

echo.
echo ========================================
echo 部署完成!
echo ========================================
echo.

echo 📊 查看服务状态:
docker-compose ps

echo.
echo 📋 查看实时日志:
echo docker-compose logs -f
echo.

echo 🧪 运行测试模式:
echo docker-compose run --rm reset-test
echo.

echo 📱 查看账号列表:
echo docker-compose run --rm reset-scheduler -mode=list
echo.

echo ⏹️  停止服务:
echo docker-compose down
echo.

echo ✅ 服务已启动!
echo 提示: 使用 Ctrl+C 可停止查看日志
pause
```

### 步骤 3：一键启动

1. 双击 `start.bat` 文件
2. 等待部署完成
3. 查看输出确认服务启动成功

## 📋 常用命令

### 在 PowerShell 或 CMD 中使用

```powershell
# 🧪 测试配置
docker-compose run --rm reset-test

# 📱 查看账号列表  
docker-compose run --rm reset-scheduler -mode=list

# 📊 查看实时日志
docker-compose logs -f

# 📊 查看最近日志
docker-compose logs --tail=100

# 🔄 重启服务
docker-compose restart

# ⏹️ 停止服务
docker-compose down

# 🏗️  重新构建并启动
docker-compose up -d --build
```

## 🎯 配置示例

### 保守策略（推荐新手）
```env
API_KEYS=sk-key1,sk-key2,sk-key3
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MAX=83
ENABLE_FIRST_RESET=false
PLANS=
```

### 激进策略
```env
API_KEYS=sk-key1,sk-key2,sk-key3
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MIN=50
ENABLE_FIRST_RESET=true
PLANS=
```

### 精准控制（仅特定套餐）
```env
API_KEYS=sk-key1,sk-key2,sk-key3
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MAX=75
ENABLE_FIRST_RESET=true
PLANS=FREE,PRO
```

## 🔍 验证部署成功

### 1. 检查服务状态
```powershell
docker-compose ps
```
应该看到 `reset-scheduler` 服务状态为 `Up`

### 2. 查看启动日志
```powershell
docker-compose logs --tail=20
```
应该看到：
- `调度器启动`
- `活跃账号: X 个`
- `多账号调度器已启动`

### 3. 验证 API Keys
```powershell
docker-compose run --rm reset-scheduler -mode=list
```
应该显示所有配置的账号信息

## 🚨 故障排查

### 问题 1：Docker Desktop 未运行
**解决方案：**
1. 启动 Docker Desktop
2. 等待图标变为绿色
3. 重新运行脚本

### 问题 2：API Keys 格式错误
**正确格式：**
```env
API_KEYS=sk-xxxxxxxxxxx,sk-yyyyyyyyyy,sk-zzzzzzzzz
```

**错误格式：**
```env
# ❌ 缺少 sk- 前缀
API_KEYS=xxxxxxxxx,yyyyyyyyy

# ❌ 有空格
API_KEYS=sk-key1 , sk-key2

# ❌ 使用了引号
API_KEYS="sk-key1,sk-key2"
```

### 问题 3：容器启动失败
```powershell
# 查看详细错误
docker-compose logs

# 重建容器
docker-compose down
docker-compose up -d --build
```

### 问题 4：网络连接问题
```powershell
# 测试网络连接
docker-compose run --rm reset-test

# 如果连接失败，检查：
# 1. 网络是否正常
# 2. 防火墙设置
# 3. Docker Desktop 网络配置
```

## 📁 目录结构

部署后的目录结构：
```
d:\88code_reset\
├── .env                    # 你的配置文件
├── docker-compose.yml      # Docker 编排文件
├── Dockerfile             # Docker 镜像定义
├── start.bat              # 启动脚本
├── data\                  # 数据目录（自动创建）
│   ├── account.json       # 账号信息
│   ├── status.json        # 执行状态
│   └── accounts\          # 多账号数据
└── logs\                  # 日志目录（自动创建）
    └── app.log            # 应用日志
```

## 📈 监控建议

### 每日检查
```powershell
# 查看今天是否有重置
docker-compose logs --since=24h | findstr "重置"

# 查看账号状态
docker-compose run --rm reset-scheduler -mode=list
```

### 每周维护
```powershell
# 重启服务（清理内存）
docker-compose restart

# 检查磁盘空间
dir data
dir logs

# 清理旧日志
del logs\*.log.old
```

## 🎉 成功标志

当看到以下输出时，说明部署成功：

```
========================================
多账号调度器已启动
将为 X 个账号执行定时重置
按 Ctrl+C 停止
========================================
```

以及：

```
✅ 获取到 X 个订阅
✅ 找到 X 个目标订阅
```

## 📞 获取帮助

如果遇到问题：
1. 查看 [详细部署指南](WINDOWS_DOCKER_SETUP.md)
2. 检查项目 GitHub Issues
3. 查看日志文件获取错误信息

---

**🎯 恭喜！你的多账号 88code Reset 服务已成功部署！**