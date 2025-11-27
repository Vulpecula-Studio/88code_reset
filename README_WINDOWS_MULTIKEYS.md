# Windows 10 多 Key Docker Compose 部署指南

## 📁 已创建的文件

我已经为你创建了以下文件来简化部署过程：

### 🎯 快速开始文件
- **[`QUICK_START_WINDOWS.md`](QUICK_START_WINDOWS.md)** - 3分钟快速部署指南
- **[`start.bat`](start.bat)** - 一键启动脚本
- **[`.env.multi-keys-template`](.env.multi-keys-template)** - 多 Key 配置模板

### 📚 详细文档
- **[`WINDOWS_DOCKER_SETUP.md`](WINDOWS_DOCKER_SETUP.md)** - 完整部署文档和故障排查

## 🚀 立即开始（3 步搞定）

### 步骤 1：准备你的 API Keys

准备好你的所有 88code API Keys，格式类似：
```
sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
sk-yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy
sk-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz
```

### 步骤 2：创建配置文件

```powershell
# 复制配置模板
copy .env.multi-keys-template .env

# 编辑配置文件
notepad .env
```

在 `.env` 文件中，找到这行：
```env
API_KEYS=sk-your-first-key-here,sk-your-second-key-here,sk-your-third-key-here
```

替换为你的实际 API Keys：
```env
API_KEYS=sk-你的第一个key,sk-你的第二个key,sk-你的第三个key
```

### 步骤 3：一键启动

双击 `start.bat` 文件，或运行：
```powershell
.\start.bat
```

## ✅ 验证部署成功

部署完成后，运行以下命令验证：

```powershell
# 查看服务状态
docker-compose ps

# 查看账号列表
docker-compose run --rm reset-scheduler -mode=list

# 查看实时日志
docker-compose logs -f
```

## 🎯 推荐配置

### 保守策略（推荐新手）
```env
API_KEYS=你的keys
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MAX=83
ENABLE_FIRST_RESET=false
PLANS=
```

### 激进策略
```env
API_KEYS=你的keys
TIMEZONE=Asia/Shanghai
CREDIT_THRESHOLD_MIN=50
ENABLE_FIRST_RESET=true
PLANS=
```

## 📋 常用命令

```powershell
# 测试配置
docker-compose run --rm reset-test

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 重建并启动
docker-compose up -d --build
```

## 📁 最终目录结构

```
d:\88code_reset\
├── .env                           # 你的配置文件（需要创建）
├── .env.multi-keys-template       # 配置模板（已提供）
├── start.bat                      # 启动脚本（已提供）
├── docker-compose.yml             # Docker 编排文件
├── QUICK_START_WINDOWS.md         # 快速指南
├── WINDOWS_DOCKER_SETUP.md        # 详细文档
├── README_WINDOWS_MULTIKEYS.md    # 本文件
├── data\                          # 数据目录（自动创建）
└── logs\                          # 日志目录（自动创建）
```

## 🎉 成功标志

当你看到以下输出时，说明部署成功：

1. **启动脚本输出**：
   ```
   ✅ 服务已启动!
   ```

2. **服务状态**：
   ```
   reset-scheduler    Up
   ```

3. **日志内容**：
   ```
   多账号调度器已启动
   将为 X 个账号执行定时重置
   ```

## 🔧 如果遇到问题

### 1. Docker Desktop 未运行
- 启动 Docker Desktop
- 等待图标变绿
- 重新运行 `start.bat`

### 2. API Keys 格式错误
- 确保以 `sk-` 开头
- 用英文逗号分隔
- 不要有空格

### 3. 容器启动失败
```powershell
# 查看错误详情
docker-compose logs

# 重新构建
docker-compose down
docker-compose up -d --build
```

## 📞 需要帮助？

1. 查看 [详细部署文档](WINDOWS_DOCKER_SETUP.md)
2. 查看 [快速启动指南](QUICK_START_WINDOWS.md)
3. 检查配置模板 [`.env.multi-keys-template`](.env.multi-keys-template)

---

**🎯 恭喜！你的 Windows 10 多 Key 88code Reset 服务已准备就绪！**

现在你可以：
- 运行 `.\start.bat` 启动服务
- 使用 `docker-compose logs -f` 监控运行状态
- 享受自动化的订阅额度管理