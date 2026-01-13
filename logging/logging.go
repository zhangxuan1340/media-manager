package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/user/media-manager/utils"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	// DebugLevel 调试级别
	DebugLevel LogLevel = iota
	// InfoLevel 信息级别
	InfoLevel
	// WarningLevel 警告级别
	WarningLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命级别
	FatalLevel
)

// 日志级别对应的字符串
var levelNames = map[LogLevel]string{
	DebugLevel:   "DEBUG",
	InfoLevel:    "INFO",
	WarningLevel: "WARNING",
	ErrorLevel:   "ERROR",
	FatalLevel:   "FATAL",
}

// CurrentLevel 当前日志级别
var CurrentLevel = InfoLevel

// SetLogLevel 设置日志级别
func SetLogLevel(level LogLevel) {
	CurrentLevel = level
}

// GetLogFilePath 获取日志文件路径
func GetLogFilePath() string {
	var logsDir string
	var err error

	// 1. 首先检查用户当前目录下是否存在logs目录（只检查不创建）
	currentDir, err := os.Getwd()
	if err == nil {
		logsDir = filepath.Join(currentDir, "logs")
		if _, err := os.Stat(logsDir); err == nil {
			logFileName := time.Now().Format("2006-01-02") + ".log"
			return filepath.Join(logsDir, logFileName)
		}
	}

	// 2. 检查程序执行文件所在目录下是否存在logs目录（只检查不创建）
	exeDir, err := utils.GetExecutableDir()
	if err == nil {
		logsDir = filepath.Join(exeDir, "logs")
		if _, err := os.Stat(logsDir); err == nil {
			logFileName := time.Now().Format("2006-01-02") + ".log"
			return filepath.Join(logsDir, logFileName)
		}
	}

	// 3. 最后使用用户主目录下的.media-manager/logs目录（不存在则创建）
	homeDir, err := os.UserHomeDir()
	if err == nil {
		logsDir = filepath.Join(homeDir, ".media-manager", "logs")
		// 确保用户主目录下的日志目录存在
		if err := os.MkdirAll(logsDir, 0755); err == nil {
			logFileName := time.Now().Format("2006-01-02") + ".log"
			return filepath.Join(logsDir, logFileName)
		}
	}

	// 如果所有尝试都失败，输出错误并退出
	fmt.Printf("无法创建日志目录\n")
	os.Exit(1)
	return "" // 永远不会执行到这里
}

// log 记录日志的通用函数
func log(level LogLevel, format string, args ...interface{}) {
	// 如果当前级别低于设置的级别，不记录日志
	if level < CurrentLevel {
		return
	}

	// 获取当前时间
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	// 生成日志内容
	logContent := fmt.Sprintf("[%s] %s: %s\n", currentTime, levelNames[level], fmt.Sprintf(format, args...))

	// 输出到控制台
	fmt.Print(logContent)

	// 写入日志文件
	logFilePath := GetLogFilePath()
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法打开日志文件: %v\n", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(logContent); err != nil {
		fmt.Printf("写入日志文件失败: %v\n", err)
		return
	}

	// 如果是致命级别，程序退出
	if level == FatalLevel {
		os.Exit(1)
	}
}

// Debug 记录调试级别日志
func Debug(format string, args ...interface{}) {
	log(DebugLevel, format, args...)
}

// Info 记录信息级别日志
func Info(format string, args ...interface{}) {
	log(InfoLevel, format, args...)
}

// Warning 记录警告级别日志
func Warning(format string, args ...interface{}) {
	log(WarningLevel, format, args...)
}

// Error 记录错误级别日志
func Error(format string, args ...interface{}) {
	log(ErrorLevel, format, args...)
}

// Fatal 记录致命级别日志并退出程序
func Fatal(format string, args ...interface{}) {
	log(FatalLevel, format, args...)
}
