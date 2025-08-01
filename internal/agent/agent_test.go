package agent

import (
	"testing"
	"time"

	"assistant_agent/internal/config"
	"assistant_agent/internal/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 初始化配置和日志
	config.Init()
	logger.Init()
}

func TestNew(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 测试创建 Agent
	agent, err := New()
	require.NoError(t, err)
	assert.NotNil(t, agent)

	// 验证组件是否创建
	assert.NotNil(t, agent.stateMgr)
	assert.NotNil(t, agent.heartbeat)
	assert.NotNil(t, agent.wsClient)
	assert.NotNil(t, agent.pluginMgr)
}

func TestAgentStartStop(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 测试启动
	err = agent.Start()
	require.NoError(t, err)

	// 等待一小段时间让组件启动
	time.Sleep(100 * time.Millisecond)

	// 测试停止
	agent.Stop()
}

func TestAgentHandleMessage(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 测试处理不同类型的消息
	tests := []struct {
		name    string
		msgType string
		msgData interface{}
	}{
		{
			name:    "Command message",
			msgType: "command",
			msgData: map[string]interface{}{
				"id":     "test-cmd",
				"type":   "shell",
				"script": "echo 'test'",
			},
		},
		{
			name:    "Task message",
			msgType: "schedule",
			msgData: map[string]interface{}{
				"id":        "test-task",
				"name":      "Test Task",
				"cron_expr": "*/1 * * * * *",
				"command": map[string]interface{}{
					"id":     "test-cmd",
					"type":   "shell",
					"script": "echo 'test'",
				},
			},
		},
		{
			name:    "Unknown message type",
			msgType: "unknown",
			msgData: "test data",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 处理消息（应该不会崩溃）
			err := agent.handleMessage(test.msgType, test.msgData)
			assert.NoError(t, err)
		})
	}
}

func TestAgentHandleCommandMessage(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 创建命令消息数据
	commandData := map[string]interface{}{
		"id":     "test-cmd",
		"type":   "shell",
		"script": "echo 'test command'",
	}

	// 处理命令消息
	err = agent.handleMessage("command", commandData)
	assert.NoError(t, err)

	// 验证命令是否被添加到执行器
	// 注意：这里只是测试不会崩溃，实际验证需要更复杂的测试设置
}

func TestAgentHandleTaskMessage(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 创建任务消息数据
	taskData := map[string]interface{}{
		"id":        "test-task",
		"name":      "Test Task",
		"cron_expr": "*/1 * * * * *",
		"command": map[string]interface{}{
			"id":     "test-cmd",
			"type":   "shell",
			"script": "echo 'test task'",
		},
	}

	// 处理任务消息
	err = agent.handleMessage("schedule", taskData)
	assert.NoError(t, err)

	// 验证任务是否被添加到调度器
	// 注意：这里只是测试不会崩溃，实际验证需要更复杂的测试设置
}

func TestAgentHandleInvalidMessage(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 测试处理无效消息
	invalidMessages := []struct {
		msgType string
		msgData interface{}
	}{
		{"", "empty type"},
		{"command", "invalid data type"},
		{"schedule", "invalid data type"},
		{"command", map[string]interface{}{
			"invalid_field": "value",
		}},
	}

	for _, message := range invalidMessages {
		// 处理无效消息（应该不会崩溃）
		err := agent.handleMessage(message.msgType, message.msgData)
		assert.NoError(t, err)
	}
}

func TestAgentComponentIntegration(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 验证组件集成
	assert.NotNil(t, agent.stateMgr)
	assert.NotNil(t, agent.heartbeat)
	assert.NotNil(t, agent.wsClient)
	assert.NotNil(t, agent.pluginMgr)

	// 验证组件之间的依赖关系
	// 插件管理器应该已初始化
	assert.NotNil(t, agent.pluginMgr)
	// WebSocket 客户端应该配置正确
	assert.NotEmpty(t, agent.wsClient.GetURL())
}

func TestAgentLifecycle(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 启动 Agent
	err = agent.Start()
	require.NoError(t, err)

	// 验证 Agent 状态
	status := agent.stateMgr.GetStatus()
	assert.Equal(t, "running", status.Status)

	// 等待一段时间
	time.Sleep(200 * time.Millisecond)

	// 停止 Agent
	agent.Stop()

	// 验证 Agent 状态
	status = agent.stateMgr.GetStatus()
	assert.Equal(t, "stopped", status.Status)
}

func TestAgentConcurrentOperations(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 启动 Agent
	err = agent.Start()
	require.NoError(t, err)
	defer agent.Stop()

	// 并发操作
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			// 发送不同类型的消息
			agent.handleMessage("command", map[string]interface{}{
				"id":     "concurrent-cmd",
				"type":   "shell",
				"script": "echo 'concurrent test'",
			})
			done <- true
		}()
	}

	// 等待所有操作完成
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestAgentErrorHandling(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 测试错误情况下的处理
	errorScenarios := []struct {
		name    string
		msgType string
		msgData interface{}
	}{
		{
			name:    "Empty message type",
			msgType: "",
			msgData: "test",
		},
		{
			name:    "Invalid command data",
			msgType: "command",
			msgData: map[string]interface{}{
				"invalid": "data",
			},
		},
		{
			name:    "Invalid task data",
			msgType: "schedule",
			msgData: map[string]interface{}{
				"invalid": "data",
			},
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// 处理错误情况（应该不会崩溃）
			err := agent.handleMessage(scenario.msgType, scenario.msgData)
			assert.NoError(t, err)
		})
	}
}

func TestAgentSystemInfoCollection(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 启动 Agent
	err = agent.Start()
	require.NoError(t, err)
	defer agent.Stop()

	// 等待系统信息收集
	time.Sleep(200 * time.Millisecond)

	// 验证系统信息是否被收集
	status := agent.stateMgr.GetStatus()
	assert.NotNil(t, status.SystemInfo)
	// 系统信息现在是 map[string]interface{} 类型
	systemInfo := status.SystemInfo
	assert.NotNil(t, systemInfo)
}

func TestAgentHeartbeat(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 启动 Agent
	err = agent.Start()
	require.NoError(t, err)
	defer agent.Stop()

	// 等待心跳发送
	time.Sleep(200 * time.Millisecond)

	// 验证心跳是否正常
	status := agent.stateMgr.GetStatus()
	assert.NotZero(t, status.LastHeartbeat)
	assert.True(t, agent.heartbeat.IsHealthy())
}

func TestAgentUpdateCheck(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 启动 Agent
	err = agent.Start()
	require.NoError(t, err)
	defer agent.Stop()

	// 等待更新检查
	time.Sleep(200 * time.Millisecond)

	// 验证更新检查是否正常
	// 由于是插件架构，更新功能通过插件实现
	// 这里只是验证不会崩溃
	assert.NotNil(t, agent.pluginMgr)
}

func TestAgentWebSocketCommunication(t *testing.T) {
	// 初始化配置
	err := config.Init()
	require.NoError(t, err)

	// 创建 Agent
	agent, err := New()
	require.NoError(t, err)

	// 启动 Agent
	err = agent.Start()
	require.NoError(t, err)
	defer agent.Stop()

	// 等待 WebSocket 连接
	time.Sleep(200 * time.Millisecond)

	// 验证 WebSocket 客户端状态
	// 注意：在实际环境中，连接可能会失败，这是正常的
	assert.NotNil(t, agent.wsClient)
}
