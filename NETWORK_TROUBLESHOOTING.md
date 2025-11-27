# Docker 网络连接问题排查和解决方案

## 🚨 当前问题

你遇到的是 Docker 无法连接到 Docker Hub 下载镜像的问题：
```
failed to solve: DeadlineExceeded: failed to fetch anonymous token: Get "https://auth.docker.io/token?scope=repository%3Alibrary%2Falpine%3Apull&service=registry.docker.io": dial tcp 54.164.151.35:443: i/o timeout
```

## 🔧 解决方案

### 方案 1：配置 Docker 镜像加速器（推荐）

#### 1.1 打开 Docker Desktop 设置
1. 右键系统托盘的 Docker 图标
2. 选择 "Settings"
3. 进入 "Docker Engine" 设置

#### 1.2 配置镜像加速器
在 "Docker Engine" 配置中添加以下内容：

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://ccr.ccs.tencentyun.com"
  ],
  "experimental": false,
  "features": {
    "buildkit": true
  }
}
```

#### 1.3 应用并重启
1. 点击 "Apply & Restart"
2. 等待 Docker Desktop 重启完成

### 方案 2：使用国内镜像源

如果方案 1 不行，可以手动指定镜像源：

```powershell
# 临时设置镜像源
$env:DOCKER_REGISTRY_MIRROR="https://docker.mirrors.ustc.edu.cn"

# 或者设置环境变量
[System.Environment]::SetEnvironmentVariable("DOCKER_REGISTRY_MIRROR", "https://docker.mirrors.ustc.edu.cn", "User")
```

### 方案 3：检查网络连接

#### 3.1 测试 Docker Hub 连接
```powershell
# 测试 Docker Hub 连接
Test-NetConnection -ComputerName registry-1.docker.io -Port 443

# 测试 DNS 解析
nslookup registry-1.docker.io
```

#### 3.2 检查防火墙设置
1. 打开 Windows Defender 防火墙
2. 允许 Docker Desktop 通过防火墙
3. 检查是否有代理软件拦截

### 方案 4：重置 Docker 网络

```powershell
# 停止所有容器
docker-compose down

# 清理 Docker 网络
docker network prune -f

# 重启 Docker Desktop
# 手动重启 Docker Desktop
```

### 方案 5：使用代理（如果在公司网络）

如果在公司网络环境，可能需要配置代理：

```powershell
# 设置 Docker 代理
docker run --rm -it --privileged \
  -e HTTP_PROXY=http://your-proxy:port \
  -e HTTPS_PROXY=http://your-proxy:port \
  alpine sh
```

## 🚀 重新尝试部署

解决网络问题后，重新运行：

```powershell
# 清理之前的构建缓存
docker-compose down --rmi all

# 重新构建并测试
docker-compose run --rm reset-test
```

## 📋 完整排查步骤

### 步骤 1：检查 Docker Desktop
```powershell
# 检查 Docker 是否运行
docker --version
docker info
```

### 步骤 2：测试网络连接
```powershell
# 测试基本网络
ping 8.8.8.8

# 测试 Docker Hub
Test-NetConnection -ComputerName registry-1.docker.io -Port 443
```

### 步骤 3：配置镜像加速器
按照上述方案 1 配置镜像加速器

### 步骤 4：清理并重试
```powershell
# 清理所有容器和镜像
docker-compose down --rmi all --volumes

# 重新构建
docker-compose build --no-cache

# 测试
docker-compose run --rm reset-test
```

## 🔍 验证修复成功

修复成功后，你应该看到：
```
[+] Building 10.2s (4/4) FINISHED
 => [internal] load build definition from Dockerfile
 => [internal] load metadata for docker.io/library/golang:1.21-alpine
 => [internal] load metadata for docker.io/library/alpine:latest
 => [1/4] FROM docker.io/library/golang:1.21-alpine
 => [2/4] WORKDIR /app
 => [3/4] COPY . .
 => [4/4] RUN go build -o reset cmd/reset/main.go
```

然后测试模式应该正常运行并显示你的账号信息。

## 🆘 如果仍然失败

如果以上方案都不行，可以尝试：

### 1. 使用预构建镜像
```powershell
# 直接使用官方镜像（如果有）
docker run --rm -v "$(pwd)/.env:/app/.env" -v "$(pwd)/data:/app/data" -v "$(pwd)/logs:/app/logs" ghcr.io/vulpecula-studio/88code_reset:latest -mode=test
```

### 2. 切换网络环境
- 尝试使用手机热点
- 尝试不同的 WiFi 网络
- 检查是否有 VPN 影响

### 3. 联系网络管理员
如果在公司网络，联系 IT 部门：
- 询问是否需要配置代理
- 询问是否限制了 Docker Hub 访问
- 申请开放相关网络权限

## 📞 获取帮助

如果问题仍然存在：
1. 查看 Docker Desktop 日志
2. 检查 Windows 事件查看器
3. 在 GitHub 提交 issue

---

**💡 提示：网络问题是 Docker 常见问题，通常配置镜像加速器可以解决。**