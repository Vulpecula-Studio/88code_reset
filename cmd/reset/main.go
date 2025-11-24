package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"code88reset/internal/account"
	"code88reset/internal/app"
	appconfig "code88reset/internal/config"
	"code88reset/internal/storage"
	"code88reset/pkg/logger"
)

var (
	mode               = flag.String("mode", "test", "运行模式: test(测试), run(自动调度器), list(列出历史账号)")
	apiKey             = flag.String("apikey", "", "API Key，支持单个或多个（逗号分隔），优先使用环境变量或.env文件")
	apiKeys            = flag.String("apikeys", "", "多个 API Keys（逗号分隔），与 -apikey 等效")
	baseURL            = flag.String("baseurl", appconfig.DefaultBaseURL, "API Base URL")
	dataDir            = flag.String("datadir", appconfig.DefaultDataDir, "数据目录")
	logDir             = flag.String("logdir", appconfig.DefaultLogDir, "日志目录")
	planNames          = flag.String("plans", "", "要重置的订阅计划名称（匹配 subscriptionName），多个用逗号分隔；留空表示所有 MONTHLY 套餐")
	timezone           = flag.String("timezone", "", "时区设置 (例如: Asia/Shanghai, Asia/Hong_Kong, UTC)")
	creditThresholdMax = flag.Float64("threshold-max", 0, "额度上限百分比(0-100)，当额度>上限时跳过18点重置，0表示使用环境变量或默认值83")
	creditThresholdMin = flag.Float64("threshold-min", 0, "额度下限百分比(0-100)，当额度<下限时才执行18点重置，0表示不使用下限")
	enableFirstReset   = flag.Bool("first-reset", false, "是否启用18:55重置，默认关闭")
)

func main() {
	flag.Parse()

	if err := logger.Init(*logDir); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}

	logger.Info("========================================")
	logger.Info("88code FREE 订阅重置工具")
	logger.Info("========================================")
	logger.Info("运行模式: %s", *mode)

	// 解析配置
	tz := appconfig.GetTimezone(*timezone)
	thresholdMax, thresholdMin, useMax := appconfig.GetCreditThresholds(*creditThresholdMax, *creditThresholdMin)
	firstReset := appconfig.GetEnableFirstReset(*enableFirstReset)

	logger.Info("Base URL: %s", *baseURL)
	logger.Info("时区设置: %s", tz)
	logger.Info("数据目录: %s", *dataDir)
	logger.Info("日志目录: %s", *logDir)
	if strings.TrimSpace(*planNames) == "" {
		logger.Info("目标套餐: 所有 MONTHLY 套餐")
	} else {
		logger.Info("目标套餐: %s", *planNames)
	}
	if useMax {
		logger.Info("额度判断模式: 上限模式 - 当额度 > %.1f%% 时跳过18点重置", thresholdMax)
	} else if thresholdMin > 0 {
		logger.Info("额度判断模式: 下限模式 - 当额度 < %.1f%% 时才执行18点重置", thresholdMin)
	} else {
		logger.Info("额度判断模式: 已禁用")
	}
	logger.Info("18:55重置: %v", firstReset)

	plans := appconfig.ParsePlans(*planNames)
	keys := appconfig.GetAllAPIKeys(*apiKey, *apiKeys)
	saveResponses := appconfig.GetSaveResponses()

	// 初始化依赖
	store, err := storage.NewStorage(*dataDir, saveResponses)
	if err != nil {
		logger.Error("初始化存储失败: %v", err)
		os.Exit(1)
	}
	accountMgr := account.NewManager(store, *baseURL)

	cfg := appconfig.Settings{
		Mode:               *mode,
		APIKeys:            keys,
		BaseURL:            *baseURL,
		DataDir:            *dataDir,
		LogDir:             *logDir,
		Plans:              plans,
		Timezone:           tz,
		CreditThresholdMax: thresholdMax,
		CreditThresholdMin: thresholdMin,
		UseMaxThreshold:    useMax,
		EnableFirstReset:   firstReset,
		SaveResponses:      saveResponses,
	}

	application := app.New(cfg, store, accountMgr)
	if err := application.Run(); err != nil {
		logger.Error("程序运行失败: %v", err)
		os.Exit(1)
	}
}
