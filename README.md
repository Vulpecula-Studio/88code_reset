# 88code Reset

极简命令行工具，用于定时重置 88code.org 上所有按月计费（`planType=MONTHLY`）的订阅额度，并自动跳过 PAYGO（`PAY_PER_USE`）套餐。

## 立即上手
1. **获取代码或二进制**
   ```bash
   git clone https://github.com/Vulpecula-Studio/88code_reset.git
   cd 88code_reset
   ```
   或在 [Releases](https://github.com/Vulpecula-Studio/88code_reset/releases) 下载对应平台的可执行文件。
2. **配置凭证与可选套餐**
   ```bash
   cat > .env <<'EOF'
   API_KEY=你的88codeAPIKey
   # TARGET_PLANS=FREE,PLUS    # 留空=所有 MONTHLY 套餐
   # TZ=Asia/Shanghai          # 影响定时任务
   EOF
   ```
3. **试运行**
   ```bash
   go run cmd/reset/main.go -mode=test
   # 或使用二进制/容器:
   ./88code_reset -mode=test
   docker run --rm -e API_KEY=xxx ghcr.io/vulpecula-studio/88code_reset:latest -mode=test
   ```
4. **正式运行**
   ```bash
   go run cmd/reset/main.go -mode=run
   ```

### 一键部署选项
- **Docker**：执行下面的一行命令立即启动（可按需挂载数据和日志）：
  ```bash
  docker run -d --name 88code-reset \
    -e API_KEY=你的88codeAPIKey \
    -e TARGET_PLANS=FREE,PLUS \
    ghcr.io/vulpecula-studio/88code_reset:latest -mode=run
  ```
- **Zeabur**：在 Zeabur 控制台创建服务，指向本仓库或官方镜像，最少配置 `API_KEY` 环境变量即可；可选设置 `TARGET_PLANS` 与 `TZ`。

## 核心行为
- 默认匹配所有 `planType=MONTHLY` 的订阅；通过 `TARGET_PLANS` 或 `-plans` 限定 `subscriptionName`。
- 每次触发会对所有符合条件的订阅逐个调用重置，无需指定单独 ID。
- 当 `resetTimes >= 2` 时，计划会在本地时间 18:50 与 23:55 自动重置，并记录数据到 `data/`。
- PAYGO 套餐始终被忽略，避免误重置。

## 关键配置

| 变量/参数      | 默认值 | 说明 |
|----------------|--------|------|
| `API_KEY`      | -      | **必填**。单个或逗号分隔多个 Key。|
| `TARGET_PLANS` | `""`   | 限定要处理的套餐名称（匹配 `subscriptionName`）。留空=全部 MONTHLY。|
| `TZ` / `-timezone` | `Asia/Shanghai` | 用于换算调度时间。|
| `-plans`       | `""`   | 与 `TARGET_PLANS` 等效。|
| `-mode`        | `test` | `test`、`run`、`list`。|
| `-threshold-max` | `83` | 大于该比率时跳过 18:50 重置。|
| `-threshold-min` | `0`  | 仅在余额低于该比率时执行 18:50 重置。|
| `-first-reset` | `false` | `true` 时启用 18:50 首次重置。|

## 常用模式

```bash
# 列出历史账号
go run cmd/reset/main.go -mode=list

# Docker 持续运行（挂载数据日志目录）
docker run -d --name 88code-reset \
  -e API_KEY=xxx \
  -e TARGET_PLANS=FREE,PLUS \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  ghcr.io/vulpecula-studio/88code_reset:latest -mode=run
```

## 最佳实践
- 「多个账号」：通过 `-apikeys` 或 `API_KEYS` 传入，工具会自动去重并并发调度。
- 「配置托管」：Zeabur/Docker 部署时只需设置 `API_KEY` (+ `TARGET_PLANS` 可选)。
- 「排障」：查看 `logs/`，或启用 `-mode=test` 观察订阅详情。
