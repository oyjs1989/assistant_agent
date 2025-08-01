package state

import (
	"os"
	"path/filepath"
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

func TestNewManager(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// 测试创建状态管理器
	manager, err := NewManager(dataDir)
	require.NoError(t, err)
	assert.NotNil(t, manager)

	// 验证目录是否创建
	assert.DirExists(t, dataDir)
}

func TestManagerStartStop(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 测试启动
	err = manager.Start()
	require.NoError(t, err)

	// 测试停止
	manager.Stop()
}

func TestManagerGetStatus(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 获取状态
	status := manager.GetStatus()
	assert.NotNil(t, status)
	assert.Equal(t, "", status.AgentID) // 默认为空
	assert.Equal(t, "", status.Version) // 默认为空
	assert.Equal(t, "stopped", status.Status)
}

func TestManagerUpdateSystemInfo(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 创建模拟系统信息
	sysInfo := map[string]interface{}{
		"hostname":     "test-host",
		"os":           "test-os",
		"architecture": "test-arch",
		"cpu_info": map[string]interface{}{
			"cores": 4,
			"usage": 25.5,
		},
		"memory_info": map[string]interface{}{
			"total":     8192,
			"used":      4096,
			"available": 4096,
			"usage":     50.0,
		},
	}

	// 更新系统信息
	manager.UpdateSystemInfo(sysInfo)

	// 验证更新
	status := manager.GetStatus()
	assert.NotNil(t, status.SystemInfo)

	// 验证系统信息
	systemInfo := status.SystemInfo
	assert.Equal(t, "test-host", systemInfo["hostname"])
	assert.Equal(t, "test-os", systemInfo["os"])
	assert.Equal(t, "test-arch", systemInfo["architecture"])

	// 验证CPU信息
	cpuInfo := systemInfo["cpu_info"].(map[string]interface{})
	assert.Equal(t, 4, cpuInfo["cores"])
	assert.Equal(t, 25.5, cpuInfo["usage"])
}

func TestManagerUpdateTaskCount(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 更新任务计数
	manager.UpdateTaskCount(5, 10)

	// 验证更新
	status := manager.GetStatus()
	assert.Equal(t, 5, status.RunningTasks)
	assert.Equal(t, 10, status.TotalTasks)
}

func TestManagerUpdateHeartbeat(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 更新心跳
	manager.UpdateHeartbeat()

	// 验证更新
	status := manager.GetStatus()
	assert.NotZero(t, status.LastHeartbeat)
}

func TestManagerSetAgentID(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 设置 Agent ID
	agentID := "test-agent-123"
	manager.SetAgentID(agentID)

	// 验证设置
	status := manager.GetStatus()
	assert.Equal(t, agentID, status.AgentID)
}

func TestManagerSetVersion(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 设置版本
	version := "2.0.0"
	manager.SetVersion(version)

	// 验证设置
	status := manager.GetStatus()
	assert.Equal(t, version, status.Version)
}

func TestManagerSaveLoadStatus(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	manager, err := NewManager(dataDir)
	require.NoError(t, err)

	// 设置一些状态数据
	manager.SetAgentID("test-agent-save")
	manager.SetVersion("3.0.0")
	manager.UpdateTaskCount(3, 7)

	// 保存状态
	err = manager.saveStatus()
	require.NoError(t, err)

	// 验证状态文件存在
	statusFile := filepath.Join(dataDir, "status.json")
	assert.FileExists(t, statusFile)

	// 创建新的管理器来加载状态
	newManager, err := NewManager(dataDir)
	require.NoError(t, err)

	// 验证状态是否正确加载
	status := newManager.GetStatus()
	assert.Equal(t, "test-agent-save", status.AgentID)
	assert.Equal(t, "3.0.0", status.Version)
	assert.Equal(t, 3, status.RunningTasks)
	assert.Equal(t, 7, status.TotalTasks)
}

func TestManagerGetStatusSummary(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 设置一些状态数据
	manager.SetAgentID("test-agent-summary")
	manager.UpdateTaskCount(2, 5)

	// 获取状态摘要
	summary := manager.GetStatusSummary()

	// 验证摘要内容
	assert.NotNil(t, summary)
	assert.Contains(t, summary, "test-agent-summary")
	assert.Contains(t, summary, "2/5")
}

func TestManagerIsHealthy(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 更新心跳为当前时间
	manager.UpdateHeartbeat()

	// 验证健康状态
	healthy := manager.IsHealthy()
	assert.True(t, healthy)

	// 手动设置心跳时间为很久以前（通过直接修改状态）
	manager.mu.Lock()
	manager.status.LastHeartbeat = time.Now().Add(-10 * time.Hour)
	manager.mu.Unlock()

	// 验证不健康状态
	healthy = manager.IsHealthy()
	assert.False(t, healthy)
}

func TestManagerGetUptime(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 启动管理器
	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 获取运行时间
	uptime := manager.GetUptime()
	assert.Greater(t, uptime, time.Duration(0))
}

func TestManagerLoadStatusFileNotFound(t *testing.T) {
	// 创建状态管理器（状态文件不存在）
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// 确保目录存在但文件不存在
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	manager, err := NewManager(dataDir)
	require.NoError(t, err)

	// 验证使用默认值
	status := manager.GetStatus()
	assert.Equal(t, "", status.AgentID) // 默认为空
	assert.Equal(t, "", status.Version) // 默认为空
}

func TestManagerLoadStatusInvalidJSON(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// 创建无效的 JSON 文件
	statusFile := filepath.Join(dataDir, "status.json")
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(statusFile, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// 应该能正常创建管理器（使用默认值）
	manager, err := NewManager(dataDir)
	require.NoError(t, err)

	// 验证使用默认值
	status := manager.GetStatus()
	assert.Equal(t, "", status.AgentID) // 默认为空
	assert.Equal(t, "", status.Version) // 默认为空
}

func TestManagerConcurrentAccess(t *testing.T) {
	// 创建状态管理器
	tempDir := t.TempDir()
	manager, err := NewManager(filepath.Join(tempDir, "data"))
	require.NoError(t, err)

	// 并发测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			manager.SetAgentID("concurrent-agent")
			manager.UpdateTaskCount(id, id*2)
			manager.UpdateHeartbeat()
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最终状态
	status := manager.GetStatus()
	assert.Equal(t, "concurrent-agent", status.AgentID)
}
