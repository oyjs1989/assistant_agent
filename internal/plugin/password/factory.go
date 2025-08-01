package password

import (
	"assistant_agent/internal/plugin"
)

// PasswordPluginFactory 密码管理插件工厂
type PasswordPluginFactory struct{}

func (f *PasswordPluginFactory) CreatePlugin(config map[string]interface{}) (plugin.Plugin, error) {
	return NewPasswordPlugin(), nil
}

func (f *PasswordPluginFactory) GetPluginType() string {
	return "password"
}

// NewFactory 创建密码管理插件工厂
func NewFactory() plugin.PluginFactory {
	return &PasswordPluginFactory{}
}
