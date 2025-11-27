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