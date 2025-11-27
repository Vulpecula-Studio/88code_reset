## Highlights
- 23:55 第二次重置不再受额度阈值限制，只要 `resetTimes >= 1` 就会执行
- `.env.example` 的 `PLANS` 改为默认留空，未配置时会自动处理所有月付套餐

## Technical
- 更新 `reset.shouldSkipByThreshold`，在第二次重置时直接跳过额度判断，并补充单元测试覆盖上限模式分支
- 新增 `internal/reset/reset_test.go`，验证首次/二次重置对额度阈值的不同处理逻辑
