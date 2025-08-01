package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// 清理全局配置
	GlobalConfig = nil

	// 测试初始化
	err := Init()
	require.NoError(t, err)
	assert.NotNil(t, GlobalConfig)
}

func TestGetConfig(t *testing.T) {
	// 确保配置已初始化
	if GlobalConfig == nil {
		err := Init()
		require.NoError(t, err)
	}

	config := GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, GlobalConfig, config)
}

func TestConfigDefaults(t *testing.T) {
	// 清理全局配置
	GlobalConfig = nil

	// 初始化配置
	err := Init()
	require.NoError(t, err)
	require.NotNil(t, GlobalConfig)

	// 测试默认值
	assert.Equal(t, "localhost", GlobalConfig.Server.Host)
	assert.Equal(t, 8080, GlobalConfig.Server.Port)
	assert.Equal(t, "ws://localhost:8080/ws", GlobalConfig.Server.URL)

	assert.Equal(t, "", GlobalConfig.Agent.ID)
	assert.Equal(t, "assistant-agent", GlobalConfig.Agent.Name)
	assert.Equal(t, "1.0.0", GlobalConfig.Agent.Version)
	assert.Equal(t, 30, GlobalConfig.Agent.Heartbeat)
	assert.Equal(t, 3, GlobalConfig.Agent.MaxRetries)
	assert.Equal(t, 5, GlobalConfig.Agent.RetryDelay)
	assert.False(t, GlobalConfig.Agent.ContainerMode)

	assert.Equal(t, "info", GlobalConfig.Logging.Level)
	assert.Equal(t, "json", GlobalConfig.Logging.Format)
	assert.Equal(t, "assistant_agent.log", GlobalConfig.Logging.File)

	assert.Equal(t, "", GlobalConfig.Security.Token)
	assert.Equal(t, "", GlobalConfig.Security.CertFile)
	assert.Equal(t, "", GlobalConfig.Security.KeyFile)
	assert.True(t, GlobalConfig.Security.VerifySSL)
}

func TestSystemDirectories(t *testing.T) {
	// 清理全局配置
	GlobalConfig = nil

	// 初始化配置
	err := Init()
	require.NoError(t, err)
	require.NotNil(t, GlobalConfig)

	// 测试系统目录
	assert.NotEmpty(t, GlobalConfig.Agent.TempDir)
	assert.NotEmpty(t, GlobalConfig.Agent.LogDir)
	assert.NotEmpty(t, GlobalConfig.Agent.WorkDir)
	assert.NotEmpty(t, GlobalConfig.Agent.DataDir)

	// 验证目录路径符合系统标准
	switch runtime.GOOS {
	case "windows":
		// Windows 应该使用系统临时目录
		assert.Contains(t, GlobalConfig.Agent.TempDir, "Temp")
		// 其他目录应该在 ProgramData 或 AppData 下
		assert.True(t, 
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, os.Getenv("PROGRAMDATA")) ||
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, os.Getenv("APPDATA")) ||
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, filepath.Join(os.Getenv("USERPROFILE"), "AppData")),
		)
	case "linux":
		// Linux 应该使用 /tmp 作为临时目录
		assert.Equal(t, "/tmp", GlobalConfig.Agent.TempDir)
		// 其他目录应该在 /var 下或用户目录下
		assert.True(t, 
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, "/var/log") ||
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, filepath.Join(os.Getenv("HOME"), ".local")),
		)
	case "darwin":
		// macOS 应该使用 /tmp 作为临时目录
		assert.Equal(t, "/tmp", GlobalConfig.Agent.TempDir)
		// 其他目录应该在 /var 下或用户目录下
		assert.True(t, 
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, "/var/log") ||
			filepath.HasPrefix(GlobalConfig.Agent.LogDir, filepath.Join(os.Getenv("HOME"), "Library")),
		)
	}
}

func TestConfigEnvironmentVariables(t *testing.T) {
	// 清理全局配置
	GlobalConfig = nil

	// 设置环境变量
	os.Setenv("ASSISTANT_AGENT_SERVER_HOST", "test-host")
	os.Setenv("ASSISTANT_AGENT_SERVER_PORT", "9090")
	os.Setenv("ASSISTANT_AGENT_AGENT_NAME", "test-agent")
	os.Setenv("ASSISTANT_AGENT_LOGGING_LEVEL", "debug")

	// 初始化配置
	err := Init()
	require.NoError(t, err)
	require.NotNil(t, GlobalConfig)

	// 验证环境变量覆盖了默认值
	assert.Equal(t, "test-host", GlobalConfig.Server.Host)
	assert.Equal(t, 9090, GlobalConfig.Server.Port)
	assert.Equal(t, "test-agent", GlobalConfig.Agent.Name)
	assert.Equal(t, "debug", GlobalConfig.Logging.Level)

	// 清理环境变量
	os.Unsetenv("ASSISTANT_AGENT_SERVER_HOST")
	os.Unsetenv("ASSISTANT_AGENT_SERVER_PORT")
	os.Unsetenv("ASSISTANT_AGENT_AGENT_NAME")
	os.Unsetenv("ASSISTANT_AGENT_LOGGING_LEVEL")

	// 重新初始化以清理状态
	GlobalConfig = nil
	err = Init()
	require.NoError(t, err)
}

func TestConfigFile(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFilePath := filepath.Join(tempDir, "config.yaml")
	
	configContent := `
server:
  host: "file-host"
  port: 7070
agent:
  name: "file-agent"
logging:
  level: "warn"
`
	
	err := os.WriteFile(configFilePath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 保存原始工作目录
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// 切换到临时目录
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// 清理全局配置
	GlobalConfig = nil

	// 初始化配置
	err = Init()
	require.NoError(t, err)
	require.NotNil(t, GlobalConfig)

	// 验证配置文件中的值
	assert.Equal(t, "file-host", GlobalConfig.Server.Host)
	assert.Equal(t, 7070, GlobalConfig.Server.Port)
	assert.Equal(t, "file-agent", GlobalConfig.Agent.Name)
	assert.Equal(t, "warn", GlobalConfig.Logging.Level)
}

func TestCreateDirectories(t *testing.T) {
	// 清理全局配置
	GlobalConfig = nil

	// 初始化配置
	err := Init()
	require.NoError(t, err)
	require.NotNil(t, GlobalConfig)

	// 验证目录已创建
	assert.DirExists(t, GlobalConfig.Agent.WorkDir)
	assert.DirExists(t, GlobalConfig.Agent.TempDir)
	assert.DirExists(t, GlobalConfig.Agent.LogDir)
	assert.DirExists(t, GlobalConfig.Agent.DataDir)
}

func TestCanWrite(t *testing.T) {
	// 测试可写目录
	tempDir := t.TempDir()
	assert.True(t, canWrite(tempDir))

	// 测试不可写目录（如果存在）
	if runtime.GOOS != "windows" {
		assert.False(t, canWrite("/root"))
	}
} 