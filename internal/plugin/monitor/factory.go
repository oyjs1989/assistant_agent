package monitor

import (
	"assistant_agent/internal/plugin"
)

// MonitorPluginFactory 系统监控插件工厂
type MonitorPluginFactory struct{}

func (f *MonitorPluginFactory) CreatePlugin(config map[string]interface{}) (plugin.Plugin, error) {
	return NewMonitorPlugin(), nil
}

func (f *MonitorPluginFactory) GetPluginType() string {
	return "monitor"
}

// NewFactory 创建系统监控插件工厂
func NewFactory() plugin.PluginFactory {
	return &MonitorPluginFactory{}
}
