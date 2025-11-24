package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code88reset/internal/models"
	"code88reset/pkg/logger"
)

const (
	AccountFile      = "account.json"
	StatusFile       = "status.json"
	LockFile         = "reset.lock"
	ResponseLogDir   = "responses"     // API响应体保存目录
	MultiAccountFile = "accounts.json" // 多账号配置文件
	AccountsDir      = "accounts"      // 多账号数据目录
)

// Storage 存储管理器
type Storage struct {
	dataDir       string
	saveResponses bool
	mu            sync.RWMutex
}

// NewStorage 创建新的存储管理器
func NewStorage(dataDir string, saveResponses bool) (*Storage, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	return &Storage{
		dataDir:       dataDir,
		saveResponses: saveResponses,
	}, nil
}

// SaveAccountInfo 保存账号信息
func (s *Storage) SaveAccountInfo(account *models.AccountInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account.LastUpdated = time.Now()

	filePath := filepath.Join(s.dataDir, AccountFile)
	return s.saveJSON(filePath, account)
}

// LoadAccountInfo 加载账号信息
func (s *Storage) LoadAccountInfo() (*models.AccountInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.dataDir, AccountFile)
	var account models.AccountInfo

	if err := s.loadJSON(filePath, &account); err != nil {
		if os.IsNotExist(err) {
			logger.Warn("账号信息文件不存在，将创建新文件")
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// SaveStatus 保存执行状态
func (s *Storage) SaveStatus(status *models.ExecutionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status.LastCheckTime = time.Now()

	filePath := filepath.Join(s.dataDir, StatusFile)
	return s.saveJSON(filePath, status)
}

// LoadStatus 加载执行状态
func (s *Storage) LoadStatus() (*models.ExecutionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.dataDir, StatusFile)
	var status models.ExecutionStatus

	if err := s.loadJSON(filePath, &status); err != nil {
		if os.IsNotExist(err) {
			logger.Warn("状态文件不存在，将创建新文件")
			return s.initializeStatus(), nil
		}
		return nil, err
	}

	// 检查日期是否变化，如果是新的一天，重置今日标志
	today := time.Now().Format("2006-01-02")
	if status.TodayDate != today {
		logger.Info("检测到日期变化: %s -> %s，重置今日标志", status.TodayDate, today)
		status.TodayDate = today
		status.FirstResetToday = false
		status.SecondResetToday = false
		status.ResetTimesBeforeReset = 0
		status.ResetTimesAfterReset = 0
		status.CreditsBeforeReset = 0
		status.CreditsAfterReset = 0
	}

	return &status, nil
}

// initializeStatus 初始化状态
func (s *Storage) initializeStatus() *models.ExecutionStatus {
	today := time.Now().Format("2006-01-02")
	return &models.ExecutionStatus{
		LastCheckTime:       time.Now(),
		FirstResetToday:     false,
		SecondResetToday:    false,
		LastResetSuccess:    false,
		ConsecutiveFailures: 0,
		TodayDate:           today,
	}
}

// AcquireLock 获取锁
func (s *Storage) AcquireLock(operation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lockPath := filepath.Join(s.dataDir, LockFile)

	// 检查锁文件是否存在
	if _, err := os.Stat(lockPath); err == nil {
		// 锁文件存在，读取锁信息
		var existingLock models.LockFile
		if err := s.loadJSON(lockPath, &existingLock); err == nil {
			// 检查锁是否过期（超过 10 分钟认为是僵尸锁）
			if time.Since(existingLock.StartTime) < 10*time.Minute {
				return fmt.Errorf("操作正在进行中: %s (PID: %d, 开始时间: %s)",
					existingLock.Operation, existingLock.PID, existingLock.StartTime.Format("15:04:05"))
			}
			logger.Warn("检测到僵尸锁文件，将覆盖 (PID: %d, 时间: %s)",
				existingLock.PID, existingLock.StartTime.Format("2006-01-02 15:04:05"))
		}
	}

	// 创建新的锁
	hostname, _ := os.Hostname()
	lock := models.LockFile{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Operation: operation,
		Hostname:  hostname,
	}

	if err := s.saveJSON(lockPath, lock); err != nil {
		return fmt.Errorf("创建锁文件失败: %w", err)
	}

	logger.Debug("获取锁成功: %s (PID: %d)", operation, lock.PID)
	return nil
}

// ReleaseLock 释放锁
func (s *Storage) ReleaseLock() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lockPath := filepath.Join(s.dataDir, LockFile)

	if err := os.Remove(lockPath); err != nil {
		if os.IsNotExist(err) {
			logger.Debug("锁文件不存在，无需释放")
			return nil
		}
		return fmt.Errorf("释放锁失败: %w", err)
	}

	logger.Debug("锁释放成功")
	return nil
}

// IsLocked 检查是否有锁
func (s *Storage) IsLocked() (bool, *models.LockFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lockPath := filepath.Join(s.dataDir, LockFile)

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return false, nil, nil
	}

	var lock models.LockFile
	if err := s.loadJSON(lockPath, &lock); err != nil {
		return true, nil, fmt.Errorf("读取锁文件失败: %w", err)
	}

	// 检查锁是否过期
	if time.Since(lock.StartTime) > 10*time.Minute {
		return false, &lock, nil // 锁已过期，视为未锁定
	}

	return true, &lock, nil
}

// saveJSON 保存 JSON 到文件
func (s *Storage) saveJSON(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	// 先写入临时文件，然后重命名（原子操作）
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	logger.Debug("保存文件成功: %s", filePath)
	return nil
}

// loadJSON 从文件加载 JSON
func (s *Storage) loadJSON(filePath string, target interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return nil
}

// SaveMultiAccountConfig 保存多账号配置
func (s *Storage) SaveMultiAccountConfig(config *models.MultiAccountConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dataDir, MultiAccountFile)
	return s.saveJSON(filePath, config)
}

// LoadMultiAccountConfig 加载多账号配置
func (s *Storage) LoadMultiAccountConfig() (*models.MultiAccountConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.dataDir, MultiAccountFile)
	var config models.MultiAccountConfig

	if err := s.loadJSON(filePath, &config); err != nil {
		if os.IsNotExist(err) {
			logger.Warn("多账号配置文件不存在，返回空配置")
			return &models.MultiAccountConfig{Accounts: []models.AccountConfig{}}, nil
		}
		return nil, err
	}

	return &config, nil
}

// GetAccountDataDir 获取指定账号的数据目录（基于 employeeEmail）
func (s *Storage) GetAccountDataDir(employeeEmail string) string {
	// 使用邮箱作为目录名，确保安全
	safeEmail := filepath.Base(employeeEmail)
	return filepath.Join(s.dataDir, AccountsDir, safeEmail)
}

// SaveAccountInfoByEmail 保存指定账号的信息
func (s *Storage) SaveAccountInfoByEmail(employeeEmail string, account *models.AccountInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account.LastUpdated = time.Now()
	accountDir := s.GetAccountDataDir(employeeEmail)

	// 确保账号目录存在
	if err := os.MkdirAll(accountDir, 0755); err != nil {
		return fmt.Errorf("创建账号目录失败: %w", err)
	}

	filePath := filepath.Join(accountDir, AccountFile)
	return s.saveJSON(filePath, account)
}

// LoadAccountInfoByEmail 加载指定账号的信息
func (s *Storage) LoadAccountInfoByEmail(employeeEmail string) (*models.AccountInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accountDir := s.GetAccountDataDir(employeeEmail)
	filePath := filepath.Join(accountDir, AccountFile)
	var account models.AccountInfo

	if err := s.loadJSON(filePath, &account); err != nil {
		if os.IsNotExist(err) {
			logger.Warn("账号 %s 的信息文件不存在", employeeEmail)
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// SaveStatusByEmail 保存指定账号的执行状态
func (s *Storage) SaveStatusByEmail(employeeEmail string, status *models.ExecutionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status.LastCheckTime = time.Now()
	accountDir := s.GetAccountDataDir(employeeEmail)

	// 确保账号目录存在
	if err := os.MkdirAll(accountDir, 0755); err != nil {
		return fmt.Errorf("创建账号目录失败: %w", err)
	}

	filePath := filepath.Join(accountDir, StatusFile)
	return s.saveJSON(filePath, status)
}

// LoadStatusByEmail 加载指定账号的执行状态
func (s *Storage) LoadStatusByEmail(employeeEmail string) (*models.ExecutionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accountDir := s.GetAccountDataDir(employeeEmail)
	filePath := filepath.Join(accountDir, StatusFile)
	var status models.ExecutionStatus

	if err := s.loadJSON(filePath, &status); err != nil {
		if os.IsNotExist(err) {
			logger.Warn("账号 %s 的状态文件不存在，将创建新文件", employeeEmail)
			return s.initializeStatus(), nil
		}
		return nil, err
	}

	// 检查日期是否变化
	today := time.Now().Format("2006-01-02")
	if status.TodayDate != today {
		logger.Info("账号 %s 检测到日期变化: %s -> %s，重置今日标志", employeeEmail, status.TodayDate, today)
		status.TodayDate = today
		status.FirstResetToday = false
		status.SecondResetToday = false
		status.ResetTimesBeforeReset = 0
		status.ResetTimesAfterReset = 0
		status.CreditsBeforeReset = 0
		status.CreditsAfterReset = 0
	}

	return &status, nil
}

// SaveAPIResponse 保存 API 响应体用于调试
func (s *Storage) SaveAPIResponse(endpoint, method string, requestBody, responseBody []byte, statusCode int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 创建响应日志目录
	responseDir := filepath.Join(s.dataDir, ResponseLogDir)
	if err := os.MkdirAll(responseDir, 0755); err != nil {
		return fmt.Errorf("创建响应日志目录失败: %w", err)
	}

	// 生成文件名：endpoint_timestamp.json
	timestamp := time.Now().Format("20060102_150405")
	safeEndpoint := filepath.Base(endpoint) // 避免路径问题
	if safeEndpoint == "." || safeEndpoint == "/" {
		safeEndpoint = "root"
	}
	fileName := fmt.Sprintf("%s_%s_%s.json", method, safeEndpoint, timestamp)
	filePath := filepath.Join(responseDir, fileName)

	// 构造完整的响应记录
	responseLog := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"method":      method,
		"endpoint":    endpoint,
		"status_code": statusCode,
	}

	// 处理请求体（可能为空）
	if len(requestBody) > 0 {
		responseLog["request_body"] = json.RawMessage(requestBody)
	} else {
		responseLog["request_body"] = nil
	}

	// 处理响应体（可能为空）
	if len(responseBody) > 0 {
		responseLog["response_body"] = json.RawMessage(responseBody)
	} else {
		responseLog["response_body"] = ""
	}

	// 检查是否开启了响应保存
	if !s.saveResponses {
		logger.Debug("API响应未保存 (设置 SAVE_RESPONSES=true 开启)")
		return nil
	}

	// 保存为格式化的 JSON
	data, err := json.MarshalIndent(responseLog, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化响应日志失败: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("保存响应日志失败: %w", err)
	}

	logger.Info("API响应已保存: %s", filePath)
	return nil
}
