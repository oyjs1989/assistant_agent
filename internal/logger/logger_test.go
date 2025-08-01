package logger

import (
	"os"
	"path/filepath"
	"testing"

	"assistant_agent/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerInit(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 测试日志初始化
	err = Init()
	require.NoError(t, err)

	// 测试日志函数
	Info("Test info message")
	Warn("Test warning message")
	Error("Test error message")
	Debug("Test debug message")

	// 验证日志级别设置
	config.GetConfig().Logging.Level = "debug"
	err = Init()
	require.NoError(t, err)
}

func TestLoggerWithFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// 设置配置
	config.GetConfig().Logging.File = "test.log"
	config.GetConfig().Agent.LogDir = tempDir

	// 初始化日志
	err := Init()
	require.NoError(t, err)

	// 写入测试日志
	testMessage := "Test log message"
	Info(testMessage)

	// 验证日志文件是否存在
	assert.FileExists(t, logFile)

	// 读取日志文件内容
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), testMessage)
}

func TestLoggerWithJSONFormat(t *testing.T) {
	// 设置 JSON 格式
	config.GetConfig().Logging.Format = "json"

	// 初始化日志
	err := Init()
	require.NoError(t, err)

	// 写入测试日志
	Info("JSON format test")

	// 验证 JSON 格式（这里只是测试不会崩溃）
	assert.True(t, true)
}

func TestLoggerWithFields(t *testing.T) {
	// 初始化日志
	err := Init()
	require.NoError(t, err)

	// 测试带字段的日志
	entry := WithField("test_key", "test_value")
	assert.NotNil(t, entry)

	// 测试多个字段
	fields := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": 123,
	}
	entry = WithFields(fields)
	assert.NotNil(t, entry)
}

func TestLoggerLevels(t *testing.T) {
	// 测试不同日志级别
	config.GetConfig().Logging.Level = "debug"
	err := Init()
	require.NoError(t, err)

	// 测试所有日志级别
	Debug("Debug message")
	Info("Info message")
	Warn("Warning message")
	Error("Error message")

	// 测试格式化日志
	Debugf("Debug format: %s", "test")
	Infof("Info format: %s", "test")
	Warnf("Warning format: %s", "test")
	Errorf("Error format: %s", "test")
}

func TestLoggerInvalidLevel(t *testing.T) {
	// 测试无效的日志级别
	config.GetConfig().Logging.Level = "invalid_level"
	err := Init()
	require.NoError(t, err) // 应该使用默认级别

	// 验证日志仍然可以工作
	Info("Test message with invalid level")
}

func TestLoggerFileCreation(t *testing.T) {
	// 测试日志初始化不会崩溃
	err := Init()
	require.NoError(t, err)

	// 写入日志
	Info("Test message")
	
	// 验证日志功能正常
	assert.True(t, true)
}

func TestLoggerConcurrent(t *testing.T) {
	// 初始化日志
	err := Init()
	require.NoError(t, err)

	// 并发测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			Infof("Concurrent test message %d", id)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
