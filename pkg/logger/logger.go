package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLog  *log.Logger
	WarnLog  *log.Logger
	ErrorLog *log.Logger
	DebugLog *log.Logger
)

// Init 初始化日志系统
func Init(logDir string) error {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 生成日志文件名（按日期）
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("reset_%s.log", today))

	// 打开日志文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 同时输出到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, file)

	// 初始化不同级别的日志记录器
	InfoLog = log.New(multiWriter, "[INFO]  ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLog = log.New(multiWriter, "[WARN]  ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLog = log.New(multiWriter, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)

	InfoLog.Println("========================================")
	InfoLog.Printf("日志系统初始化成功，日志文件: %s", logFile)
	InfoLog.Println("========================================")

	return nil
}

// Info 记录信息日志
func Info(format string, v ...interface{}) {
	if InfoLog != nil {
		InfoLog.Printf(format, v...)
	}
}

// Warn 记录警告日志
func Warn(format string, v ...interface{}) {
	if WarnLog != nil {
		WarnLog.Printf(format, v...)
	}
}

// Error 记录错误日志
func Error(format string, v ...interface{}) {
	if ErrorLog != nil {
		ErrorLog.Printf(format, v...)
	}
}

// Debug 记录调试日志
func Debug(format string, v ...interface{}) {
	if DebugLog != nil {
		DebugLog.Printf(format, v...)
	}
}
