package scheduler

import (
	"fmt"
	"time"

	"code88reset/internal/config"
	"code88reset/internal/models"
	"code88reset/internal/storage"
	"code88reset/pkg/logger"
)

// BaseScheduler 包含调度器的公共逻辑和依赖
type BaseScheduler struct {
	storage            *storage.Storage
	location           *time.Location
	creditThresholdMax float64
	creditThresholdMin float64
	useMaxThreshold    bool
	enableFirstReset   bool
	loop               *loopController
	accountUpdater     accountUpdater
	logAgg             *logAggregator
}

func newBaseScheduler(
	storage *storage.Storage,
	timezone string,
	thresholdMax, thresholdMin float64,
	useMax, enableFirstReset bool,
	logLabel string,
) (*BaseScheduler, error) {
	if timezone == "" {
		timezone = config.BeijingTimezone
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("加载时区失败 (%s): %w", timezone, err)
	}

	return &BaseScheduler{
		storage:            storage,
		location:           loc,
		creditThresholdMax: thresholdMax,
		creditThresholdMin: thresholdMin,
		useMaxThreshold:    useMax,
		enableFirstReset:   enableFirstReset,
		loop:               newLoopController(SubscriptionCheckInterval),
		accountUpdater:     newAccountUpdater(storage),
		logAgg:             newLogAggregator(logLabel, 5*time.Minute),
	}, nil
}

func (s *BaseScheduler) Start(logInfo func()) {
	logger.Info("========================================")
	logger.Info("%s启动", s.logAgg.label)
	logger.Info("时区: %s", s.location.String())
	if s.enableFirstReset {
		logger.Info("第一次重置时间: %02d:%02d (已启用)", config.FirstResetHour, config.FirstResetMinute)
	} else {
		logger.Info("第一次重置时间: %02d:%02d (已禁用)", config.FirstResetHour, config.FirstResetMinute)
	}
	logger.Info("第二次重置时间: %02d:%02d", config.SecondResetHour, config.SecondResetMinute)

	if s.useMaxThreshold && s.creditThresholdMax > 0 {
		logger.Info("额度判断模式: 上限模式 - 当额度 > %.1f%% 时跳过18点重置", s.creditThresholdMax)
	} else if !s.useMaxThreshold && s.creditThresholdMin > 0 {
		logger.Info("额度判断模式: 下限模式 - 当额度 < %.1f%% 时才执行18点重置", s.creditThresholdMin)
	} else {
		logger.Info("额度判断模式: 已禁用")
	}

	if logInfo != nil {
		logInfo()
	}

	logger.Info("========================================")
}

func (s *BaseScheduler) Stop() {
	logger.Info("正在停止%s...", s.logAgg.label)
	s.loop.Stop()
	s.logAgg.Flush()
}

func (s *BaseScheduler) checkTime() (bool, string) {
	now := time.Now().In(s.location)
	currentHour := now.Hour()
	currentMinute := now.Minute()

	s.logAgg.Add("检查时间: %s", now.Format("2006-01-02 15:04:05"))

	if currentHour == config.FirstResetHour && currentMinute == config.FirstResetMinute {
		if !s.enableFirstReset {
			logger.Debug("18:55重置已禁用，跳过")
			s.logAgg.Flush()
			return false, ""
		}
		s.logAgg.Flush()
		return true, "first"
	}

	if currentHour == config.SecondResetHour && currentMinute == config.SecondResetMinute {
		s.logAgg.Flush()
		return true, "second"
	}

	return false, ""
}

func (s *BaseScheduler) recordFailure(status *models.ExecutionStatus, message, resetType string) {
	now := time.Now()
	if resetType == "first" {
		status.FirstResetToday = true
		status.LastFirstResetTime = &now
	} else {
		status.SecondResetToday = true
		status.LastSecondResetTime = &now
	}
	status.LastResetSuccess = false
	status.LastResetMessage = message
	status.ConsecutiveFailures++
	if err := s.storage.SaveStatus(status); err != nil {
		logger.Error("保存状态失败: %v", err)
	}
}

func (s *BaseScheduler) recordSkip(status *models.ExecutionStatus, resetType string, reason string) {
	now := time.Now()
	if resetType == "first" {
		status.FirstResetToday = true
		status.LastFirstResetTime = &now
	} else {
		status.SecondResetToday = true
		status.LastSecondResetTime = &now
	}
	status.LastResetSuccess = true
	status.LastResetMessage = fmt.Sprintf("跳过: %s", reason)
	if err := s.storage.SaveStatus(status); err != nil {
		logger.Error("保存状态失败: %v", err)
	}
}
