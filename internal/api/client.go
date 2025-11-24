package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code88reset/internal/models"
	"code88reset/pkg/logger"
)

// Client API 客户端
type Client struct {
	BaseURL     string
	APIKey      string
	HTTPClient  *http.Client
	TargetPlans []string // 目标订阅计划名称列表
	Storage     interface {
		SaveAPIResponse(endpoint, method string, requestBody, responseBody []byte, statusCode int) error
	} // 存储接口，用于保存响应
}

// NewClient 创建新的 API 客户端
func NewClient(baseURL, apiKey string, targetPlans []string) *Client {
	return &Client{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		TargetPlans: targetPlans,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest 通用的 HTTP 请求方法
func (c *Client) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := c.BaseURL + endpoint

	var reqBody io.Reader
	var requestData []byte
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		requestData = jsonData
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	logger.Debug("发起请求: %s %s", method, url)

	// 发送请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	logger.Debug("响应状态码: %d", resp.StatusCode)

	// 保存完整的 API 响应体（如果配置了 Storage）
	if c.Storage != nil {
		if err := c.Storage.SaveAPIResponse(endpoint, method, requestData, respBody, resp.StatusCode); err != nil {
			logger.Warn("保存API响应失败: %v", err)
		}
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 尝试解析错误响应
		var errorResp struct {
			Error models.ErrorResponse `json:"error"`
			Type  string               `json:"type"`
		}
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Type == "error" {
			return nil, fmt.Errorf("API错误 [%d]: %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetUsage 获取用量信息
func (c *Client) GetUsage() (*models.UsageResponse, error) {
	logger.Info("获取用量信息...")

	respBody, err := c.makeRequest("POST", "/api/usage", nil)
	if err != nil {
		return nil, err
	}

	var usage models.UsageResponse
	if err := json.Unmarshal(respBody, &usage); err != nil {
		return nil, fmt.Errorf("解析用量响应失败: %w", err)
	}

	logger.Info("用量信息获取成功: 当前积分=%.4f, 限制=%.2f", usage.CurrentCredits, usage.CreditLimit)
	return &usage, nil
}

// GetSubscriptions 获取所有订阅信息
func (c *Client) GetSubscriptions() ([]models.Subscription, error) {
	logger.Info("获取订阅列表...")

	respBody, err := c.makeRequest("POST", "/api/subscription", nil)
	if err != nil {
		return nil, err
	}

	// 尝试解析为新格式 {code: 0, data: [...]}
	var responseWrapper struct {
		Code int                   `json:"code"`
		Data []models.Subscription `json:"data"`
	}

	if err := json.Unmarshal(respBody, &responseWrapper); err == nil && responseWrapper.Data != nil {
		logger.Info("订阅列表获取成功 (新格式)，共 %d 个订阅", len(responseWrapper.Data))
		return responseWrapper.Data, nil
	}

	// 如果新格式解析失败，尝试回退到旧格式 [...]
	var subscriptions []models.Subscription
	if err := json.Unmarshal(respBody, &subscriptions); err != nil {
		return nil, fmt.Errorf("解析订阅列表失败: %w", err)
	}

	logger.Info("订阅列表获取成功，共 %d 个订阅", len(subscriptions))
	return subscriptions, nil
}

// GetTargetSubscription 获取目标订阅信息（根据配置的计划名称）
func (c *Client) GetTargetSubscription() (*models.Subscription, error) {
	subscriptions, err := c.GetSubscriptions()
	if err != nil {
		return nil, err
	}

	targetSet := buildTargetPlanSet(c.TargetPlans)

	for _, sub := range subscriptions {
		if !matchesTargetPlan(sub, targetSet) {
			continue
		}

		// 🚨 PAYGO 保护：永远不重置 PAYGO 类型订阅
		// 检查套餐名称或 PlanType 是否为 PAYGO/PAY_PER_USE
		isPAYGO := sub.SubscriptionName == "PAYGO" ||
			sub.SubscriptionPlan.SubscriptionName == "PAYGO" ||
			sub.SubscriptionPlan.PlanType == "PAYGO" ||
			sub.SubscriptionPlan.PlanType == "PAY_PER_USE"

		if isPAYGO {
			logger.Error("🚨 检测到 PAYGO 订阅 (ID=%d, 名称=%s, 类型=%s)，已自动跳过",
				sub.ID, sub.SubscriptionName, sub.SubscriptionPlan.PlanType)
			logger.Error("🚨 PAYGO 订阅为按量付费，不应进行重置操作")
			continue
		}

		logger.Info("找到目标订阅: 名称=%s, ID=%d, 类型=%s, ResetTimes=%d, Credits=%.4f/%.2f",
			sub.SubscriptionName, sub.ID, sub.SubscriptionPlan.PlanType,
			sub.ResetTimes, sub.CurrentCredits, sub.SubscriptionPlan.CreditLimit)

		return &sub, nil
	}

	return nil, fmt.Errorf("未找到符合条件的目标订阅（目标计划: %v）", c.TargetPlans)
}

// GetFreeSubscription 获取 FREE 订阅信息（保留向后兼容）
func (c *Client) GetFreeSubscription() (*models.Subscription, error) {
	// 临时设置目标为 FREE
	originalPlans := c.TargetPlans
	c.TargetPlans = []string{"FREE"}
	defer func() { c.TargetPlans = originalPlans }()

	return c.GetTargetSubscription()
}

// ResetCredits 重置订阅积分
func (c *Client) ResetCredits(subscriptionID int) (*models.ResetResponse, error) {
	// 🚨 PAYGO 保护：二次确认，防止误重置 PAYGO 订阅
	subscriptions, err := c.GetSubscriptions()
	if err != nil {
		logger.Warn("无法验证订阅类型: %v，继续重置", err)
	} else {
		for _, sub := range subscriptions {
			if sub.ID == subscriptionID {
				// 检查是否为 PAYGO 类型
				isPAYGO := sub.SubscriptionName == "PAYGO" ||
					sub.SubscriptionPlan.SubscriptionName == "PAYGO" ||
					sub.SubscriptionPlan.PlanType == "PAYGO" ||
					sub.SubscriptionPlan.PlanType == "PAY_PER_USE"

				if isPAYGO {
					return nil, fmt.Errorf("🚨 拒绝重置：订阅 ID=%d (名称=%s, 类型=%s) 为 PAYGO 类型，不允许重置",
						subscriptionID, sub.SubscriptionName, sub.SubscriptionPlan.PlanType)
				}
				logger.Debug("已验证订阅 ID=%d 类型=%s，允许重置", subscriptionID, sub.SubscriptionPlan.PlanType)
				break
			}
		}
	}

	endpoint := fmt.Sprintf("/api/reset-credits/%d", subscriptionID)
	logger.Info("重置订阅积分: subscriptionID=%d", subscriptionID)

	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// 尝试解析成功响应
	var resetResp models.ResetResponse

	// 首先尝试解析为标准响应格式
	if err := json.Unmarshal(respBody, &resetResp); err == nil {
		// 解析成功，检查是否有错误
		if resetResp.Error != nil {
			return &resetResp, fmt.Errorf("重置失败: %s", resetResp.Error.Message)
		}
		logger.Info("重置成功: %s", resetResp.Message)
		return &resetResp, nil
	}

	// 如果标准格式解析失败，尝试作为原始响应处理
	logger.Debug("重置响应原始内容: %s", string(respBody))

	// 构造一个基本的成功响应
	resetResp = models.ResetResponse{
		Success: true,
		Message: "重置请求已发送",
	}

	return &resetResp, nil
}

// TestConnection 测试 API 连接
func (c *Client) TestConnection() error {
	logger.Info("测试 API 连接...")

	_, err := c.GetUsage()
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}

	logger.Info("API 连接测试成功")
	return nil
}

// GetAccountInfo 获取账号信息（通过 Usage API）
func (c *Client) GetAccountInfo() (*models.AccountConfig, error) {
	usage, err := c.GetUsage()
	if err != nil {
		return nil, fmt.Errorf("获取账号信息失败: %w", err)
	}

	accountConfig := &models.AccountConfig{
		APIKey:        c.APIKey,
		KeyID:         usage.KeyID,
		EmployeeID:    usage.EmployeeID,
		EmployeeName:  "", // Usage API 没有直接返回，需要从订阅中获取
		EmployeeEmail: "", // Usage API 没有直接返回，需要从订阅中获取
		Name:          usage.Name,
		Enabled:       true,
		AddedAt:       time.Now().Format(time.RFC3339),
	}

	// 尝试从订阅列表中获取员工信息
	subscriptions, err := c.GetSubscriptions()
	if err == nil && len(subscriptions) > 0 {
		// 取第一个订阅的员工信息
		accountConfig.EmployeeName = subscriptions[0].EmployeeName
		accountConfig.EmployeeEmail = subscriptions[0].EmployeeEmail
	}

	logger.Info("账号信息获取成功: KeyID=%s, Name=%s, EmployeeID=%d, Email=%s",
		accountConfig.KeyID, accountConfig.Name, accountConfig.EmployeeID, accountConfig.EmployeeEmail)

	return accountConfig, nil
}

// buildTargetPlanSet 生成标准化的目标套餐集合，方便快速匹配
func buildTargetPlanSet(targetPlans []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, plan := range targetPlans {
		if normalized := normalizePlanIdentifier(plan); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	return set
}

// matchesTargetPlan 判断订阅是否匹配目标套餐
func matchesTargetPlan(sub models.Subscription, normalizedTargets map[string]struct{}) bool {
	if !isMonthlyPlan(sub) {
		return false
	}

	if len(normalizedTargets) == 0 {
		return true
	}

	candidates := []string{
		sub.SubscriptionName,
		sub.SubscriptionPlan.SubscriptionName,
	}

	for _, candidate := range candidates {
		if _, ok := normalizedTargets[normalizePlanIdentifier(candidate)]; ok {
			return true
		}
	}

	return false
}

// normalizePlanIdentifier 标准化套餐标识，便于匹配不同格式
func normalizePlanIdentifier(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)

	// 移除常见分隔符/括号等，保留数字与中英文字符
	replacer := strings.NewReplacer(
		"（", "",
		"）", "",
		"(", "",
		")", "",
		"-", "",
		"_", "",
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"|", "",
		"/", "",
		"\\", "",
		":", "",
		";", "",
		"@", "",
		"#", "",
		"+", "",
		",", "",
		"，", "",
		".", "",
	)

	return replacer.Replace(lower)
}

// isMonthlyPlan 判断订阅是否属于 MONTHLY 类型（可重置）
func isMonthlyPlan(sub models.Subscription) bool {
	planType := strings.TrimSpace(strings.ToUpper(sub.SubscriptionPlan.PlanType))
	if planType == "" {
		return true
	}
	return planType == "MONTHLY"
}
