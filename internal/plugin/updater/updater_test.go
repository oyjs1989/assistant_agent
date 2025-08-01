package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"assistant_agent/internal/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger 模拟日志器
type MockLogger struct{}

func (l *MockLogger) Debug(args ...interface{})                 {}
func (l *MockLogger) Info(args ...interface{})                  {}
func (l *MockLogger) Warn(args ...interface{})                  {}
func (l *MockLogger) Error(args ...interface{})                 {}
func (l *MockLogger) Debugf(format string, args ...interface{}) {}
func (l *MockLogger) Infof(format string, args ...interface{})  {}
func (l *MockLogger) Warnf(format string, args ...interface{})  {}
func (l *MockLogger) Errorf(format string, args ...interface{}) {}

// MockAgent 模拟 Agent 接口
type MockAgent struct{}

func (a *MockAgent) GetSystemInfo() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *MockAgent) ExecuteCommand(command string, args []string, timeout time.Duration) (string, error) {
	return "", nil
}

func (a *MockAgent) ReadFile(path string) ([]byte, error) {
	return []byte{}, nil
}

func (a *MockAgent) WriteFile(path string, data []byte) error {
	return nil
}

func (a *MockAgent) FileExists(path string) bool {
	return false
}

func (a *MockAgent) GetConfig(key string) interface{} {
	return nil
}

func (a *MockAgent) SetConfig(key string, value interface{}) error {
	return nil
}

func (a *MockAgent) GetStatus() map[string]interface{} {
	return map[string]interface{}{}
}

func (a *MockAgent) SetStatus(key string, value interface{}) error {
	return nil
}

func (a *MockAgent) NotifyEvent(eventType string, data map[string]interface{}) error {
	return nil
}

func TestNewUpdaterPlugin(t *testing.T) {
	// 测试创建更新插件
	updaterPlugin := NewUpdaterPlugin()
	require.NotNil(t, updaterPlugin)
	assert.Equal(t, "stopped", updaterPlugin.status.Status)
}

func TestUpdaterPluginInfo(t *testing.T) {
	// 测试插件信息
	updaterPlugin := NewUpdaterPlugin()
	info := updaterPlugin.Info()

	assert.Equal(t, "updater", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "Automatic update plugin for assistant agent", info.Description)
	assert.Contains(t, info.Tags, "updater")
}

func TestUpdaterPluginInit(t *testing.T) {
	// 测试插件初始化
	updaterPlugin := NewUpdaterPlugin()
	ctx := &plugin.PluginContext{
		Agent:  &MockAgent{},
		Logger: &MockLogger{},
	}

	err := updaterPlugin.Init(ctx)
	require.NoError(t, err)
	assert.Equal(t, "initialized", updaterPlugin.status.Status)
}

func TestUpdaterPluginStartStop(t *testing.T) {
	// 测试插件启动和停止
	updaterPlugin := NewUpdaterPlugin()
	ctx := &plugin.PluginContext{
		Agent:  &MockAgent{},
		Logger: &MockLogger{},
	}

	// 初始化
	err := updaterPlugin.Init(ctx)
	require.NoError(t, err)

	// 启动
	err = updaterPlugin.Start()
	require.NoError(t, err)
	assert.Equal(t, "running", updaterPlugin.status.Status)

	// 停止
	err = updaterPlugin.Stop()
	require.NoError(t, err)
	assert.Equal(t, "stopped", updaterPlugin.status.Status)
}

func TestUpdaterPluginHandleCommand(t *testing.T) {
	// 测试命令处理
	updaterPlugin := NewUpdaterPlugin()
	ctx := &plugin.PluginContext{
		Agent:  &MockAgent{},
		Logger: &MockLogger{},
	}

	// 初始化
	err := updaterPlugin.Init(ctx)
	require.NoError(t, err)

	// 测试获取版本命令
	result, err := updaterPlugin.HandleCommand("get_version", map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// 测试获取状态命令
	result, err = updaterPlugin.HandleCommand("get_status", map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// 测试检查更新命令
	result, err = updaterPlugin.HandleCommand("check_update", map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// 测试未知命令
	_, err = updaterPlugin.HandleCommand("unknown_command", map[string]interface{}{})
	assert.Error(t, err)
}

func TestUpdaterPluginCompareVersions(t *testing.T) {
	// 测试版本比较
	updaterPlugin := NewUpdaterPlugin()

	tests := []struct {
		version1 string
		version2 string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
	}

	for _, test := range tests {
		result := updaterPlugin.compareVersions(test.version1, test.version2)
		assert.Equal(t, test.expected, result,
			"compareVersions(%s, %s) = %d, expected %d",
			test.version1, test.version2, result, test.expected)
	}
}

func TestUpdaterPluginCopyFile(t *testing.T) {
	// 测试文件复制功能
	// 创建临时文件
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "destination.txt")

	// 写入源文件
	err := os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 复制文件
	err = copyFile(srcFile, dstFile)
	require.NoError(t, err)

	// 验证目标文件存在
	_, err = os.Stat(dstFile)
	require.NoError(t, err)

	// 验证内容
	content, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestUpdaterPluginFactory(t *testing.T) {
	// 测试插件工厂
	factory := NewFactory()
	assert.Equal(t, "updater", factory.GetPluginType())

	// 创建插件
	updaterPlugin, err := factory.CreatePlugin(map[string]interface{}{
		"update_url": "https://test.com/updates",
	})
	require.NoError(t, err)
	assert.NotNil(t, updaterPlugin)

	// 验证插件类型
	info := updaterPlugin.Info()
	assert.Equal(t, "updater", info.Name)
}

func TestUpdateInfo(t *testing.T) {
	// 测试 UpdateInfo 结构
	updateInfo := &UpdateInfo{
		Version:     "1.0.1",
		URL:         "https://example.com/update",
		Checksum:    "abc123",
		ReleaseDate: time.Now(),
		Changelog:   "Test update",
		Size:        1024,
	}

	assert.Equal(t, "1.0.1", updateInfo.Version)
	assert.Equal(t, "https://example.com/update", updateInfo.URL)
	assert.Equal(t, "abc123", updateInfo.Checksum)
	assert.Equal(t, "Test update", updateInfo.Changelog)
	assert.Equal(t, int64(1024), updateInfo.Size)
}
