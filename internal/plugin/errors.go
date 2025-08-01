package plugin

import "errors"

// 插件系统错误定义
var (
	ErrPluginNotFound        = errors.New("plugin not found")
	ErrPluginAlreadyExists   = errors.New("plugin already exists")
	ErrPluginFactoryNotFound = errors.New("plugin factory not found")
	ErrInvalidPluginInfo     = errors.New("invalid plugin info")
	ErrPluginNotStarted      = errors.New("plugin not started")
	ErrPluginAlreadyStarted  = errors.New("plugin already started")
	ErrPluginInitFailed      = errors.New("plugin initialization failed")
	ErrPluginStartFailed     = errors.New("plugin start failed")
	ErrPluginStopFailed      = errors.New("plugin stop failed")
	ErrInvalidCommand        = errors.New("invalid command")
	ErrInvalidEvent          = errors.New("invalid event")
	ErrPluginConfigNotFound  = errors.New("plugin config not found")
	ErrPluginConfigInvalid   = errors.New("plugin config invalid")
) 