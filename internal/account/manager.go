package account

import (
	"fmt"
	"strings"

	"code88reset/internal/api"
	"code88reset/internal/models"
	"code88reset/internal/storage"
	"code88reset/pkg/logger"
)

// Manager 账号管理器
type Manager struct {
	storage *storage.Storage
	baseURL string
}

// NewManager 创建账号管理器
func NewManager(storage *storage.Storage, baseURL string) *Manager {
	return &Manager{
		storage: storage,
		baseURL: baseURL,
	}
}

// SyncAccountsFromAPIKeys 从 API Keys 同步账号信息（持久化到配置文件）
// 这个方法会：
// 1. 为新的 API Key 创建账号记录
// 2. 为已存在的账号更新 API Key
// 3. 不会删除任何历史账号（即使 API Key 不在当前列表中）
func (m *Manager) SyncAccountsFromAPIKeys(apiKeys []string, targetPlans []string) error {
	logger.Info("同步 %d 个 API Key 到账号配置...", len(apiKeys))

	// 加载现有配置
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建 employeeEmail 映射以快速查找现有账号
	existingEmails := make(map[string]*models.AccountConfig)
	for i := range config.Accounts {
		existingEmails[config.Accounts[i].EmployeeEmail] = &config.Accounts[i]
	}

	successCount := 0
	failCount := 0
	updateCount := 0

	for i, apiKey := range apiKeys {
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			continue
		}

		logger.Info("[%d/%d] 正在处理 API Key...", i+1, len(apiKeys))

		// 创建临时客户端获取账号信息
		client := api.NewClient(m.baseURL, apiKey, targetPlans)
		accountConfig, err := client.GetAccountInfo()
		if err != nil {
			logger.Error("获取账号信息失败: %v", err)
			failCount++
			continue
		}

		// 检查是否已存在（基于 employeeEmail）
		if existing, exists := existingEmails[accountConfig.EmployeeEmail]; exists {
			// 账号已存在，更新 API Key（使用最新的）
			logger.Info("账号已存在，更新 API Key: Email=%s, Name=%s",
				accountConfig.EmployeeEmail, accountConfig.Name)
			existing.APIKey = accountConfig.APIKey
			existing.KeyID = accountConfig.KeyID
			existing.Name = accountConfig.Name
			updateCount++
		} else {
			// 新账号，添加到配置
			config.Accounts = append(config.Accounts, *accountConfig)
			existingEmails[accountConfig.EmployeeEmail] = accountConfig

			logger.Info("成功添加新账号: Email=%s, Name=%s, EmployeeID=%d",
				accountConfig.EmployeeEmail, accountConfig.Name, accountConfig.EmployeeID)
			successCount++
		}
	}

	// 保存配置
	if err := m.storage.SaveMultiAccountConfig(config); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	logger.Info("同步完成: 新增 %d 个，更新 %d 个，失败 %d 个，历史总计 %d 个账号",
		successCount, updateCount, failCount, len(config.Accounts))

	return nil
}

// GetActiveAccountsFromAPIKeys 获取当前 API Keys 对应的活跃账号
// 只返回在当前 API Keys 列表中的账号，用于执行重置任务
func (m *Manager) GetActiveAccountsFromAPIKeys(apiKeys []string) ([]models.AccountConfig, error) {
	logger.Info("从 %d 个 API Key 中获取活跃账号...", len(apiKeys))

	// 加载现有配置
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建 API Key 到账号的映射
	apiKeyMap := make(map[string]bool)
	for _, key := range apiKeys {
		key = strings.TrimSpace(key)
		if key != "" {
			apiKeyMap[key] = true
		}
	}

	// 创建 employeeEmail 到账号配置的映射（用于去重）
	emailToAccount := make(map[string]*models.AccountConfig)

	// 筛选出在当前 API Keys 列表中的账号
	activeAccounts := []models.AccountConfig{}
	for i := range config.Accounts {
		acc := &config.Accounts[i]

		// 检查此账号的 API Key 是否在当前列表中
		if apiKeyMap[acc.APIKey] {
			// 使用 employeeEmail 去重（可能多个 API Key 属于同一账号）
			if existing, exists := emailToAccount[acc.EmployeeEmail]; exists {
				// 如果已存在，使用最新的 API Key
				logger.Debug("发现重复账号 %s，保留最新的 API Key", acc.EmployeeEmail)
				existing.APIKey = acc.APIKey
				existing.KeyID = acc.KeyID
				existing.Name = acc.Name
			} else {
				// 新账号，添加到活跃列表
				emailToAccount[acc.EmployeeEmail] = acc
				activeAccounts = append(activeAccounts, *acc)
			}
		}
	}

	logger.Info("找到 %d 个活跃账号（已通过 employeeEmail 去重）", len(activeAccounts))

	return activeAccounts, nil
}

// ListAccounts 列出所有账号
func (m *Manager) ListAccounts() ([]models.AccountConfig, error) {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return nil, err
	}

	return config.Accounts, nil
}

// GetAccount 获取指定账号（基于 employeeEmail）
func (m *Manager) GetAccount(employeeEmail string) (*models.AccountConfig, error) {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return nil, err
	}

	for _, acc := range config.Accounts {
		if acc.EmployeeEmail == employeeEmail {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("账号不存在: Email=%s", employeeEmail)
}

// EnableAccount 启用账号（基于 employeeEmail）
func (m *Manager) EnableAccount(employeeEmail string) error {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return err
	}

	found := false
	for i := range config.Accounts {
		if config.Accounts[i].EmployeeEmail == employeeEmail {
			config.Accounts[i].Enabled = true
			found = true
			logger.Info("已启用账号: Email=%s", employeeEmail)
			break
		}
	}

	if !found {
		return fmt.Errorf("账号不存在: Email=%s", employeeEmail)
	}

	return m.storage.SaveMultiAccountConfig(config)
}

// DisableAccount 禁用账号（基于 employeeEmail）
func (m *Manager) DisableAccount(employeeEmail string) error {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return err
	}

	found := false
	for i := range config.Accounts {
		if config.Accounts[i].EmployeeEmail == employeeEmail {
			config.Accounts[i].Enabled = false
			found = true
			logger.Info("已禁用账号: Email=%s", employeeEmail)
			break
		}
	}

	if !found {
		return fmt.Errorf("账号不存在: Email=%s", employeeEmail)
	}

	return m.storage.SaveMultiAccountConfig(config)
}

// RemoveAccount 移除账号（基于 employeeEmail）
func (m *Manager) RemoveAccount(employeeEmail string) error {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return err
	}

	newAccounts := []models.AccountConfig{}
	found := false
	for _, acc := range config.Accounts {
		if acc.EmployeeEmail == employeeEmail {
			found = true
			logger.Info("已移除账号: Email=%s, Name=%s", employeeEmail, acc.Name)
			continue
		}
		newAccounts = append(newAccounts, acc)
	}

	if !found {
		return fmt.Errorf("账号不存在: Email=%s", employeeEmail)
	}

	config.Accounts = newAccounts
	return m.storage.SaveMultiAccountConfig(config)
}

// GetEnabledAccounts 获取所有启用的账号
func (m *Manager) GetEnabledAccounts() ([]models.AccountConfig, error) {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return nil, err
	}

	enabledAccounts := []models.AccountConfig{}
	for _, acc := range config.Accounts {
		if acc.Enabled {
			enabledAccounts = append(enabledAccounts, acc)
		}
	}

	return enabledAccounts, nil
}

// GetAccountCount 获取账号数量统计
func (m *Manager) GetAccountCount() (total, enabled, disabled int, err error) {
	config, err := m.storage.LoadMultiAccountConfig()
	if err != nil {
		return 0, 0, 0, err
	}

	total = len(config.Accounts)
	for _, acc := range config.Accounts {
		if acc.Enabled {
			enabled++
		} else {
			disabled++
		}
	}

	return total, enabled, disabled, nil
}
