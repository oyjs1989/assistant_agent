package filetransfer

import (
	"assistant_agent/internal/plugin"
)

// FileTransferPluginFactory 文件传输插件工厂
type FileTransferPluginFactory struct{}

func (f *FileTransferPluginFactory) CreatePlugin(config map[string]interface{}) (plugin.Plugin, error) {
	return NewFileTransferPlugin(), nil
}

func (f *FileTransferPluginFactory) GetPluginType() string {
	return "filetransfer"
}

// NewFactory 创建文件传输插件工厂
func NewFactory() plugin.PluginFactory {
	return &FileTransferPluginFactory{}
}
