package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"assistant_agent/internal/config"
	"assistant_agent/internal/logger"
)

// Manager 插件管理器实现
type Manager struct {
	factories map[string]PluginFactory
	agent     AgentInterface
	config    *config.Config
	plugins   map[string]*PluginInstance
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// PluginInstance 插件实例
type PluginInstance struct {
	Plugin     Plugin
	Context    *PluginContext
	Config     map[string]interface{}
	Status     *PluginStatus
	ConfigFile string
	mu         sync.RWMutex
}

// NewManager 创建插件管理器
func NewManager(agent AgentInterface, cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		factories: make(map[string]PluginFactory),
		agent:     agent,
		config:    cfg,
		plugins:   make(map[string]*PluginInstance),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Register 注册插件
func (m *Manager) Register(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := plugin.Info()
	if info == nil {
		return ErrInvalidPluginInfo
	}

	// 检查插件是否已存在
	if _, exists := m.plugins[info.Name]; exists {
		return ErrPluginAlreadyExists
	}

	// 创建插件实例
	instance := &PluginInstance{
		Plugin:     plugin,
		Config:     make(map[string]interface{}),
		ConfigFile: filepath.Join(m.config.Agent.DataDir, "plugins", fmt.Sprintf("%s.json", info.Name)),
		Status: &PluginStatus{
			Status:      "stopped",
			StartTime:   time.Time{},
			Metrics:     make(map[string]interface{}),
			LastUpdated: time.Now(),
		},
	}

	// 插件直接添加到管理器

	// 添加到管理器
	m.plugins[info.Name] = instance

	logger.Infof("Plugin registered: %s v%s", info.Name, info.Version)
	return nil
}

// Unregister 注销插件
func (m *Manager) Unregister(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.plugins[pluginName]
	if !exists {
		return ErrPluginNotFound
	}

	// 停止插件
	if instance.Status.Status == "running" {
		if err := instance.Plugin.Stop(); err != nil {
			logger.Warnf("Failed to stop plugin %s: %v", pluginName, err)
		}
	}

	// 从管理器移除

	// 从管理器移除
	delete(m.plugins, pluginName)

	logger.Infof("Plugin unregistered: %s", pluginName)
	return nil
}

// GetPlugin 获取插件
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.plugins[name]
	if !exists {
		return nil, false
	}
	return instance.Plugin, true
}

// ListPlugins 列出所有插件
func (m *Manager) ListPlugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]Plugin, 0, len(m.plugins))
	for _, instance := range m.plugins {
		plugins = append(plugins, instance.Plugin)
	}
	return plugins
}

// StartPlugin 启动插件
func (m *Manager) StartPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	if instance.Status.Status == "running" {
		return ErrPluginAlreadyStarted
	}

	// 加载配置
	if err := m.LoadPluginConfig(name); err != nil {
		logger.Warnf("Failed to load config for plugin %s: %v", name, err)
	}

	// 创建插件上下文
	instance.Context = &PluginContext{
		Agent:  m.agent,
		Logger: &PluginLogger{pluginName: name},
	}

	// 初始化插件
	if err := instance.Plugin.Init(instance.Context); err != nil {
		instance.Status.Status = "error"
		instance.Status.LastError = err.Error()
		return fmt.Errorf("failed to init plugin %s: %w", name, err)
	}

	// 启动插件
	if err := instance.Plugin.Start(); err != nil {
		instance.Status.Status = "error"
		instance.Status.LastError = err.Error()
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	// 更新状态
	instance.Status.Status = "running"
	instance.Status.StartTime = time.Now()
	instance.Status.LastError = ""

	logger.Infof("Plugin started: %s", name)
	return nil
}

// StopPlugin 停止插件
func (m *Manager) StopPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	if instance.Status.Status != "running" {
		return ErrPluginNotStarted
	}

	// 停止插件
	if err := instance.Plugin.Stop(); err != nil {
		instance.Status.Status = "error"
		instance.Status.LastError = err.Error()
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	// 保存配置
	if err := m.SavePluginConfig(name); err != nil {
		logger.Warnf("Failed to save config for plugin %s: %v", name, err)
	}

	// 更新状态
	instance.Status.Status = "stopped"
	instance.Status.LastError = ""

	logger.Infof("Plugin stopped: %s", name)
	return nil
}

// StartAll 启动所有插件
func (m *Manager) StartAll() error {
	m.mu.RLock()
	plugins := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		plugins = append(plugins, name)
	}
	m.mu.RUnlock()

	var errors []error
	for _, name := range plugins {
		if err := m.StartPlugin(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to start plugin %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start some plugins: %v", errors)
	}
	return nil
}

// StopAll 停止所有插件
func (m *Manager) StopAll() error {
	m.mu.RLock()
	plugins := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		plugins = append(plugins, name)
	}
	m.mu.RUnlock()

	var errors []error
	for _, name := range plugins {
		if err := m.StopPlugin(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop plugin %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some plugins: %v", errors)
	}
	return nil
}

// GetPluginStatus 获取插件状态
func (m *Manager) GetPluginStatus(name string) (*PluginStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.plugins[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	// 获取插件的最新状态
	status := instance.Plugin.Status()
	if status != nil {
		instance.Status = status
	}

	return instance.Status, nil
}

// GetAllPluginStatus 获取所有插件状态
func (m *Manager) GetAllPluginStatus() map[string]*PluginStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]*PluginStatus)
	for name, instance := range m.plugins {
		status := instance.Plugin.Status()
		if status != nil {
			instance.Status = status
		}
		statuses[name] = instance.Status
	}
	return statuses
}

// SendCommand 发送命令到插件
func (m *Manager) SendCommand(pluginName, command string, args map[string]interface{}) (interface{}, error) {
	m.mu.RLock()
	instance, exists := m.plugins[pluginName]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrPluginNotFound
	}

	if instance.Status.Status != "running" {
		return nil, ErrPluginNotStarted
	}

	return instance.Plugin.HandleCommand(command, args)
}

// SendEvent 发送事件到插件
func (m *Manager) SendEvent(pluginName, eventType string, data map[string]interface{}) error {
	m.mu.RLock()
	instance, exists := m.plugins[pluginName]
	m.mu.RUnlock()

	if !exists {
		return ErrPluginNotFound
	}

	if instance.Status.Status != "running" {
		return ErrPluginNotStarted
	}

	return instance.Plugin.HandleEvent(eventType, data)
}

// LoadPluginConfig 加载插件配置
func (m *Manager) LoadPluginConfig(name string) error {
	m.mu.RLock()
	instance, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return ErrPluginNotFound
	}

	// 确保配置目录存在
	configDir := filepath.Dir(instance.ConfigFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 读取配置文件
	data, err := os.ReadFile(instance.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认配置
			return nil
		}
		return err
	}

	// 解析配置
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	instance.mu.Lock()
	instance.Config = config
	instance.mu.Unlock()

	return nil
}

// SavePluginConfig 保存插件配置
func (m *Manager) SavePluginConfig(name string) error {
	m.mu.RLock()
	instance, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return ErrPluginNotFound
	}

	// 获取插件配置
	config := instance.Plugin.GetConfig()
	if config == nil {
		config = make(map[string]interface{})
	}

	// 确保配置目录存在
	configDir := filepath.Dir(instance.ConfigFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// 写入配置文件
	return os.WriteFile(instance.ConfigFile, data, 0644)
}

// RegisterFactory 注册插件工厂
func (m *Manager) RegisterFactory(pluginType string, factory PluginFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factories[pluginType] = factory
}

// CreatePlugin 创建插件实例
func (m *Manager) CreatePlugin(pluginType string, config map[string]interface{}) (Plugin, error) {
	m.mu.RLock()
	factory, exists := m.factories[pluginType]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("plugin factory not found: %s", pluginType)
	}
	
	return factory.CreatePlugin(config)
}

// Stop 停止插件管理器
func (m *Manager) Stop() {
	m.cancel()
	m.StopAll()
}

// PluginLogger 插件日志适配器
type PluginLogger struct {
	pluginName string
}

func (l *PluginLogger) Debug(args ...interface{}) {
	logger.Debugf("[Plugin:%s] %v", l.pluginName, args)
}

func (l *PluginLogger) Info(args ...interface{}) {
	logger.Infof("[Plugin:%s] %v", l.pluginName, args)
}

func (l *PluginLogger) Warn(args ...interface{}) {
	logger.Warnf("[Plugin:%s] %v", l.pluginName, args)
}

func (l *PluginLogger) Error(args ...interface{}) {
	logger.Errorf("[Plugin:%s] %v", l.pluginName, args)
}

func (l *PluginLogger) Debugf(format string, args ...interface{}) {
	logger.Debugf("[Plugin:%s] "+format, append([]interface{}{l.pluginName}, args...)...)
}

func (l *PluginLogger) Infof(format string, args ...interface{}) {
	logger.Infof("[Plugin:%s] "+format, append([]interface{}{l.pluginName}, args...)...)
}

func (l *PluginLogger) Warnf(format string, args ...interface{}) {
	logger.Warnf("[Plugin:%s] "+format, append([]interface{}{l.pluginName}, args...)...)
}

func (l *PluginLogger) Errorf(format string, args ...interface{}) {
	logger.Errorf("[Plugin:%s] "+format, append([]interface{}{l.pluginName}, args...)...)
}
