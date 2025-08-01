package plugin

import (
	"time"
)

// PluginInfo 插件信息
type PluginInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	License     string            `json:"license"`
	Homepage    string            `json:"homepage"`
	Tags        []string          `json:"tags"`
	Config      map[string]string `json:"config"`
}

// PluginStatus 插件状态
type PluginStatus struct {
	Status      string                 `json:"status"`
	StartTime   time.Time              `json:"start_time"`
	StopTime    time.Time              `json:"stop_time"`
	Metrics     map[string]interface{} `json:"metrics"`
	LastError   string                 `json:"last_error,omitempty"`
	LastUpdated time.Time              `json:"last_updated"`
}

// PluginContext 插件上下文
type PluginContext struct {
	Agent  AgentInterface
	Logger Logger
}

// Logger 日志接口
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// AgentInterface Agent 接口
type AgentInterface interface {
	GetSystemInfo() (map[string]interface{}, error)
	ExecuteCommand(command string, args []string, timeout time.Duration) (string, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	FileExists(path string) bool
	GetConfig(key string) interface{}
	SetConfig(key string, value interface{}) error
	GetStatus() map[string]interface{}
	SetStatus(key string, value interface{}) error
	NotifyEvent(eventType string, data map[string]interface{}) error
}

// Plugin 插件接口
type Plugin interface {
	Info() *PluginInfo
	Init(ctx *PluginContext) error
	Start() error
	Stop() error
	HandleCommand(command string, args map[string]interface{}) (interface{}, error)
	HandleEvent(eventType string, data map[string]interface{}) error
	Status() *PluginStatus
	Health() error
	GetConfig() map[string]interface{}
	SetConfig(config map[string]interface{}) error
}

// PluginManager 插件管理器接口
type PluginManager interface {
	Register(plugin Plugin) error
	Unregister(pluginName string) error
	GetPlugin(pluginName string) (Plugin, bool)
	ListPlugins() []Plugin
	StartPlugin(pluginName string) error
	StopPlugin(pluginName string) error
	SendCommand(pluginName string, command string, args map[string]interface{}) (interface{}, error)
	SendEvent(pluginName string, eventType string, data map[string]interface{}) error
	StartAll() error
	StopAll() error
	GetAllPluginStatus() map[string]*PluginStatus
	RegisterFactory(pluginType string, factory PluginFactory)
	CreatePlugin(pluginType string, config map[string]interface{}) (Plugin, error)
}

// PluginFactory 插件工厂接口
type PluginFactory interface {
	CreatePlugin(config map[string]interface{}) (Plugin, error)
	GetPluginType() string
}

// PluginRegistry 插件注册表接口
type PluginRegistry interface {
	RegisterFactory(pluginType string, factory PluginFactory) error
	GetFactory(pluginType string) (PluginFactory, bool)
	ListFactories() []string
}
