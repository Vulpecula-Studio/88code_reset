## Highlights
- 调度器只在每个 5 分钟整点检查月付订阅，避免无谓的接口调用
- 多账号与单账号日志展示精简为三位小数，并汇总检查区间
- 新增多套餐重置失败自动重试机制，确保积分回升且 `resetTimes` 减少

## Technical
- 新增 `internal/reset` 模块集中处理 API 订阅筛选与重置流程
- `loopController` 对齐到固定 5 分钟时间槽
- `logAggregator` 按区间摘要输出且保留额外备注
