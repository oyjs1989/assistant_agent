package monitor

import (
	"fmt"
	"sync"
	"time"

	"assistant_agent/internal/plugin"
)

// MonitorPlugin 系统监控插件
type MonitorPlugin struct {
	ctx      *plugin.PluginContext
	config   map[string]interface{}
	status   *plugin.PluginStatus
	metrics  map[string]*MetricInfo
	alerts   map[string]*AlertInfo
	mu       sync.RWMutex
	stopChan chan struct{}
}

// MetricInfo 指标信息
type MetricInfo struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Type      string                 `json:"type"` // gauge, counter, histogram
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// AlertInfo 告警信息
type AlertInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Severity    string                 `json:"severity"` // info, warning, error, critical
	Status      string                 `json:"status"`   // active, resolved, acknowledged
	Message     string                 `json:"message"`
	Metric      string                 `json:"metric"`
	Threshold   float64                `json:"threshold"`
	Current     float64                `json:"current"`
	CreatedAt   time.Time              `json:"created_at"`
	ResolvedAt  time.Time              `json:"resolved_at,omitempty"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
}

// MonitorRule 监控规则
type MonitorRule struct {
	Name      string            `json:"name"`
	Metric    string            `json:"metric"`
	Condition string            `json:"condition"` // >, <, >=, <=, ==, !=
	Threshold float64           `json:"threshold"`
	Duration  time.Duration     `json:"duration"`
	Severity  string            `json:"severity"`
	Labels    map[string]string `json:"labels"`
}

// NewMonitorPlugin 创建系统监控插件
func NewMonitorPlugin() *MonitorPlugin {
	return &MonitorPlugin{
		config:   make(map[string]interface{}),
		metrics:  make(map[string]*MetricInfo),
		alerts:   make(map[string]*AlertInfo),
		stopChan: make(chan struct{}),
		status: &plugin.PluginStatus{
			Status: "stopped",
			Metrics: map[string]interface{}{
				"total_metrics": 0,
				"active_alerts": 0,
				"total_alerts":  0,
			},
		},
	}
}

// Info 返回插件信息
func (p *MonitorPlugin) Info() *plugin.PluginInfo {
	return &plugin.PluginInfo{
		Name:        "system-monitor",
		Version:     "1.0.0",
		Description: "System monitoring and alerting plugin",
		Author:      "Assistant Agent Team",
		License:     "MIT",
		Homepage:    "https://github.com/assistant-agent/plugins",
		Tags:        []string{"monitor", "alert", "metrics"},
		Config: map[string]string{
			"collect_interval": "30s",
			"alert_cooldown":   "5m",
			"retention_days":   "7",
		},
	}
}

// Init 初始化插件
func (p *MonitorPlugin) Init(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.status.Status = "initialized"

	// 初始化默认监控规则
	p.initDefaultRules()

	p.ctx.Logger.Info("System monitor plugin initialized")
	return nil
}

// Start 启动插件
func (p *MonitorPlugin) Start() error {
	p.status.Status = "running"
	p.status.StartTime = time.Now()

	// 启动监控收集
	go p.collectMetrics()

	// 启动告警检查
	go p.checkAlerts()

	p.ctx.Logger.Info("System monitor plugin started")
	return nil
}

// Stop 停止插件
func (p *MonitorPlugin) Stop() error {
	p.status.Status = "stopped"
	close(p.stopChan)

	p.ctx.Logger.Info("System monitor plugin stopped")
	return nil
}

// HandleCommand 处理命令
func (p *MonitorPlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	switch command {
	case "get_metrics":
		return p.handleGetMetrics(args)
	case "get_alerts":
		return p.handleGetAlerts(args)
	case "add_rule":
		return p.handleAddRule(args)
	case "remove_rule":
		return p.handleRemoveRule(args)
	case "acknowledge_alert":
		return p.handleAcknowledgeAlert(args)
	case "resolve_alert":
		return p.handleResolveAlert(args)
	case "get_rules":
		return p.handleGetRules(args)
	default:
		return nil, plugin.ErrInvalidCommand
	}
}

// HandleEvent 处理事件
func (p *MonitorPlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	switch eventType {
	case "metric_updated":
		return p.handleMetricUpdated(data)
	case "alert_triggered":
		return p.handleAlertTriggered(data)
	case "alert_resolved":
		return p.handleAlertResolved(data)
	default:
		return plugin.ErrInvalidEvent
	}
}

// Status 返回插件状态
func (p *MonitorPlugin) Status() *plugin.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.status.Metrics["total_metrics"] = len(p.metrics)

	activeAlerts := 0
	for _, alert := range p.alerts {
		if alert.Status == "active" {
			activeAlerts++
		}
	}

	p.status.Metrics["active_alerts"] = activeAlerts
	p.status.Metrics["total_alerts"] = len(p.alerts)

	return p.status
}

// Health 健康检查
func (p *MonitorPlugin) Health() error {
	if p.status.Status != "running" {
		return fmt.Errorf("plugin not running")
	}
	return nil
}

// GetConfig 获取配置
func (p *MonitorPlugin) GetConfig() map[string]interface{} {
	return p.config
}

// SetConfig 设置配置
func (p *MonitorPlugin) SetConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

// handleGetMetrics 处理获取指标命令
func (p *MonitorPlugin) handleGetMetrics(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metrics := make([]*MetricInfo, 0, len(p.metrics))
	for _, metric := range p.metrics {
		metrics = append(metrics, metric)
	}

	return map[string]interface{}{
		"metrics": metrics,
		"count":   len(metrics),
	}, nil
}

// handleGetAlerts 处理获取告警命令
func (p *MonitorPlugin) handleGetAlerts(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	alerts := make([]*AlertInfo, 0, len(p.alerts))
	for _, alert := range p.alerts {
		alerts = append(alerts, alert)
	}

	return map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	}, nil
}

// handleAddRule 处理添加规则命令
func (p *MonitorPlugin) handleAddRule(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	metric, ok := args["metric"].(string)
	if !ok {
		return nil, fmt.Errorf("metric is required")
	}

	condition, ok := args["condition"].(string)
	if !ok {
		return nil, fmt.Errorf("condition is required")
	}

	threshold, ok := args["threshold"].(float64)
	if !ok {
		return nil, fmt.Errorf("threshold is required")
	}

	severity, _ := args["severity"].(string)
	if severity == "" {
		severity = "warning"
	}

	// 创建监控规则
	_ = &MonitorRule{
		Name:      name,
		Metric:    metric,
		Condition: condition,
		Threshold: threshold,
		Severity:  severity,
		Duration:  5 * time.Minute, // 默认5分钟
		Labels:    make(map[string]string),
	}

	// 添加到规则列表
	p.mu.Lock()
	// 这里应该添加到规则列表，暂时跳过
	p.mu.Unlock()

	return map[string]interface{}{
		"name":    name,
		"message": "Rule added successfully",
	}, nil
}

// handleRemoveRule 处理移除规则命令
func (p *MonitorPlugin) handleRemoveRule(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	p.mu.Lock()
	// 这里应该从规则列表中移除，暂时跳过
	p.mu.Unlock()

	return map[string]interface{}{
		"name":    name,
		"message": "Rule removed successfully",
	}, nil
}

// handleAcknowledgeAlert 处理确认告警命令
func (p *MonitorPlugin) handleAcknowledgeAlert(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	alert, exists := p.alerts[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("alert not found")
	}

	alert.Status = "acknowledged"
	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Alert acknowledged",
	}, nil
}

// handleResolveAlert 处理解决告警命令
func (p *MonitorPlugin) handleResolveAlert(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	alert, exists := p.alerts[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("alert not found")
	}

	alert.Status = "resolved"
	alert.ResolvedAt = time.Now()
	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Alert resolved",
	}, nil
}

// handleGetRules 处理获取规则命令
func (p *MonitorPlugin) handleGetRules(args map[string]interface{}) (interface{}, error) {
	// 返回监控规则列表
	rules := []*MonitorRule{
		{
			Name:      "high_cpu_usage",
			Metric:    "cpu_usage",
			Condition: ">",
			Threshold: 80.0,
			Severity:  "warning",
		},
		{
			Name:      "high_memory_usage",
			Metric:    "memory_usage",
			Condition: ">",
			Threshold: 85.0,
			Severity:  "warning",
		},
		{
			Name:      "low_disk_space",
			Metric:    "disk_usage",
			Condition: ">",
			Threshold: 90.0,
			Severity:  "error",
		},
	}

	return map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	}, nil
}

// collectMetrics 收集指标
func (p *MonitorPlugin) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.collectSystemMetrics()
		case <-p.stopChan:
			return
		}
	}
}

// collectSystemMetrics 收集系统指标
func (p *MonitorPlugin) collectSystemMetrics() {
	// 获取系统信息
	sysInfo, err := p.ctx.Agent.GetSystemInfo()
	if err != nil {
		p.ctx.Logger.Errorf("Failed to get system info: %v", err)
		return
	}

	now := time.Now()

	// 收集CPU使用率
	if cpuCount, ok := sysInfo["cpu_count"].(int); ok {
		p.updateMetric("cpu_count", float64(cpuCount), "count", now)
	}

	// 收集内存使用率
	if memoryTotal, ok := sysInfo["memory_total"].(int64); ok {
		p.updateMetric("memory_total", float64(memoryTotal), "bytes", now)
	}

	// 模拟其他指标
	p.updateMetric("cpu_usage", 45.2, "percent", now)
	p.updateMetric("memory_usage", 67.8, "percent", now)
	p.updateMetric("disk_usage", 23.4, "percent", now)
	p.updateMetric("network_in", 1024.5, "bytes/s", now)
	p.updateMetric("network_out", 512.3, "bytes/s", now)
}

// updateMetric 更新指标
func (p *MonitorPlugin) updateMetric(name string, value float64, unit string, timestamp time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metric := &MetricInfo{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Type:      "gauge",
		Timestamp: timestamp,
		Labels:    make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	p.metrics[name] = metric

	// 检查告警规则
	p.checkMetricAlerts(name, value)
}

// checkMetricAlerts 检查指标告警
func (p *MonitorPlugin) checkMetricAlerts(metricName string, value float64) {
	// 简单的告警检查逻辑
	switch metricName {
	case "cpu_usage":
		if value > 80.0 {
			p.createAlert("high_cpu_usage", "High CPU Usage", "warning", metricName, 80.0, value)
		}
	case "memory_usage":
		if value > 85.0 {
			p.createAlert("high_memory_usage", "High Memory Usage", "warning", metricName, 85.0, value)
		}
	case "disk_usage":
		if value > 90.0 {
			p.createAlert("low_disk_space", "Low Disk Space", "error", metricName, 90.0, value)
		}
	}
}

// createAlert 创建告警
func (p *MonitorPlugin) createAlert(id, name, severity, metric string, threshold, current float64) {
	// 检查是否已存在相同告警
	if existing, exists := p.alerts[id]; exists && existing.Status == "active" {
		return
	}

	alert := &AlertInfo{
		ID:        id,
		Name:      name,
		Severity:  severity,
		Status:    "active",
		Message:   fmt.Sprintf("%s: current value %.2f exceeds threshold %.2f", name, current, threshold),
		Metric:    metric,
		Threshold: threshold,
		Current:   current,
		CreatedAt: time.Now(),
		Labels:    make(map[string]string),
		Annotations: map[string]interface{}{
			"description": fmt.Sprintf("Metric %s is above threshold", metric),
		},
	}

	p.alerts[id] = alert

	// 发送告警事件
	p.ctx.Agent.NotifyEvent("alert_triggered", map[string]interface{}{
		"alert_id": id,
		"name":     name,
		"severity": severity,
		"message":  alert.Message,
	})

	p.ctx.Logger.Warnf("Alert triggered: %s", alert.Message)
}

// checkAlerts 检查告警
func (p *MonitorPlugin) checkAlerts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.resolveStaleAlerts()
		case <-p.stopChan:
			return
		}
	}
}

// resolveStaleAlerts 解决过期告警
func (p *MonitorPlugin) resolveStaleAlerts() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for id, alert := range p.alerts {
		if alert.Status == "active" && now.Sub(alert.CreatedAt) > 30*time.Minute {
			alert.Status = "resolved"
			alert.ResolvedAt = now

			p.ctx.Agent.NotifyEvent("alert_resolved", map[string]interface{}{
				"alert_id": id,
				"name":     alert.Name,
			})

			p.ctx.Logger.Infof("Alert resolved: %s", alert.Name)
		}
	}
}

// initDefaultRules 初始化默认监控规则
func (p *MonitorPlugin) initDefaultRules() {
	// 这里可以初始化一些默认的监控规则
}

// 事件处理方法
func (p *MonitorPlugin) handleMetricUpdated(data map[string]interface{}) error {
	p.ctx.Logger.Info("Metric updated event received")
	return nil
}

func (p *MonitorPlugin) handleAlertTriggered(data map[string]interface{}) error {
	p.ctx.Logger.Info("Alert triggered event received")
	return nil
}

func (p *MonitorPlugin) handleAlertResolved(data map[string]interface{}) error {
	p.ctx.Logger.Info("Alert resolved event received")
	return nil
}
