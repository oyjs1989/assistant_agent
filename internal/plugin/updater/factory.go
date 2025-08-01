package updater

import (
	"assistant_agent/internal/plugin"
)

// UpdaterPluginFactory 自动更新插件工厂
type UpdaterPluginFactory struct{}

func (f *UpdaterPluginFactory) CreatePlugin(config map[string]interface{}) (plugin.Plugin, error) {
	updaterPlugin := NewUpdaterPlugin()

	// 应用配置
	if config != nil {
		updaterPlugin.SetConfig(config)
	}

	return updaterPlugin, nil
}

func (f *UpdaterPluginFactory) GetPluginType() string {
	return "updater"
}

// NewFactory 创建自动更新插件工厂
func NewFactory() plugin.PluginFactory {
	return &UpdaterPluginFactory{}
}
