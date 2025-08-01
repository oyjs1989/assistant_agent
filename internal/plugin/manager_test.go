package plugin

import (
	"testing"
	"time"

	"assistant_agent/internal/config"
	"assistant_agent/internal/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgent 模拟 Agent 接口
type MockAgent struct {
	config map[string]interface{}
}

func (m *MockAgent) GetSystemInfo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"hostname":     "test-host",
		"os":           "linux",
		"arch":         "amd64",
		"cpu_count":    4,
		"memory_total": int64(8192 * 1024 * 1024),
	}, nil
}

func (m *MockAgent) ExecuteCommand(command string, args []string, timeout time.Duration) (string, error) {
	return "command executed", nil
}

func (m *MockAgent) ReadFile(path string) ([]byte, error) {
	return []byte("test content"), nil
}

func (m *MockAgent) WriteFile(path string, data []byte) error {
	return nil
}

func (m *MockAgent) FileExists(path string) bool {
	return true
}

func (m *MockAgent) GetConfig(key string) interface{} {
	return m.config[key]
}

func (m *MockAgent) SetConfig(key string, value interface{}) error {
	m.config[key] = value
	return nil
}

func (m *MockAgent) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"status": "running",
	}
}

func (m *MockAgent) SetStatus(key string, value interface{}) error {
	return nil
}

func (m *MockAgent) NotifyEvent(eventType string, data map[string]interface{}) error {
	return nil
}

// MockPlugin 模拟插件
type MockPlugin struct {
	info   *PluginInfo
	status *PluginStatus
	config map[string]interface{}
}

func (p *MockPlugin) Info() *PluginInfo {
	return p.info
}

func (p *MockPlugin) Init(ctx *PluginContext) error {
	return nil
}

func (p *MockPlugin) Start() error {
	p.status.Status = "running"
	p.status.StartTime = time.Now()
	return nil
}

func (p *MockPlugin) Stop() error {
	p.status.Status = "stopped"
	return nil
}

func (p *MockPlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"command": command,
		"args":    args,
	}, nil
}

func (p *MockPlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	return nil
}

func (p *MockPlugin) Status() *PluginStatus {
	return p.status
}

func (p *MockPlugin) Health() error {
	return nil
}

func (p *MockPlugin) GetConfig() map[string]interface{} {
	return p.config
}

func (p *MockPlugin) SetConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}

	manager := NewManager(agent, cfg)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.factories)
	assert.Equal(t, agent, manager.agent)
	assert.Equal(t, cfg, manager.config)
}

func TestManagerRegister(t *testing.T) {
	// 初始化配置
	config.Init()
	
	// 初始化 logger
	logger.Init()
	
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	err := manager.Register(plugin)
	require.NoError(t, err)

	// 测试重复注册
	err = manager.Register(plugin)
	assert.Error(t, err)
	assert.Equal(t, ErrPluginAlreadyExists, err)
}

func TestManagerUnregister(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	// 先注册插件
	err := manager.Register(plugin)
	require.NoError(t, err)

	// 测试注销
	err = manager.Unregister("test-plugin")
	require.NoError(t, err)

	// 测试注销不存在的插件
	err = manager.Unregister("non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrPluginNotFound, err)
}

func TestManagerGetPlugin(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	// 注册插件
	err := manager.Register(plugin)
	require.NoError(t, err)

	// 获取插件
	retrievedPlugin, exists := manager.GetPlugin("test-plugin")
	assert.True(t, exists)
	assert.Equal(t, plugin, retrievedPlugin)

	// 获取不存在的插件
	_, exists = manager.GetPlugin("non-existent")
	assert.False(t, exists)
}

func TestManagerListPlugins(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin1 := &MockPlugin{
		info: &PluginInfo{
			Name:    "plugin1",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	plugin2 := &MockPlugin{
		info: &PluginInfo{
			Name:    "plugin2",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	// 注册插件
	err := manager.Register(plugin1)
	require.NoError(t, err)

	err = manager.Register(plugin2)
	require.NoError(t, err)

	// 列出插件
	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 2)
}

func TestManagerStartStopPlugin(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	// 注册插件
	err := manager.Register(plugin)
	require.NoError(t, err)

	// 启动插件
	err = manager.StartPlugin("test-plugin")
	require.NoError(t, err)

	// 检查状态
	status, err := manager.GetPluginStatus("test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "running", status.Status)

	// 停止插件
	err = manager.StopPlugin("test-plugin")
	require.NoError(t, err)

	// 检查状态
	status, err = manager.GetPluginStatus("test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "stopped", status.Status)
}

func TestManagerSendCommand(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "running",
		},
		config: make(map[string]interface{}),
	}

	// 注册并启动插件
	err := manager.Register(plugin)
	require.NoError(t, err)

	// 发送命令
	result, err := manager.SendCommand("test-plugin", "test-command", map[string]interface{}{
		"arg1": "value1",
	})
	require.NoError(t, err)

	// 验证结果
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test-command", resultMap["command"])
}

func TestManagerSendEvent(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "running",
		},
		config: make(map[string]interface{}),
	}

	// 注册并启动插件
	err := manager.Register(plugin)
	require.NoError(t, err)

	// 发送事件
	err = manager.SendEvent("test-plugin", "test-event", map[string]interface{}{
		"data": "value",
	})
	require.NoError(t, err)
}

func TestManagerStartAllStopAll(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin1 := &MockPlugin{
		info: &PluginInfo{
			Name:    "plugin1",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	plugin2 := &MockPlugin{
		info: &PluginInfo{
			Name:    "plugin2",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	// 注册插件
	err := manager.Register(plugin1)
	require.NoError(t, err)

	err = manager.Register(plugin2)
	require.NoError(t, err)

	// 启动所有插件
	err = manager.StartAll()
	require.NoError(t, err)

	// 检查状态
	status1, err := manager.GetPluginStatus("plugin1")
	require.NoError(t, err)
	assert.Equal(t, "running", status1.Status)

	status2, err := manager.GetPluginStatus("plugin2")
	require.NoError(t, err)
	assert.Equal(t, "running", status2.Status)

	// 停止所有插件
	err = manager.StopAll()
	require.NoError(t, err)

	// 检查状态
	status1, err = manager.GetPluginStatus("plugin1")
	require.NoError(t, err)
	assert.Equal(t, "stopped", status1.Status)

	status2, err = manager.GetPluginStatus("plugin2")
	require.NoError(t, err)
	assert.Equal(t, "stopped", status2.Status)
}

func TestManagerGetAllPluginStatus(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "running",
		},
		config: make(map[string]interface{}),
	}

	// 注册插件
	err := manager.Register(plugin)
	require.NoError(t, err)

	// 获取所有插件状态
	statuses := manager.GetAllPluginStatus()
	assert.Len(t, statuses, 1)
	assert.Equal(t, "running", statuses["test-plugin"].Status)
}

func TestManagerErrorCases(t *testing.T) {
	cfg := &config.Config{}
	agent := &MockAgent{config: make(map[string]interface{})}
	manager := NewManager(agent, cfg)

	// 测试启动不存在的插件
	err := manager.StartPlugin("non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrPluginNotFound, err)

	// 测试停止不存在的插件
	err = manager.StopPlugin("non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrPluginNotFound, err)

	// 测试获取不存在的插件状态
	_, err = manager.GetPluginStatus("non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrPluginNotFound, err)

	// 测试向不存在的插件发送命令
	_, err = manager.SendCommand("non-existent", "test", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrPluginNotFound, err)

	// 测试向未启动的插件发送命令
	plugin := &MockPlugin{
		info: &PluginInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		status: &PluginStatus{
			Status: "stopped",
		},
		config: make(map[string]interface{}),
	}

	err = manager.Register(plugin)
	require.NoError(t, err)

	_, err = manager.SendCommand("test-plugin", "test", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrPluginNotStarted, err)
}
