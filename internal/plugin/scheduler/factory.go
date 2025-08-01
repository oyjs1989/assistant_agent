package scheduler

import (
	"assistant_agent/internal/plugin"
)

// SchedulerPluginFactory 定时任务调度器插件工厂
type SchedulerPluginFactory struct{}

func (f *SchedulerPluginFactory) CreatePlugin(config map[string]interface{}) (plugin.Plugin, error) {
	return NewSchedulerPlugin(), nil
}

func (f *SchedulerPluginFactory) GetPluginType() string {
	return "scheduler"
}

// NewFactory 创建调度器插件工厂
func NewFactory() plugin.PluginFactory {
	return &SchedulerPluginFactory{}
} 