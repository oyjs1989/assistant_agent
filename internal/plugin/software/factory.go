package software

import (
	"assistant_agent/internal/plugin"
)

// SoftwarePluginFactory 软件管理插件工厂
type SoftwarePluginFactory struct{}

func (f *SoftwarePluginFactory) CreatePlugin(config map[string]interface{}) (plugin.Plugin, error) {
	return NewSoftwarePlugin(), nil
}

func (f *SoftwarePluginFactory) GetPluginType() string {
	return "software"
}

// NewFactory 创建软件管理插件工厂
func NewFactory() plugin.PluginFactory {
	return &SoftwarePluginFactory{}
}
