# Docker 代理配置指南（Clash 端口 7897）

## 🎯 问题已确认

你安装了 **clash** 代理软件，端口为 **7897**，这导致 Docker 无法直接连接到 Docker Hub。我们需要配置 Docker 使用代理。

## 🔧 配置 Docker Desktop 代理

### 方案 1：在 Docker Desktop 中配置代理（推荐）

#### 1.1 打开 Docker Desktop 设置
1. 右键系统托盘的 Docker 图标
2. 选择 "Settings"
3. 进入 "Resources" → "Proxies"

#### 1.2 配置代理设置
勾选 "Manual proxy configuration" 并填入：
- **HTTP proxy**: `http://localhost:7897`
- **HTTPS proxy**: `http://localhost:7897`
- **Bypass**: `localhost,127.*,10.*,192.168.*,*.local`

#### 1.3 应用并重启
1. 点击 "Apply & Restart"
2. 等待 Docker Desktop 重启完成

### 方案 2：使用命令行配置代理

```powershell
# 设置系统环境变量
[System.Environment]::SetEnvironmentVariable("HTTP_PROXY", "http://localhost:7897", "User")
[System.Environment]::SetEnvironmentVariable("HTTPS_PROXY", "http://localhost:7897", "User")
[System.Environment]::SetEnvironmentVariable("NO_PROXY", "localhost,127.0.0.1,10.0.0.0/8,192.168.0.0/16", "User")

# 当前会话立即生效
$env:HTTP_PROXY = "http://localhost:7897"
$env:HTTPS_PROXY = "http://localhost:7897"
$env:NO_PROXY = "localhost,127.0.0.1,10.0.0.0/8,192.168.0.0/16"
```

### 方案 3：在 Docker Compose 中配置代理

如果方案 1 和 2 不行，修改 docker-compose.yml：

```yaml
services:
  reset-scheduler:
    build: 
      context: .
      args:
        HTTP_PROXY: http://host.docker.internal:7897
        HTTPS_PROXY: http://host.docker.internal:7897
        NO_PROXY: localhost,127.0.0.1,10.0.0.0/8,192.168.0.0/16
    # ... 其他配置
```

## 🚀 测试代理配置

配置完成后，运行测试命令：

```powershell
# 先清理之前的构建缓存
docker-compose down --rmi all

# 重新构建（应该通过代理下载镜像）
docker-compose build

# 测试运行
docker-compose run --rm reset-test
```

## 🔍 验证代理是否生效

### 查看构建日志
如果代理配置成功，你应该看到：
```
[+] Building 30.2s (4/4) FINISHED
 => [internal] load metadata for docker.io/library/golang:1.21-alpine
 => [internal] load metadata for docker.io/library/alpine:latest
```

### 检查 Docker 连接
```powershell
# 测试 Docker Hub 连接
docker run --rm alpine ping -c 1 google.com

# 查看容器网络
docker network ls
```

## 🆘 故障排查

### 问题 1：代理连接失败
**症状**: 
```
dial tcp 127.0.0.1:7897: connect: connection refused
```

**解决方案**:
1. 确保 clash 正在运行
2. 检查 clash 端口是否确实是 7897
3. 确保 clash 允许局域网连接

### 问题 2：Docker 无法解析域名
**症状**:
```
lookup registry-1.docker.io: no such host
```

**解决方案**:
1. 检查 clash 的 DNS 设置
2. 尝试在 clash 中开启 DNS 服务器
3. 重启 clash 和 Docker Desktop

### 问题 3：构建仍然超时
**解决方案**:
1. 重启 clash
2. 重启 Docker Desktop
3. 检查 clash 是否开启了系统代理

## 📋 完整配置步骤

### 步骤 1: 检查 clash 状态
```powershell
# 检查 clash 端口是否可用
Test-NetConnection -ComputerName localhost -Port 7897

# 应该显示：TcpTestSucceeded : True
```

### 步骤 2: 配置 Docker Desktop
1. 打开 Docker Desktop
2. Settings → Resources → Proxies
3. 勾选 Manual proxy configuration
4. 填入：`http://localhost:7897`
5. Apply & Restart

### 步骤 3: 验证配置
```powershell
# 清理并重新构建
docker-compose down --rmi all
docker-compose build

# 测试
docker-compose run --rm reset-test
```

### 步骤 4: 启动服务
```powershell
# 如果测试成功，启动正式服务
docker-compose up -d
docker-compose logs -f
```

## 💡 额外提示

### Clash 配置建议
在 clash 设置中：
1. 开启 "Allow LAN"（允许局域网连接）
2. 设置系统代理模式为 "Global" 或 "Rule"
3. 确保 DNS 设置正确

### Docker Desktop 建议
1. 使用最新版本的 Docker Desktop
2. 定期清理 Docker 缓存
3. 监控网络流量

## 🎉 成功标志

当配置成功后，你应该看到：
1. ✅ Docker 镜像下载成功
2. ✅ 容器构建完成
3. ✅ 测试模式显示你的 10 个账号信息
4. ✅ 服务可以正常启动

---

**🚀 现在就去配置 Docker Desktop 的代理设置吧！**