package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"assistant_agent/internal/logger"
)

// Status Agent 状态
type Status struct {
	AgentID       string                 `json:"agent_id"`
	Version       string                 `json:"version"`
	Status        string                 `json:"status"`
	StartTime     time.Time              `json:"start_time"`
	LastHeartbeat time.Time              `json:"last_heartbeat"`
	SystemInfo    map[string]interface{} `json:"system_info,omitempty"`
	RunningTasks  int                    `json:"running_tasks"`
	TotalTasks    int                    `json:"total_tasks"`
	Uptime        float64                `json:"uptime"`
	MemoryUsage   float64                `json:"memory_usage"`
	CPUUsage      float64                `json:"cpu_usage"`
	DiskUsage     float64                `json:"disk_usage"`
}

// Manager 状态管理器
type Manager struct {
	dataDir   string
	status    *Status
	mu        sync.RWMutex
	startTime time.Time
}

// NewManager 创建新的状态管理器
func NewManager(dataDir string) (*Manager, error) {
	// 创建数据目录
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	manager := &Manager{
		dataDir:   dataDir,
		startTime: time.Now(),
		status: &Status{
			Status:    "stopped",
			StartTime: time.Now(),
		},
	}

	// 加载保存的状态
	if err := manager.loadStatus(); err != nil {
		logger.Warnf("Failed to load status: %v", err)
	}

	return manager, nil
}

// Start 启动状态管理器
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Status = "running"
	m.status.StartTime = time.Now()
	m.status.LastHeartbeat = time.Now()

	if err := m.saveStatus(); err != nil {
		return err
	}

	logger.Info("State manager started")
	return nil
}

// Stop 停止状态管理器
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Status = "stopped"
	m.saveStatus()

	logger.Info("State manager stopped")
}

// GetStatus 获取当前状态
func (m *Manager) GetStatus() *Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 更新运行时间
	m.status.Uptime = time.Since(m.startTime).Seconds()

	return m.status
}

// UpdateSystemInfo 更新系统信息
func (m *Manager) UpdateSystemInfo(info map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.SystemInfo = info
	m.status.LastHeartbeat = time.Now()

	// 更新资源使用情况
	if cpu, ok := info["cpu_usage"].(float64); ok {
		m.status.CPUUsage = cpu
	}
	if memory, ok := info["memory_usage"].(float64); ok {
		m.status.MemoryUsage = memory
	}
	if disk, ok := info["disk_usage"].(float64); ok {
		m.status.DiskUsage = disk
	}

	m.saveStatus()
}

// UpdateTaskCount 更新任务计数
func (m *Manager) UpdateTaskCount(running, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.RunningTasks = running
	m.status.TotalTasks = total
	m.status.LastHeartbeat = time.Now()

	m.saveStatus()
}

// UpdateHeartbeat 更新心跳时间
func (m *Manager) UpdateHeartbeat() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.LastHeartbeat = time.Now()
	m.saveStatus()
}

// SetAgentID 设置 Agent ID
func (m *Manager) SetAgentID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.AgentID = id
	m.saveStatus()
}

// SetVersion 设置版本
func (m *Manager) SetVersion(version string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Version = version
	m.saveStatus()
}

// saveStatus 保存状态到文件
func (m *Manager) saveStatus() error {
	statusFile := filepath.Join(m.dataDir, "status.json")

	data, err := json.MarshalIndent(m.status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %v", err)
	}

	if err := os.WriteFile(statusFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %v", err)
	}

	return nil
}

// loadStatus 从文件加载状态
func (m *Manager) loadStatus() error {
	statusFile := filepath.Join(m.dataDir, "status.json")

	data, err := os.ReadFile(statusFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用默认状态
		}
		return fmt.Errorf("failed to read status file: %v", err)
	}

	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("failed to unmarshal status: %v", err)
	}

	m.status = &status
	return nil
}

// GetStatusSummary 获取状态摘要
func (m *Manager) GetStatusSummary() map[string]interface{} {
	status := m.GetStatus()

	return map[string]interface{}{
		"agent_id":       status.AgentID,
		"version":        status.Version,
		"status":         status.Status,
		"uptime":         status.Uptime,
		"last_heartbeat": status.LastHeartbeat,
		"running_tasks":  status.RunningTasks,
		"total_tasks":    status.TotalTasks,
		"memory_usage":   status.MemoryUsage,
		"cpu_usage":      status.CPUUsage,
		"disk_usage":     status.DiskUsage,
	}
}

// IsHealthy 检查 Agent 是否健康
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查心跳时间，如果超过5分钟没有心跳则认为不健康
	if time.Since(m.status.LastHeartbeat) > 5*time.Minute {
		return false
	}

	// 检查状态
	if m.status.Status != "running" {
		return false
	}

	return true
}

// GetUptime 获取运行时间
func (m *Manager) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// GetStartTime 获取启动时间
func (m *Manager) GetStartTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.startTime
}
