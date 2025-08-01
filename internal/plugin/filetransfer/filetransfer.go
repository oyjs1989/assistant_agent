package filetransfer

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"assistant_agent/internal/plugin"
)

// FileTransferPlugin 文件传输插件
type FileTransferPlugin struct {
	ctx       *plugin.PluginContext
	config    map[string]interface{}
	status    *plugin.PluginStatus
	transfers map[string]*TransferInfo
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// TransferInfo 传输信息
type TransferInfo struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // upload, download
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Size        int64     `json:"size"`
	Transferred int64     `json:"transferred"`
	Status      string    `json:"status"` // pending, running, completed, failed
	Progress    float64   `json:"progress"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Error       string    `json:"error,omitempty"`
	MD5         string    `json:"md5,omitempty"`
}

// TransferRequest 传输请求
type TransferRequest struct {
	Type        string            `json:"type"`
	Source      string            `json:"source"`
	Destination string            `json:"destination"`
	Options     map[string]string `json:"options"`
}

// NewFileTransferPlugin 创建文件传输插件
func NewFileTransferPlugin() *FileTransferPlugin {
	return &FileTransferPlugin{
		config:    make(map[string]interface{}),
		transfers: make(map[string]*TransferInfo),
		stopChan:  make(chan struct{}),
		status: &plugin.PluginStatus{
			Status: "stopped",
			Metrics: map[string]interface{}{
				"total_transfers":  0,
				"active_transfers": 0,
				"total_bytes":      0,
			},
		},
	}
}

// Info 返回插件信息
func (p *FileTransferPlugin) Info() *plugin.PluginInfo {
	return &plugin.PluginInfo{
		Name:        "file-transfer",
		Version:     "1.0.0",
		Description: "File transfer and synchronization plugin",
		Author:      "Assistant Agent Team",
		License:     "MIT",
		Homepage:    "https://github.com/assistant-agent/plugins",
		Tags:        []string{"file", "transfer", "sync"},
		Config: map[string]string{
			"max_concurrent": "5",
			"chunk_size":     "8192",
			"retry_count":    "3",
		},
	}
}

// Init 初始化插件
func (p *FileTransferPlugin) Init(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.status.Status = "initialized"

	p.ctx.Logger.Info("File transfer plugin initialized")
	return nil
}

// Start 启动插件
func (p *FileTransferPlugin) Start() error {
	p.status.Status = "running"
	p.status.StartTime = time.Now()

	p.ctx.Logger.Info("File transfer plugin started")
	return nil
}

// Stop 停止插件
func (p *FileTransferPlugin) Stop() error {
	p.status.Status = "stopped"
	close(p.stopChan)

	p.ctx.Logger.Info("File transfer plugin stopped")
	return nil
}

// HandleCommand 处理命令
func (p *FileTransferPlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	switch command {
	case "upload":
		return p.handleUpload(args)
	case "download":
		return p.handleDownload(args)
	case "list":
		return p.handleList(args)
	case "status":
		return p.handleStatus(args)
	case "cancel":
		return p.handleCancel(args)
	case "sync":
		return p.handleSync(args)
	default:
		return nil, plugin.ErrInvalidCommand
	}
}

// HandleEvent 处理事件
func (p *FileTransferPlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	switch eventType {
	case "transfer_completed":
		return p.handleTransferCompleted(data)
	case "transfer_failed":
		return p.handleTransferFailed(data)
	case "disk_space_low":
		return p.handleDiskSpaceLow(data)
	default:
		return plugin.ErrInvalidEvent
	}
}

// Status 返回插件状态
func (p *FileTransferPlugin) Status() *plugin.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.status.Metrics["total_transfers"] = len(p.transfers)

	activeCount := 0
	var totalBytes int64
	for _, transfer := range p.transfers {
		if transfer.Status == "running" {
			activeCount++
		}
		totalBytes += transfer.Transferred
	}

	p.status.Metrics["active_transfers"] = activeCount
	p.status.Metrics["total_bytes"] = totalBytes

	return p.status
}

// Health 健康检查
func (p *FileTransferPlugin) Health() error {
	if p.status.Status != "running" {
		return fmt.Errorf("plugin not running")
	}
	return nil
}

// GetConfig 获取配置
func (p *FileTransferPlugin) GetConfig() map[string]interface{} {
	return p.config
}

// SetConfig 设置配置
func (p *FileTransferPlugin) SetConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

// handleUpload 处理上传命令
func (p *FileTransferPlugin) handleUpload(args map[string]interface{}) (interface{}, error) {
	source, ok := args["source"].(string)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("destination is required")
	}

	// 检查源文件是否存在
	if !p.ctx.Agent.FileExists(source) {
		return nil, fmt.Errorf("source file does not exist: %s", source)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(source)
	if err != nil {
		return nil, err
	}

	// 创建传输信息
	transferID := p.generateID()
	transfer := &TransferInfo{
		ID:          transferID,
		Type:        "upload",
		Source:      source,
		Destination: destination,
		Size:        fileInfo.Size(),
		Status:      "pending",
		StartTime:   time.Now(),
	}

	// 添加到传输列表
	p.mu.Lock()
	p.transfers[transferID] = transfer
	p.mu.Unlock()

	// 异步执行上传
	go func() {
		if err := p.performUpload(transfer); err != nil {
			transfer.Status = "failed"
			transfer.Error = err.Error()
			p.ctx.Logger.Errorf("Upload failed: %v", err)
		} else {
			transfer.Status = "completed"
			transfer.Progress = 100.0
			p.ctx.Logger.Infof("Upload completed: %s", source)
		}
		transfer.EndTime = time.Now()
	}()

	return map[string]interface{}{
		"id":      transferID,
		"status":  "started",
		"message": "Upload started",
	}, nil
}

// handleDownload 处理下载命令
func (p *FileTransferPlugin) handleDownload(args map[string]interface{}) (interface{}, error) {
	source, ok := args["source"].(string)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("destination is required")
	}

	// 创建传输信息
	transferID := p.generateID()
	transfer := &TransferInfo{
		ID:          transferID,
		Type:        "download",
		Source:      source,
		Destination: destination,
		Status:      "pending",
		StartTime:   time.Now(),
	}

	// 添加到传输列表
	p.mu.Lock()
	p.transfers[transferID] = transfer
	p.mu.Unlock()

	// 异步执行下载
	go func() {
		if err := p.performDownload(transfer); err != nil {
			transfer.Status = "failed"
			transfer.Error = err.Error()
			p.ctx.Logger.Errorf("Download failed: %v", err)
		} else {
			transfer.Status = "completed"
			transfer.Progress = 100.0
			p.ctx.Logger.Infof("Download completed: %s", destination)
		}
		transfer.EndTime = time.Now()
	}()

	return map[string]interface{}{
		"id":      transferID,
		"status":  "started",
		"message": "Download started",
	}, nil
}

// handleList 处理列表命令
func (p *FileTransferPlugin) handleList(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	transfers := make([]*TransferInfo, 0, len(p.transfers))
	for _, transfer := range p.transfers {
		transfers = append(transfers, transfer)
	}

	return map[string]interface{}{
		"transfers": transfers,
		"count":     len(transfers),
	}, nil
}

// handleStatus 处理状态命令
func (p *FileTransferPlugin) handleStatus(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.RLock()
	transfer, exists := p.transfers[id]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("transfer not found")
	}

	return transfer, nil
}

// handleCancel 处理取消命令
func (p *FileTransferPlugin) handleCancel(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	transfer, exists := p.transfers[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("transfer not found")
	}

	if transfer.Status == "running" {
		transfer.Status = "cancelled"
	}
	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Transfer cancelled",
	}, nil
}

// handleSync 处理同步命令
func (p *FileTransferPlugin) handleSync(args map[string]interface{}) (interface{}, error) {
	source, ok := args["source"].(string)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("destination is required")
	}

	// 执行同步
	go func() {
		if err := p.performSync(source, destination); err != nil {
			p.ctx.Logger.Errorf("Sync failed: %v", err)
		} else {
			p.ctx.Logger.Infof("Sync completed: %s -> %s", source, destination)
		}
	}()

	return map[string]interface{}{
		"status":  "started",
		"message": "Sync started",
	}, nil
}

// performUpload 执行上传
func (p *FileTransferPlugin) performUpload(transfer *TransferInfo) error {
	transfer.Status = "running"

	// 读取源文件
	sourceData, err := p.ctx.Agent.ReadFile(transfer.Source)
	if err != nil {
		return err
	}

	transfer.Size = int64(len(sourceData))

	// 写入目标文件
	if err := p.ctx.Agent.WriteFile(transfer.Destination, sourceData); err != nil {
		return err
	}

	transfer.Transferred = transfer.Size
	transfer.Progress = 100.0

	// 计算MD5
	hash := md5.Sum(sourceData)
	transfer.MD5 = hex.EncodeToString(hash[:])

	return nil
}

// performDownload 执行下载
func (p *FileTransferPlugin) performDownload(transfer *TransferInfo) error {
	transfer.Status = "running"

	// 读取源文件
	sourceData, err := p.ctx.Agent.ReadFile(transfer.Source)
	if err != nil {
		return err
	}

	transfer.Size = int64(len(sourceData))

	// 写入目标文件
	if err := p.ctx.Agent.WriteFile(transfer.Destination, sourceData); err != nil {
		return err
	}

	transfer.Transferred = transfer.Size
	transfer.Progress = 100.0

	// 计算MD5
	hash := md5.Sum(sourceData)
	transfer.MD5 = hex.EncodeToString(hash[:])

	return nil
}

// performSync 执行同步
func (p *FileTransferPlugin) performSync(source, destination string) error {
	// 简单的文件同步实现
	// 这里可以实现更复杂的同步逻辑，如增量同步、目录同步等

	if !p.ctx.Agent.FileExists(source) {
		return fmt.Errorf("source does not exist: %s", source)
	}

	// 读取源文件
	sourceData, err := p.ctx.Agent.ReadFile(source)
	if err != nil {
		return err
	}

	// 写入目标文件
	return p.ctx.Agent.WriteFile(destination, sourceData)
}

// generateID 生成唯一ID
func (p *FileTransferPlugin) generateID() string {
	b := make([]byte, 16)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)
}

// 事件处理方法
func (p *FileTransferPlugin) handleTransferCompleted(data map[string]interface{}) error {
	p.ctx.Logger.Info("Transfer completed event received")
	return nil
}

func (p *FileTransferPlugin) handleTransferFailed(data map[string]interface{}) error {
	p.ctx.Logger.Info("Transfer failed event received")
	return nil
}

func (p *FileTransferPlugin) handleDiskSpaceLow(data map[string]interface{}) error {
	p.ctx.Logger.Info("Disk space low event received")
	return nil
}
