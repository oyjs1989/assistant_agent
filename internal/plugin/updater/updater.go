package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"assistant_agent/internal/plugin"
)

// UpdateInfo 更新信息
type UpdateInfo struct {
	Version     string    `json:"version"`
	URL         string    `json:"url"`
	Checksum    string    `json:"checksum"`
	ReleaseDate time.Time `json:"release_date"`
	Changelog   string    `json:"changelog"`
	Size        int64     `json:"size"`
}

// UpdaterPlugin 自动更新插件
type UpdaterPlugin struct {
	ctx            *plugin.PluginContext
	config         map[string]interface{}
	status         *plugin.PluginStatus
	currentVersion string
	updateURL      string
	downloadDir    string
	mu             sync.RWMutex
	stopChan       chan struct{}
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	CheckOnly   bool `json:"check_only"`
	AutoInstall bool `json:"auto_install"`
}

// NewUpdaterPlugin 创建自动更新插件
func NewUpdaterPlugin() *UpdaterPlugin {
	return &UpdaterPlugin{
		config:   make(map[string]interface{}),
		stopChan: make(chan struct{}),
		status: &plugin.PluginStatus{
			Status: "stopped",
			Metrics: map[string]interface{}{
				"total_checks":       0,
				"available_updates":  0,
				"successful_updates": 0,
				"failed_updates":     0,
			},
		},
	}
}

// Info 返回插件信息
func (p *UpdaterPlugin) Info() *plugin.PluginInfo {
	return &plugin.PluginInfo{
		Name:        "updater",
		Version:     "1.0.0",
		Description: "Automatic update plugin for assistant agent",
		Author:      "Assistant Agent Team",
		License:     "MIT",
		Homepage:    "https://github.com/assistant-agent/plugins",
		Tags:        []string{"updater", "update", "version"},
		Config: map[string]string{
			"update_url":     "https://api.example.com/updates",
			"check_interval": "3600",
			"auto_update":    "false",
			"download_dir":   "./downloads",
		},
	}
}

// Init 初始化插件
func (p *UpdaterPlugin) Init(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.status.Status = "initialized"

	// 设置默认配置
	p.setDefaultConfig()

	// 获取当前版本
	p.currentVersion = p.getCurrentVersion()

	// 创建下载目录
	downloadDir := p.getDownloadDir()
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create download directory: %v", err)
	}
	p.downloadDir = downloadDir

	p.ctx.Logger.Info("Updater plugin initialized")
	return nil
}

// Start 启动插件
func (p *UpdaterPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status.Status == "running" {
		return nil
	}

	p.status.Status = "running"
	p.status.StartTime = time.Now()
	p.status.LastUpdated = time.Now()

	p.ctx.Logger.Info("Updater plugin started")
	return nil
}

// Stop 停止插件
func (p *UpdaterPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status.Status == "stopped" {
		return nil
	}

	p.status.Status = "stopped"
	p.status.StopTime = time.Now()
	p.status.LastUpdated = time.Now()

	close(p.stopChan)

	p.ctx.Logger.Info("Updater plugin stopped")
	return nil
}

// HandleCommand 处理命令
func (p *UpdaterPlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	p.ctx.Logger.Debugf("Handling command: %s", command)

	switch command {
	case "check_update":
		return p.handleCheckUpdate(args)
	case "download_update":
		return p.handleDownloadUpdate(args)
	case "install_update":
		return p.handleInstallUpdate(args)
	case "get_status":
		return p.handleGetStatus(args)
	case "get_version":
		return p.handleGetVersion(args)
	default:
		return nil, fmt.Errorf("unknown command: %s", command)
	}
}

// HandleEvent 处理事件
func (p *UpdaterPlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	p.ctx.Logger.Debugf("Handling event: %s", eventType)

	switch eventType {
	case "update_available":
		return p.handleUpdateAvailable(data)
	case "update_completed":
		return p.handleUpdateCompleted(data)
	case "update_failed":
		return p.handleUpdateFailed(data)
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}

// Status 返回插件状态
func (p *UpdaterPlugin) Status() *plugin.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.status
}

// Health 健康检查
func (p *UpdaterPlugin) Health() error {
	if p.status.Status == "running" {
		return nil
	}
	return fmt.Errorf("plugin is not running")
}

// GetConfig 获取配置
func (p *UpdaterPlugin) GetConfig() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.config
}

// SetConfig 设置配置
func (p *UpdaterPlugin) SetConfig(config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// handleCheckUpdate 处理检查更新命令
func (p *UpdaterPlugin) handleCheckUpdate(args map[string]interface{}) (interface{}, error) {
	p.ctx.Logger.Info("Checking for updates...")

	available, updateInfo, err := p.isUpdateAvailable()
	if err != nil {
		p.updateMetrics("failed_checks", 1)
		return nil, fmt.Errorf("failed to check update: %v", err)
	}

	p.updateMetrics("total_checks", 1)

	if available && updateInfo != nil {
		p.updateMetrics("available_updates", 1)
		return map[string]interface{}{
			"available": true,
			"update":    updateInfo,
		}, nil
	}

	return map[string]interface{}{
		"available": false,
		"message":   "No updates available",
	}, nil
}

// handleDownloadUpdate 处理下载更新命令
func (p *UpdaterPlugin) handleDownloadUpdate(args map[string]interface{}) (interface{}, error) {
	updateInfo, ok := args["update"].(*UpdateInfo)
	if !ok {
		return nil, fmt.Errorf("invalid update info")
	}

	p.ctx.Logger.Infof("Downloading update version %s", updateInfo.Version)

	filepath, err := p.downloadUpdate(updateInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to download update: %v", err)
	}

	return map[string]interface{}{
		"filepath": filepath,
		"size":     updateInfo.Size,
	}, nil
}

// handleInstallUpdate 处理安装更新命令
func (p *UpdaterPlugin) handleInstallUpdate(args map[string]interface{}) (interface{}, error) {
	filepath, ok := args["filepath"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid filepath")
	}

	p.ctx.Logger.Info("Installing update...")

	err := p.installUpdate(filepath)
	if err != nil {
		p.updateMetrics("failed_updates", 1)
		return nil, fmt.Errorf("failed to install update: %v", err)
	}

	p.updateMetrics("successful_updates", 1)

	return map[string]interface{}{
		"status":  "success",
		"message": "Update installed successfully",
	}, nil
}

// handleGetStatus 处理获取状态命令
func (p *UpdaterPlugin) handleGetStatus(args map[string]interface{}) (interface{}, error) {
	return p.Status(), nil
}

// handleGetVersion 处理获取版本命令
func (p *UpdaterPlugin) handleGetVersion(args map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"current_version": p.getCurrentVersion(),
		"update_url":      p.updateURL,
	}, nil
}

// checkUpdate 检查更新
func (p *UpdaterPlugin) checkUpdate() (*UpdateInfo, error) {
	// 这里实现检查更新的逻辑
	// 可以调用远程 API 获取最新版本信息
	p.ctx.Logger.Debug("Checking for updates...")

	// 模拟检查更新
	// 在实际实现中，这里应该调用远程 API
	return nil, nil
}

// isUpdateAvailable 检查是否有可用更新
func (p *UpdaterPlugin) isUpdateAvailable() (bool, *UpdateInfo, error) {
	update, err := p.checkUpdate()
	if err != nil {
		return false, nil, err
	}

	if update == nil {
		return false, nil, nil
	}

	// 比较版本号
	return p.compareVersions(update.Version, p.currentVersion) > 0, update, nil
}

// compareVersions 比较版本号
func (p *UpdaterPlugin) compareVersions(v1, v2 string) int {
	// 简单的版本号比较
	// 在实际实现中，应该使用更复杂的版本号比较逻辑
	if v1 == v2 {
		return 0
	}
	if v1 > v2 {
		return 1
	}
	return -1
}

// downloadUpdate 下载更新
func (p *UpdaterPlugin) downloadUpdate(update *UpdateInfo) (string, error) {
	p.ctx.Logger.Infof("Downloading update version %s", update.Version)

	// 创建下载文件路径
	filename := fmt.Sprintf("assistant_agent_%s_%s_%s", update.Version, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	filepath := filepath.Join(p.downloadDir, filename)

	// 下载文件
	resp, err := http.Get(update.URL)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// 写入文件
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	p.ctx.Logger.Infof("Update downloaded to: %s", filepath)
	return filepath, nil
}

// installUpdate 安装更新
func (p *UpdaterPlugin) installUpdate(filepath string) error {
	p.ctx.Logger.Info("Installing update...")

	// 获取当前可执行文件路径
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// 创建备份
	backupPath := currentExe + ".backup"
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// 复制新文件
	if err := copyFile(filepath, currentExe); err != nil {
		// 恢复备份
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to install update: %v", err)
	}

	// 设置执行权限
	if err := os.Chmod(currentExe, 0755); err != nil {
		p.ctx.Logger.Warnf("Failed to set executable permissions: %v", err)
	}

	p.ctx.Logger.Info("Update installed successfully")
	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// getCurrentVersion 获取当前版本
func (p *UpdaterPlugin) getCurrentVersion() string {
	// 这里应该从配置或编译信息中获取版本
	return "1.0.0"
}

// getDownloadDir 获取下载目录
func (p *UpdaterPlugin) getDownloadDir() string {
	if dir, ok := p.config["download_dir"].(string); ok && dir != "" {
		return dir
	}
	return "./downloads"
}

// setDefaultConfig 设置默认配置
func (p *UpdaterPlugin) setDefaultConfig() {
	defaults := map[string]interface{}{
		"update_url":     "https://api.example.com/updates",
		"check_interval": 3600,
		"auto_update":    false,
		"download_dir":   "./downloads",
	}

	for key, value := range defaults {
		if _, exists := p.config[key]; !exists {
			p.config[key] = value
		}
	}
}

// updateMetrics 更新指标
func (p *UpdaterPlugin) updateMetrics(key string, increment int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if current, ok := p.status.Metrics[key].(int); ok {
		p.status.Metrics[key] = current + increment
	}
}

// handleUpdateAvailable 处理更新可用事件
func (p *UpdaterPlugin) handleUpdateAvailable(data map[string]interface{}) error {
	p.ctx.Logger.Info("Update available event received")
	return nil
}

// handleUpdateCompleted 处理更新完成事件
func (p *UpdaterPlugin) handleUpdateCompleted(data map[string]interface{}) error {
	p.ctx.Logger.Info("Update completed event received")
	return nil
}

// handleUpdateFailed 处理更新失败事件
func (p *UpdaterPlugin) handleUpdateFailed(data map[string]interface{}) error {
	p.ctx.Logger.Error("Update failed event received")
	return nil
}
