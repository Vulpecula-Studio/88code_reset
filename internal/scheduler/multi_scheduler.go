package scheduler

import (
	"fmt"
	"sync"
	"time"

	"code88reset/internal/api"
	"code88reset/internal/config"
	"code88reset/internal/models"
	"code88reset/internal/reset"
	"code88reset/internal/storage"
	"code88reset/pkg/logger"
)

// MultiScheduler 多账号调度器
type MultiScheduler struct {
	*BaseScheduler
	activeAccounts []models.AccountConfig // 当前活跃的账号列表（从环境变量获取）
	baseURL        string
	targetPlans    []string
}

// NewMultiSchedulerWithAccounts 创建新的多账号调度器（使用指定的账号列表）
func NewMultiSchedulerWithAccounts(storage *storage.Storage, baseURL string, activeAccounts []models.AccountConfig, targetPlans []string, timezone string) (*MultiScheduler, error) {
	return NewMultiSchedulerWithConfig(storage, baseURL, activeAccounts, targetPlans, timezone, 83.0, 0, true, false)
}

// NewMultiSchedulerWithConfig 创建带配置的多账号调度器
func NewMultiSchedulerWithConfig(storage *storage.Storage, baseURL string, activeAccounts []models.AccountConfig, targetPlans []string, timezone string, thresholdMax, thresholdMin float64, useMax bool, enableFirstReset bool) (*MultiScheduler, error) {
	base, err := newBaseScheduler(storage, timezone, thresholdMax, thresholdMin, useMax, enableFirstReset, "多账号调度器")
	if err != nil {
		return nil, err
	}
	if timezone == "" {
		timezone = config.BeijingTimezone
	}

	return &MultiScheduler{
		BaseScheduler:  base,
		activeAccounts: activeAccounts,
		baseURL:        baseURL,
		targetPlans:    targetPlans,
	}, nil
}

// Start 启动多账号调度器
func (s *MultiScheduler) Start() {
	s.BaseScheduler.Start(func() {
		logger.Info("活跃账号数量: %d", len(s.activeAccounts))
	})

	if len(s.activeAccounts) == 0 {
		logger.Warn("没有活跃的账号，调度器将空转")
	}

	// 启动时立即检查所有账号的订阅状态
	s.loop.run(s.checkAllAccountsStatus, s.checkAndExecute)
	s.logAgg.Flush()
	logger.Info("多账号调度器已停止")
}

// checkAllAccountsStatus 检查所有活跃账号的订阅状态
func (s *MultiScheduler) checkAllAccountsStatus() {
	if len(s.activeAccounts) == 0 {
		logger.Debug("没有活跃的账号")
		return
	}

	logger.Info("开始检查 %d 个账号的订阅状态...", len(s.activeAccounts))

	for i, acc := range s.activeAccounts {
		logger.Info("[%d/%d] 检查账号: %s (%s)",
			i+1, len(s.activeAccounts), acc.EmployeeEmail, acc.Name)

		// 创建客户端
		client := api.NewClient(s.baseURL, acc.APIKey, s.targetPlans)
		client.Storage = s.storage

		runner := reset.NewRunner(
			client,
			reset.Filter{TargetPlans: s.targetPlans, RequireMonthly: true},
			reset.Options{},
		)
		subs, err := runner.Eligible()
		if err != nil {
			logger.Warn("账号 %s 无法获取目标订阅: %v", acc.EmployeeEmail, err)
			continue
		}

		if len(subs) == 0 {
			logger.Warn("  账号 %s 未找到符合条件的订阅", acc.EmployeeEmail)
			continue
		}

		for i := range subs {
			sub := &subs[i]
			s.updateAccountInfo(acc.EmployeeEmail, sub)
			logger.Info("  订阅[%d]: 名称=%s, 类型=%s, resetTimes=%d, 积分=%.3f/%.3f",
				i+1,
				sub.SubscriptionName,
				sub.SubscriptionPlan.PlanType,
				sub.ResetTimes,
				sub.CurrentCredits,
				sub.SubscriptionPlan.CreditLimit)

			if sub.ResetTimes < 2 {
				logger.Warn("    账号 %s 的 resetTimes=%d，不足以执行重置",
					acc.EmployeeEmail, sub.ResetTimes)
			}
		}
	}

	logger.Info("所有账号订阅状态检查完成")
}

// updateAccountInfo 更新账号信息
func (s *MultiScheduler) updateAccountInfo(employeeEmail string, sub *models.Subscription) {
	s.accountUpdater.UpdateByEmail(employeeEmail, sub)
}

// checkAndExecute 检查并执行重置任务
func (s *MultiScheduler) checkAndExecute() {
	shouldReset, resetType := s.checkTime()
	if shouldReset {
		s.executeResetForAllAccounts(resetType)
	}
}

// executeResetForAllAccounts 为所有活跃账号执行重置
func (s *MultiScheduler) executeResetForAllAccounts(resetType string) {
	s.logAgg.Flush()
	resetName := map[string]string{"first": "第一次", "second": "第二次"}[resetType]

	logger.Info("========================================")
	logger.Info("触发%s重置任务（多账号模式）", resetName)
	logger.Info("========================================")

	if len(s.activeAccounts) == 0 {
		logger.Warn("没有活跃的账号，跳过重置")
		return
	}

	logger.Info("开始为 %d 个账号执行%s重置...", len(s.activeAccounts), resetName)

	// 使用 WaitGroup 并发执行重置
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i, acc := range s.activeAccounts {
		wg.Add(1)
		go func(index int, account models.AccountConfig) {
			defer wg.Done()

			logger.Info("[%d/%d] 开始重置账号: %s (%s)",
				index+1, len(s.activeAccounts), account.EmployeeEmail, account.Name)

			success := s.executeResetForAccount(account, resetType)

			mu.Lock()
			if success {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i, acc)
	}

	// 等待所有重置完成
	wg.Wait()

	logger.Info("========================================")
	logger.Info("%s重置任务完成: 成功 %d 个，失败 %d 个",
		resetName, successCount, failCount)
	logger.Info("========================================")
}

// executeResetForAccount 为单个账号执行重置
func (s *MultiScheduler) executeResetForAccount(acc models.AccountConfig, resetType string) bool {
	employeeEmail := acc.EmployeeEmail

	// 加载账号的执行状态
	status, err := s.storage.LoadStatusByEmail(employeeEmail)
	if err != nil {
		logger.Error("账号 %s 加载状态失败: %v", employeeEmail, err)
		return false
	}

	// 检查今天是否已经执行过此次重置
	if resetType == "first" && status.FirstResetToday {
		logger.Info("账号 %s 今天已执行过第一次重置，跳过", employeeEmail)
		return true // 返回 true 因为已经完成
	}
	if resetType == "second" && status.SecondResetToday {
		logger.Info("账号 %s 今天已执行过第二次重置，跳过", employeeEmail)
		return true
	}

	// 检查时间间隔
	var lastResetTime *time.Time
	if resetType == "first" {
		lastResetTime = status.LastFirstResetTime
	} else {
		lastResetTime = status.LastSecondResetTime
	}

	if lastResetTime != nil && time.Since(*lastResetTime) < MinResetInterval {
		logger.Warn("账号 %s 距离上次重置时间不足 %v，跳过",
			employeeEmail, MinResetInterval)
		return false
	}

	// 创建客户端
	client := api.NewClient(s.baseURL, acc.APIKey, s.targetPlans)
	client.Storage = s.storage

	runner := reset.NewRunner(
		client,
		reset.Filter{TargetPlans: s.targetPlans, RequireMonthly: true},
		reset.Options{
			ResetType:          resetType,
			UseMaxThreshold:    s.useMaxThreshold,
			CreditThresholdMax: s.creditThresholdMax,
			CreditThresholdMin: s.creditThresholdMin,
		},
	)

	results, err := runner.Execute()
	if err != nil {
		logger.Error("账号 %s 执行重置失败: %v", employeeEmail, err)
		s.updateResetStatus(employeeEmail, status, resetType, false, err.Error())
		return false
	}

	if len(results) == 0 {
		logger.Warn("账号 %s 未找到符合条件的订阅", employeeEmail)
		s.updateResetStatus(employeeEmail, status, resetType, true, "无匹配订阅")
		return true
	}

	reset.LogResults(results)

	anySuccess := false
	anyError := false
	lastMessage := ""
	firstSuccessRecorded := false

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
		if !firstSuccessRecorded {
			status.ResetTimesBeforeReset = res.BeforeResets
			status.CreditsBeforeReset = res.BeforeCredits
			status.ResetTimesAfterReset = res.AfterResets
			status.CreditsAfterReset = res.AfterCredits
			firstSuccessRecorded = true
		}

		if res.UpdatedSubscription != nil {
			s.updateAccountInfo(employeeEmail, res.UpdatedSubscription)
		} else {
			s.updateAccountInfo(employeeEmail, &res.Subscription)
		}
	}

	successFlag := false
	if anySuccess {
		successFlag = true
	} else if !anyError {
		// 全部跳过也视为成功，避免统计为失败
		successFlag = true
	}

	s.updateResetStatus(employeeEmail, status, resetType, successFlag, lastMessage)

	return successFlag
}

// updateResetStatus 更新重置状态
func (s *MultiScheduler) updateResetStatus(employeeEmail string, status *models.ExecutionStatus, resetType string, success bool, message string) {
	now := time.Now()

	if resetType == "first" {
		status.FirstResetToday = true
		status.LastFirstResetTime = &now
	} else {
		status.SecondResetToday = true
		status.LastSecondResetTime = &now
	}

	status.LastResetSuccess = success
	status.LastResetMessage = message

	if success {
		status.ConsecutiveFailures = 0
	} else {
		status.ConsecutiveFailures++
	}

	if err := s.storage.SaveStatusByEmail(employeeEmail, status); err != nil {
		logger.Error("账号 %s 保存状态失败: %v", employeeEmail, err)
	}
}
