package reset

import (
	"code88reset/pkg/logger"
)

// LogResults prints summary for a reset run.
func LogResults(results []Result) {
	if len(results) == 0 {
		logger.Info("没有匹配的订阅需要处理")
		return
	}

	for _, res := range results {
		sub := res.Subscription
		if res.Err != nil {
			logger.Error("重置失败: %s (ID=%d, 尝试=%d) - %v", sub.SubscriptionName, sub.ID, res.Attempts, res.Err)
			continue
		}
		if res.Skipped {
			logger.Info("跳过 %s (ID=%d): %s", sub.SubscriptionName, sub.ID, res.SkipReason)
			continue
		}
		logger.Info("✅ %s (ID=%d, 尝试=%d) 重置成功: %s", sub.SubscriptionName, sub.ID, res.Attempts, res.ResetResponse.Message)
		logger.Info("   resetTimes %d -> %d, Credits %.4f -> %.4f",
			res.BeforeResets, res.AfterResets, res.BeforeCredits, res.AfterCredits)
	}
}
