package scheduler

import (
	"fmt"
	"time"

	"code88reset/internal/api"
	"code88reset/internal/config"
	"code88reset/internal/models"
	"code88reset/internal/reset"
	"code88reset/internal/storage"
	"code88reset/pkg/logger"
)

const (
	// 最小间隔时间（5小时）
	MinResetInterval = 5 * time.Hour

	// 订阅状态检查间隔（每 5 分钟检查一次）
	SubscriptionCheckInterval = 5 * time.Minute
)

// Scheduler 调度器
type Scheduler struct {
	*BaseScheduler
	apiClient *api.Client
}

// NewScheduler 创建新的调度器
func NewScheduler(apiClient *api.Client, storage *storage.Storage, timezone string) (*Scheduler, error) {
	return NewSchedulerWithConfig(apiClient, storage, timezone, 83.0, 0, true, false)
}

// NewSchedulerWithConfig 创建带配置的调度器
func NewSchedulerWithConfig(apiClient *api.Client, storage *storage.Storage, timezone string, thresholdMax, thresholdMin float64, useMax bool, enableFirstReset bool) (*Scheduler, error) {
	// 使用配置的时区，如果未设置则使用默认时区
	if timezone == "" {
		timezone = config.BeijingTimezone
	}

	base, err := newBaseScheduler(storage, timezone, thresholdMax, thresholdMin, useMax, enableFirstReset, "单账号调度器")
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		BaseScheduler: base,
		apiClient:     apiClient,
	}, nil
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.BaseScheduler.Start(func() {
		logger.Info("订阅状态检查间隔: %v", SubscriptionCheckInterval)
	})
	s.loop.run(s.checkSubscriptionStatus, s.checkAndExecute)
	s.logAgg.Flush()
	logger.Info("调度器已停止")
}

// checkSubscriptionStatus 检查并验证目标订阅状态
func (s *Scheduler) checkSubscriptionStatus() {
	logger.Debug("检查目标订阅状态...")

	runner := reset.NewRunner(
		s.apiClient,
		reset.Filter{TargetPlans: s.apiClient.TargetPlans, RequireMonthly: true},
		reset.Options{},
	)

	subs, err := runner.Eligible()
	if err != nil {
		logger.Warn("无法获取目标订阅: %v", err)
		return
	}

	if len(subs) == 0 {
		logger.Warn("未找到符合条件的订阅")
		return
	}

	logger.Info("订阅状态（共 %d 个）:", len(subs))
	for i := range subs {
		sub := &subs[i]
		s.updateAccountInfo(sub)
		logger.Info("  [%d] 名称=%s, 类型=%s, resetTimes=%d, 积分=%.3f/%.3f",
			i+1,
			sub.SubscriptionName,
			sub.SubscriptionPlan.PlanType,
			sub.ResetTimes,
			sub.CurrentCredits,
			sub.SubscriptionPlan.CreditLimit)

		if sub.ResetTimes < 2 {
			logger.Warn("    resetTimes=%d，不足以执行重置（需要 >= 2）", sub.ResetTimes)
		}
	}
}

// checkAndExecute 检查并执行重置任务
func (s *Scheduler) checkAndExecute() {
	shouldReset, resetType := s.checkTime()
	if shouldReset {
		s.executeReset(resetType)
	}
}

// executeReset 执行重置逻辑
func (s *Scheduler) executeReset(resetType string) {
	s.logAgg.Flush()
	logger.Info("========================================")
	logger.Info("触发%s重置任务", map[string]string{"first": "第一次", "second": "第二次"}[resetType])
	logger.Info("========================================")

	// 尝试获取锁
	operation := fmt.Sprintf("%s_reset", resetType)
	if err := s.storage.AcquireLock(operation); err != nil {
		logger.Warn("无法获取锁: %v", err)
		return
	}
	defer s.storage.ReleaseLock()

	// 加载状态
	status, err := s.storage.LoadStatus()
	if err != nil {
		logger.Error("加载状态失败: %v", err)
		return
	}

	// 检查今天是否已经执行过此次重置
	if resetType == "first" && status.FirstResetToday {
		logger.Info("今天已执行过第一次重置，跳过")
		return
	}
	if resetType == "second" && status.SecondResetToday {
		logger.Info("今天已执行过第二次重置，跳过")
		return
	}

	// 检查两次重置的时间间隔
	if resetType == "second" && status.LastFirstResetTime != nil {
		interval := time.Since(*status.LastFirstResetTime)
		if interval < MinResetInterval {
			logger.Warn("距离第一次重置时间不足5小时（%.1f小时），跳过", interval.Hours())
			return
		}
	}

	logger.Info("正在获取目标订阅信息...")
	runner := reset.NewRunner(
		s.apiClient,
		reset.Filter{TargetPlans: s.apiClient.TargetPlans, RequireMonthly: true},
		reset.Options{
			ResetType:          resetType,
			UseMaxThreshold:    s.useMaxThreshold,
			CreditThresholdMax: s.creditThresholdMax,
			CreditThresholdMin: s.creditThresholdMin,
		},
	)

	results, err := runner.Execute()
	if err != nil {
		logger.Error("执行重置失败: %v", err)
		s.recordFailure(status, err.Error(), resetType)
		return
	}

	if len(results) == 0 {
		logger.Warn("未找到需要处理的订阅")
		s.recordSkip(status, resetType, "无匹配订阅")
		return
	}

	reset.LogResults(results)

	anySuccess := false
	anyError := false
	lastMessage := ""

	for _, res := range results {
		if res.Err != nil {
			anyError = true
			lastMessage = fmt.Sprintf("[%s] %v", res.Subscription.SubscriptionName, res.Err)
			continue
		}
		if res.Skipped {
			lastMessage = fmt.Sprintf("[%s] 跳过: %s", res.Subscription.SubscriptionName, res.SkipReason)
			continue
		}

		anySuccess = true
		lastMessage = fmt.Sprintf("[%s] %s", res.Subscription.SubscriptionName, res.ResetResponse.Message)
		status.ResetTimesBeforeReset = res.BeforeResets
		status.CreditsBeforeReset = res.BeforeCredits
		status.ResetTimesAfterReset = res.AfterResets
		status.CreditsAfterReset = res.AfterCredits

		if res.UpdatedSubscription != nil {
			s.updateAccountInfo(res.UpdatedSubscription)
		} else {
			s.updateAccountInfo(&res.Subscription)
		}
	}

	now := time.Now()
	if resetType == "first" {
		status.FirstResetToday = true
		status.LastFirstResetTime = &now
	} else {
		status.SecondResetToday = true
		status.LastSecondResetTime = &now
	}

	status.LastResetMessage = lastMessage

	if anySuccess {
		status.LastResetSuccess = true
		status.ConsecutiveFailures = 0
	} else if anyError {
		status.LastResetSuccess = false
		status.ConsecutiveFailures++
	} else {
		status.LastResetSuccess = true
	}

	if err := s.storage.SaveStatus(status); err != nil {
		logger.Error("保存状态失败: %v", err)
	}

	logger.Info("========================================")
	logger.Info("%s重置任务完成", map[string]string{"first": "第一次", "second": "第二次"}[resetType])
	logger.Info("========================================")
}

// updateAccountInfo 更新账号信息
func (s *Scheduler) updateAccountInfo(sub *models.Subscription) {
	s.accountUpdater.UpdateGlobal(sub)
}
