package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	DefaultBaseURL            = "https://www.88code.org"
	DefaultDataDir            = "./data"
	DefaultLogDir             = "./logs"
	DefaultTimezone           = "Asia/Shanghai" // 默认使用北京/上海时区 (UTC+8)
	DefaultCreditThresholdMax = 83.0            // 默认额度上限百分比 83%（当额度>上限时跳过重置）
	DefaultEnableFirstReset   = false           // 默认关闭18:55重置
)

// EnvFile 提供 .env 文件位置（可在测试中重写）
var EnvFile = ".env"

// Settings 汇总运行配置
type Settings struct {
	Mode               string
	APIKeys            []string
	BaseURL            string
	DataDir            string
	LogDir             string
	Plans              []string
	Timezone           string
	CreditThresholdMax float64
	CreditThresholdMin float64
	UseMaxThreshold    bool
	EnableFirstReset   bool
}

// MaskAPIKey 遮蔽 API Key 显示
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:8] + "****"
}

// ParsePlans 解析套餐名称列表
func ParsePlans(plan string) []string {
	if plan == "" {
		return []string{}
	}

	items := strings.Split(plan, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}

// GetTimezone 从多个来源获取时区配置
func GetTimezone(cmdTimezone string) string {
	// 优先级: 命令行参数 > 环境变量 > .env 文件 > 默认值

	// 1. 命令行参数
	if cmdTimezone != "" {
		return cmdTimezone
	}

	// 2. 环境变量 TZ
	if envTZ := os.Getenv("TZ"); envTZ != "" {
		return envTZ
	}

	// 3. 环境变量 TIMEZONE
	if envTimezone := os.Getenv("TIMEZONE"); envTimezone != "" {
		return envTimezone
	}

	// 4. .env 文件
	if tzFromEnv := readTimezoneFromEnv(EnvFile); tzFromEnv != "" {
		return tzFromEnv
	}

	// 5. 默认值
	return DefaultTimezone
}

// GetCreditThresholds 从多个来源获取额度百分比上下限
// 返回值: (thresholdMax, thresholdMin, useMax)
// useMax: true 表示使用上限模式，false 表示使用下限模式
func GetCreditThresholds(cmdMax, cmdMin float64) (float64, float64, bool) {
	// 优先级: 命令行参数 > 环境变量 > .env 文件 > 默认值

	max := 0.0
	min := 0.0

	// 1. 命令行参数
	if cmdMax > 0 {
		max = cmdMax
	}
	if cmdMin > 0 {
		min = cmdMin
	}

	// 2. 环境变量
	if max == 0 {
		if envMax := os.Getenv("CREDIT_THRESHOLD_MAX"); envMax != "" {
			if val, err := parseFloat(envMax); err == nil && val > 0 {
				max = val
			}
		}
	}
	if min == 0 {
		if envMin := os.Getenv("CREDIT_THRESHOLD_MIN"); envMin != "" {
			if val, err := parseFloat(envMin); err == nil && val > 0 {
				min = val
			}
		}
	}

	// 3. .env 文件
	if max == 0 {
		if val := readCreditThresholdMaxFromEnv(EnvFile); val > 0 {
			max = val
		}
	}
	if min == 0 {
		if val := readCreditThresholdMinFromEnv(EnvFile); val > 0 {
			min = val
		}
	}

	// 4. 默认值（仅上限有默认值）
	if max == 0 && min == 0 {
		max = DefaultCreditThresholdMax
	}

	// 5. 确定模式：优先使用上限，其次使用下限
	useMax := max > 0

	return max, min, useMax
}

// GetEnableFirstReset 从多个来源获取是否启用18:55重置
func GetEnableFirstReset(cmdEnable bool) bool {
	// 优先级: 命令行参数 > 环境变量 > .env 文件 > 默认值

	// 1. 命令行参数（如果显式设置）
	if cmdEnable {
		return true
	}

	// 2. 环境变量 ENABLE_FIRST_RESET
	if envEnable := os.Getenv("ENABLE_FIRST_RESET"); envEnable != "" {
		return parseBool(envEnable)
	}

	// 3. .env 文件
	if enable := readEnableFirstResetFromEnv(EnvFile); enable {
		return true
	}

	// 4. 默认值
	return DefaultEnableFirstReset
}

// GetAllAPIKeys 从多个来源获取全部 API Keys
func GetAllAPIKeys(cmdKey, cmdKeys string) []string {
	var allKeys []string

	// 优先级: 命令行参数 > 环境变量 > .env 文件

	// 1. -apikeys 参数
	if cmdKeys != "" {
		allKeys = append(allKeys, splitAndTrim(cmdKeys)...)
	}

	// 2. -apikey 参数
	if cmdKey != "" {
		allKeys = append(allKeys, splitAndTrim(cmdKey)...)
	}

	// 3. 环境变量
	if len(allKeys) == 0 {
		if envKeys := os.Getenv("API_KEYS"); envKeys != "" {
			allKeys = append(allKeys, splitAndTrim(envKeys)...)
		}

		if len(allKeys) == 0 {
			if envKey := os.Getenv("API_KEY"); envKey != "" {
				allKeys = append(allKeys, splitAndTrim(envKey)...)
			}
		}
	}

	// 4. .env 文件
	if len(allKeys) == 0 {
		if keysStr := readAPIKeysFromEnv(EnvFile); keysStr != "" {
			allKeys = append(allKeys, splitAndTrim(keysStr)...)
		}

		if len(allKeys) == 0 {
			if keyStr := readAPIKeyFromEnv(EnvFile); keyStr != "" {
				allKeys = append(allKeys, splitAndTrim(keyStr)...)
			}
		}
	}

	return allKeys
}

func splitAndTrim(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func readTimezoneFromEnv(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 支持多种格式
		if strings.HasPrefix(line, "TZ=") {
			return strings.TrimPrefix(line, "TZ=")
		}
		if strings.HasPrefix(line, "TIMEZONE=") {
			return strings.TrimPrefix(line, "TIMEZONE=")
		}
	}

	return ""
}

func readCreditThresholdMaxFromEnv(filename string) float64 {
	file, err := os.Open(filename)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "CREDIT_THRESHOLD_MAX=") {
			valueStr := strings.TrimPrefix(line, "CREDIT_THRESHOLD_MAX=")
			if threshold, err := parseFloat(valueStr); err == nil {
				return threshold
			}
		}
	}

	return 0
}

func readCreditThresholdMinFromEnv(filename string) float64 {
	file, err := os.Open(filename)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "CREDIT_THRESHOLD_MIN=") {
			valueStr := strings.TrimPrefix(line, "CREDIT_THRESHOLD_MIN=")
			if threshold, err := parseFloat(valueStr); err == nil {
				return threshold
			}
		}
	}

	return 0
}

func readEnableFirstResetFromEnv(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "ENABLE_FIRST_RESET=") {
			valueStr := strings.TrimPrefix(line, "ENABLE_FIRST_RESET=")
			return parseBool(valueStr)
		}
	}

	return false
}

func readAPIKeysFromEnv(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 支持多种格式
		if strings.HasPrefix(line, "api-keys=") {
			return strings.TrimPrefix(line, "api-keys=")
		}
		if strings.HasPrefix(line, "API_KEYS=") {
			return strings.TrimPrefix(line, "API_KEYS=")
		}
	}

	return ""
}

func readAPIKeyFromEnv(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 支持多种格式
		if strings.HasPrefix(line, "api-key=") {
			return strings.TrimPrefix(line, "api-key=")
		}
		if strings.HasPrefix(line, "API_KEY=") {
			return strings.TrimPrefix(line, "API_KEY=")
		}
	}

	return ""
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}

func parseBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "true" || s == "1" || s == "yes" || s == "on" || s == "enabled"
}
