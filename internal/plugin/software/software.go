package software

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"assistant_agent/internal/plugin"
)

// SoftwarePlugin 软件安装插件
type SoftwarePlugin struct {
	ctx       *plugin.PluginContext
	config    map[string]interface{}
	status    *plugin.PluginStatus
	installed map[string]*SoftwareInfo
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// SoftwareInfo 软件信息
type SoftwareInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	InstallTime time.Time `json:"install_time"`
	Status      string    `json:"status"`       // installed, installing, failed, uninstalled
	PackageType string    `json:"package_type"` // apt, yum, brew, chocolatey, etc.
	Description string    `json:"description"`
	Size        int64     `json:"size"`
	LastUpdated time.Time `json:"last_updated"`
}

// InstallRequest 安装请求
type InstallRequest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	PackageType string            `json:"package_type"`
	Source      string            `json:"source"`
	Options     map[string]string `json:"options"`
}

// UninstallRequest 卸载请求
type UninstallRequest struct {
	Name        string            `json:"name"`
	PackageType string            `json:"package_type"`
	Options     map[string]string `json:"options"`
}

// NewSoftwarePlugin 创建软件安装插件
func NewSoftwarePlugin() *SoftwarePlugin {
	return &SoftwarePlugin{
		config:    make(map[string]interface{}),
		installed: make(map[string]*SoftwareInfo),
		stopChan:  make(chan struct{}),
		status: &plugin.PluginStatus{
			Status: "stopped",
			Metrics: map[string]interface{}{
				"installed_count": 0,
				"total_size":      0,
			},
		},
	}
}

// Info 返回插件信息
func (p *SoftwarePlugin) Info() *plugin.PluginInfo {
	return &plugin.PluginInfo{
		Name:        "software-manager",
		Version:     "1.0.0",
		Description: "Software installation and management plugin",
		Author:      "Assistant Agent Team",
		License:     "MIT",
		Homepage:    "https://github.com/assistant-agent/plugins",
		Tags:        []string{"software", "installation", "package-management"},
		Config: map[string]string{
			"package_manager": "auto",
			"install_dir":     "/usr/local",
			"backup_enabled":  "true",
		},
	}
}

// Init 初始化插件
func (p *SoftwarePlugin) Init(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.status.Status = "initialized"

	// 加载已安装软件列表
	p.loadInstalledSoftware()

	p.ctx.Logger.Info("Software plugin initialized")
	return nil
}

// Start 启动插件
func (p *SoftwarePlugin) Start() error {
	p.status.Status = "running"
	p.status.StartTime = time.Now()

	// 启动后台任务
	go p.backgroundTask()

	p.ctx.Logger.Info("Software plugin started")
	return nil
}

// Stop 停止插件
func (p *SoftwarePlugin) Stop() error {
	p.status.Status = "stopped"
	close(p.stopChan)

	// 保存已安装软件列表
	p.saveInstalledSoftware()

	p.ctx.Logger.Info("Software plugin stopped")
	return nil
}

// HandleCommand 处理命令
func (p *SoftwarePlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	switch command {
	case "install":
		return p.handleInstall(args)
	case "uninstall":
		return p.handleUninstall(args)
	case "list":
		return p.handleList(args)
	case "info":
		return p.handleInfo(args)
	case "update":
		return p.handleUpdate(args)
	case "search":
		return p.handleSearch(args)
	default:
		return nil, plugin.ErrInvalidCommand
	}
}

// HandleEvent 处理事件
func (p *SoftwarePlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	switch eventType {
	case "system_startup":
		return p.handleSystemStartup(data)
	case "system_shutdown":
		return p.handleSystemShutdown(data)
	case "package_update_available":
		return p.handlePackageUpdateAvailable(data)
	default:
		return plugin.ErrInvalidEvent
	}
}

// Status 返回插件状态
func (p *SoftwarePlugin) Status() *plugin.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.status.Metrics["installed_count"] = len(p.installed)

	var totalSize int64
	for _, info := range p.installed {
		totalSize += info.Size
	}
	p.status.Metrics["total_size"] = totalSize

	return p.status
}

// Health 健康检查
func (p *SoftwarePlugin) Health() error {
	if p.status.Status != "running" {
		return fmt.Errorf("plugin not running")
	}
	return nil
}

// GetConfig 获取配置
func (p *SoftwarePlugin) GetConfig() map[string]interface{} {
	return p.config
}

// SetConfig 设置配置
func (p *SoftwarePlugin) SetConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

// handleInstall 处理安装命令
func (p *SoftwarePlugin) handleInstall(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	version, _ := args["version"].(string)
	packageType, _ := args["package_type"].(string)
	source, _ := args["source"].(string)

	// 检查是否已安装
	p.mu.RLock()
	if _, exists := p.installed[name]; exists {
		p.mu.RUnlock()
		return nil, fmt.Errorf("software %s is already installed", name)
	}
	p.mu.RUnlock()

	// 创建软件信息
	info := &SoftwareInfo{
		Name:        name,
		Version:     version,
		PackageType: packageType,
		InstallTime: time.Now(),
		Status:      "installing",
	}

	// 添加到已安装列表
	p.mu.Lock()
	p.installed[name] = info
	p.mu.Unlock()

	// 执行安装
	go func() {
		if err := p.performInstall(info, source); err != nil {
			p.ctx.Logger.Errorf("Failed to install %s: %v", name, err)
			info.Status = "failed"
		} else {
			info.Status = "installed"
			p.ctx.Logger.Infof("Successfully installed %s", name)
		}
	}()

	return map[string]interface{}{
		"name":    name,
		"status":  "installing",
		"message": "Installation started",
	}, nil
}

// handleUninstall 处理卸载命令
func (p *SoftwarePlugin) handleUninstall(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	p.mu.RLock()
	info, exists := p.installed[name]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("software %s is not installed", name)
	}

	// 执行卸载
	go func() {
		if err := p.performUninstall(info); err != nil {
			p.ctx.Logger.Errorf("Failed to uninstall %s: %v", name, err)
		} else {
			p.mu.Lock()
			delete(p.installed, name)
			p.mu.Unlock()
			p.ctx.Logger.Infof("Successfully uninstalled %s", name)
		}
	}()

	return map[string]interface{}{
		"name":    name,
		"status":  "uninstalling",
		"message": "Uninstallation started",
	}, nil
}

// handleList 处理列表命令
func (p *SoftwarePlugin) handleList(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	softwareList := make([]*SoftwareInfo, 0, len(p.installed))
	for _, info := range p.installed {
		softwareList = append(softwareList, info)
	}

	return map[string]interface{}{
		"software": softwareList,
		"count":    len(softwareList),
	}, nil
}

// handleInfo 处理信息命令
func (p *SoftwarePlugin) handleInfo(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	p.mu.RLock()
	info, exists := p.installed[name]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("software %s is not installed", name)
	}

	return info, nil
}

// handleUpdate 处理更新命令
func (p *SoftwarePlugin) handleUpdate(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	p.mu.RLock()
	info, exists := p.installed[name]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("software %s is not installed", name)
	}

	// 执行更新
	go func() {
		if err := p.performUpdate(info); err != nil {
			p.ctx.Logger.Errorf("Failed to update %s: %v", name, err)
		} else {
			info.LastUpdated = time.Now()
			p.ctx.Logger.Infof("Successfully updated %s", name)
		}
	}()

	return map[string]interface{}{
		"name":    name,
		"status":  "updating",
		"message": "Update started",
	}, nil
}

// handleSearch 处理搜索命令
func (p *SoftwarePlugin) handleSearch(args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	// 这里应该调用包管理器的搜索功能
	// 暂时返回模拟结果
	results := []map[string]interface{}{
		{
			"name":        query,
			"version":     "1.0.0",
			"description": "Sample software package",
			"available":   true,
		},
	}

	return map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	}, nil
}

// performInstall 执行安装
func (p *SoftwarePlugin) performInstall(info *SoftwareInfo, source string) error {
	// 根据操作系统和包类型选择安装方法
	switch runtime.GOOS {
	case "linux":
		return p.installOnLinux(info, source)
	case "windows":
		return p.installOnWindows(info, source)
	case "darwin":
		return p.installOnMacOS(info, source)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// installOnLinux Linux 安装
func (p *SoftwarePlugin) installOnLinux(info *SoftwareInfo, source string) error {
	var cmd *exec.Cmd

	switch info.PackageType {
	case "apt":
		cmd = exec.Command("apt-get", "install", "-y", info.Name)
	case "yum":
		cmd = exec.Command("yum", "install", "-y", info.Name)
	case "dnf":
		cmd = exec.Command("dnf", "install", "-y", info.Name)
	case "pacman":
		cmd = exec.Command("pacman", "-S", "--noconfirm", info.Name)
	default:
		// 尝试自动检测包管理器
		if p.hasCommand("apt-get") {
			cmd = exec.Command("apt-get", "install", "-y", info.Name)
		} else if p.hasCommand("yum") {
			cmd = exec.Command("yum", "install", "-y", info.Name)
		} else if p.hasCommand("dnf") {
			cmd = exec.Command("dnf", "install", "-y", info.Name)
		} else {
			return fmt.Errorf("no supported package manager found")
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation failed: %v, output: %s", err, string(output))
	}

	// 更新软件信息
	info.Path = p.findExecutable(info.Name)
	info.Size = p.getFileSize(info.Path)

	return nil
}

// installOnWindows Windows 安装
func (p *SoftwarePlugin) installOnWindows(info *SoftwareInfo, source string) error {
	var cmd *exec.Cmd

	switch info.PackageType {
	case "chocolatey":
		cmd = exec.Command("choco", "install", info.Name, "-y")
	case "winget":
		cmd = exec.Command("winget", "install", info.Name)
	case "scoop":
		cmd = exec.Command("scoop", "install", info.Name)
	default:
		// 尝试自动检测包管理器
		if p.hasCommand("choco") {
			cmd = exec.Command("choco", "install", info.Name, "-y")
		} else if p.hasCommand("winget") {
			cmd = exec.Command("winget", "install", info.Name)
		} else if p.hasCommand("scoop") {
			cmd = exec.Command("scoop", "install", info.Name)
		} else {
			return fmt.Errorf("no supported package manager found")
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation failed: %v, output: %s", err, string(output))
	}

	// 更新软件信息
	info.Path = p.findExecutable(info.Name)
	info.Size = p.getFileSize(info.Path)

	return nil
}

// installOnMacOS macOS 安装
func (p *SoftwarePlugin) installOnMacOS(info *SoftwareInfo, source string) error {
	var cmd *exec.Cmd

	switch info.PackageType {
	case "brew":
		cmd = exec.Command("brew", "install", info.Name)
	case "port":
		cmd = exec.Command("port", "install", info.Name)
	default:
		// 尝试自动检测包管理器
		if p.hasCommand("brew") {
			cmd = exec.Command("brew", "install", info.Name)
		} else if p.hasCommand("port") {
			cmd = exec.Command("port", "install", info.Name)
		} else {
			return fmt.Errorf("no supported package manager found")
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation failed: %v, output: %s", err, string(output))
	}

	// 更新软件信息
	info.Path = p.findExecutable(info.Name)
	info.Size = p.getFileSize(info.Path)

	return nil
}

// performUninstall 执行卸载
func (p *SoftwarePlugin) performUninstall(info *SoftwareInfo) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		switch info.PackageType {
		case "apt":
			cmd = exec.Command("apt-get", "remove", "-y", info.Name)
		case "yum":
			cmd = exec.Command("yum", "remove", "-y", info.Name)
		case "dnf":
			cmd = exec.Command("dnf", "remove", "-y", info.Name)
		case "pacman":
			cmd = exec.Command("pacman", "-R", "--noconfirm", info.Name)
		}
	case "windows":
		switch info.PackageType {
		case "chocolatey":
			cmd = exec.Command("choco", "uninstall", info.Name, "-y")
		case "winget":
			cmd = exec.Command("winget", "uninstall", info.Name)
		case "scoop":
			cmd = exec.Command("scoop", "uninstall", info.Name)
		}
	case "darwin":
		switch info.PackageType {
		case "brew":
			cmd = exec.Command("brew", "uninstall", info.Name)
		case "port":
			cmd = exec.Command("port", "uninstall", info.Name)
		}
	}

	if cmd == nil {
		return fmt.Errorf("unsupported package type: %s", info.PackageType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uninstallation failed: %v, output: %s", err, string(output))
	}

	return nil
}

// performUpdate 执行更新
func (p *SoftwarePlugin) performUpdate(info *SoftwareInfo) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		switch info.PackageType {
		case "apt":
			cmd = exec.Command("apt-get", "upgrade", "-y", info.Name)
		case "yum":
			cmd = exec.Command("yum", "update", "-y", info.Name)
		case "dnf":
			cmd = exec.Command("dnf", "update", "-y", info.Name)
		case "pacman":
			cmd = exec.Command("pacman", "-Syu", "--noconfirm", info.Name)
		}
	case "windows":
		switch info.PackageType {
		case "chocolatey":
			cmd = exec.Command("choco", "upgrade", info.Name, "-y")
		case "winget":
			cmd = exec.Command("winget", "upgrade", info.Name)
		case "scoop":
			cmd = exec.Command("scoop", "update", info.Name)
		}
	case "darwin":
		switch info.PackageType {
		case "brew":
			cmd = exec.Command("brew", "upgrade", info.Name)
		case "port":
			cmd = exec.Command("port", "upgrade", info.Name)
		}
	}

	if cmd == nil {
		return fmt.Errorf("unsupported package type: %s", info.PackageType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update failed: %v, output: %s", err, string(output))
	}

	return nil
}

// backgroundTask 后台任务
func (p *SoftwarePlugin) backgroundTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 定期检查软件更新
			p.checkForUpdates()
		case <-p.stopChan:
			return
		}
	}
}

// checkForUpdates 检查更新
func (p *SoftwarePlugin) checkForUpdates() {
	p.mu.RLock()
	softwareList := make([]*SoftwareInfo, 0, len(p.installed))
	for _, info := range p.installed {
		softwareList = append(softwareList, info)
	}
	p.mu.RUnlock()

	for range softwareList {
		// 这里应该检查每个软件的更新
		// 暂时跳过
	}
}

// loadInstalledSoftware 加载已安装软件列表
func (p *SoftwarePlugin) loadInstalledSoftware() {
	// 从文件或数据库加载已安装软件列表
	// 暂时跳过
}

// saveInstalledSoftware 保存已安装软件列表
func (p *SoftwarePlugin) saveInstalledSoftware() {
	// 保存到文件或数据库
	// 暂时跳过
}

// hasCommand 检查命令是否存在
func (p *SoftwarePlugin) hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// findExecutable 查找可执行文件
func (p *SoftwarePlugin) findExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// getFileSize 获取文件大小
func (p *SoftwarePlugin) getFileSize(path string) int64 {
	if path == "" {
		return 0
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// 事件处理方法
func (p *SoftwarePlugin) handleSystemStartup(data map[string]interface{}) error {
	p.ctx.Logger.Info("System startup event received")
	return nil
}

func (p *SoftwarePlugin) handleSystemShutdown(data map[string]interface{}) error {
	p.ctx.Logger.Info("System shutdown event received")
	return nil
}

func (p *SoftwarePlugin) handlePackageUpdateAvailable(data map[string]interface{}) error {
	p.ctx.Logger.Info("Package update available event received")
	return nil
}
