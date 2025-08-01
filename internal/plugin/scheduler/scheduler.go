package scheduler

import (
	"fmt"
	"sync"
	"time"

	"assistant_agent/internal/plugin"

	"github.com/robfig/cron/v3"
)

// SchedulerPlugin 定时任务调度器插件
type SchedulerPlugin struct {
	ctx       *plugin.PluginContext
	config    map[string]interface{}
	status    *plugin.PluginStatus
	scheduler *cron.Cron
	tasks     map[string]*TaskInfo
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// TaskInfo 任务信息
type TaskInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	CronExpr     string                 `json:"cron_expr"`
	Command      string                 `json:"command"`
	Args         []string               `json:"args"`
	Type         string                 `json:"type"` // shell, powershell, container
	Enabled      bool                   `json:"enabled"`
	Status       string                 `json:"status"` // active, paused, disabled
	LastRun      time.Time              `json:"last_run"`
	NextRun      time.Time              `json:"next_run"`
	RunCount     int64                  `json:"run_count"`
	SuccessCount int64                  `json:"success_count"`
	FailureCount int64                  `json:"failure_count"`
	LastResult   *TaskResult            `json:"last_result,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	EntryID      cron.EntryID           `json:"entry_id"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output"`
	Error     string    `json:"error,omitempty"`
	Success   bool      `json:"success"`
}

// TaskRequest 任务请求
type TaskRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CronExpr    string            `json:"cron_expr"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Type        string            `json:"type"`
	Enabled     bool              `json:"enabled"`
	Metadata    map[string]string `json:"metadata"`
}

// NewSchedulerPlugin 创建定时任务调度器插件
func NewSchedulerPlugin() *SchedulerPlugin {
	return &SchedulerPlugin{
		config:    make(map[string]interface{}),
		tasks:     make(map[string]*TaskInfo),
		stopChan:  make(chan struct{}),
		scheduler: cron.New(cron.WithSeconds()),
		status: &plugin.PluginStatus{
			Status: "stopped",
			Metrics: map[string]interface{}{
				"total_tasks":      0,
				"active_tasks":     0,
				"enabled_tasks":    0,
				"total_executions": 0,
			},
		},
	}
}

// Info 返回插件信息
func (p *SchedulerPlugin) Info() *plugin.PluginInfo {
	return &plugin.PluginInfo{
		Name:        "task-scheduler",
		Version:     "1.0.0",
		Description: "Cron-based task scheduler plugin",
		Author:      "Assistant Agent Team",
		License:     "MIT",
		Homepage:    "https://github.com/assistant-agent/plugins",
		Tags:        []string{"scheduler", "cron", "task"},
		Config: map[string]string{
			"max_concurrent_tasks": "10",
			"default_timeout":      "300",
			"retention_days":       "30",
		},
	}
}

// Init 初始化插件
func (p *SchedulerPlugin) Init(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.status.Status = "initialized"

	// 设置默认配置
	p.setDefaultConfig()

	p.ctx.Logger.Info("Task scheduler plugin initialized")
	return nil
}

// Start 启动插件
func (p *SchedulerPlugin) Start() error {
	p.status.Status = "running"
	p.status.StartTime = time.Now()

	// 启动调度器
	p.scheduler.Start()

	// 恢复已启用的任务
	p.restoreEnabledTasks()

	p.ctx.Logger.Info("Task scheduler plugin started")
	return nil
}

// Stop 停止插件
func (p *SchedulerPlugin) Stop() error {
	p.status.Status = "stopped"

	// 停止调度器
	p.scheduler.Stop()
	close(p.stopChan)

	p.ctx.Logger.Info("Task scheduler plugin stopped")
	return nil
}

// HandleCommand 处理命令
func (p *SchedulerPlugin) HandleCommand(command string, args map[string]interface{}) (interface{}, error) {
	switch command {
	case "add_task":
		return p.handleAddTask(args)
	case "update_task":
		return p.handleUpdateTask(args)
	case "remove_task":
		return p.handleRemoveTask(args)
	case "enable_task":
		return p.handleEnableTask(args)
	case "disable_task":
		return p.handleDisableTask(args)
	case "run_task":
		return p.handleRunTask(args)
	case "list_tasks":
		return p.handleListTasks(args)
	case "get_task":
		return p.handleGetTask(args)
	case "get_task_status":
		return p.handleGetTaskStatus(args)
	case "get_next_runs":
		return p.handleGetNextRuns(args)
	default:
		return nil, plugin.ErrInvalidCommand
	}
}

// HandleEvent 处理事件
func (p *SchedulerPlugin) HandleEvent(eventType string, data map[string]interface{}) error {
	switch eventType {
	case "task_completed":
		return p.handleTaskCompleted(data)
	case "task_failed":
		return p.handleTaskFailed(data)
	case "task_started":
		return p.handleTaskStarted(data)
	default:
		return plugin.ErrInvalidEvent
	}
}

// Status 返回插件状态
func (p *SchedulerPlugin) Status() *plugin.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.status.Metrics["total_tasks"] = len(p.tasks)

	activeCount := 0
	enabledCount := 0
	var totalExecutions int64

	for _, task := range p.tasks {
		if task.Status == "active" {
			activeCount++
		}
		if task.Enabled {
			enabledCount++
		}
		totalExecutions += task.RunCount
	}

	p.status.Metrics["active_tasks"] = activeCount
	p.status.Metrics["enabled_tasks"] = enabledCount
	p.status.Metrics["total_executions"] = totalExecutions

	return p.status
}

// Health 健康检查
func (p *SchedulerPlugin) Health() error {
	if p.status.Status != "running" {
		return fmt.Errorf("plugin not running")
	}
	return nil
}

// GetConfig 获取配置
func (p *SchedulerPlugin) GetConfig() map[string]interface{} {
	return p.config
}

// SetConfig 设置配置
func (p *SchedulerPlugin) SetConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

// handleAddTask 处理添加任务命令
func (p *SchedulerPlugin) handleAddTask(args map[string]interface{}) (interface{}, error) {
	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	cronExpr, ok := args["cron_expr"].(string)
	if !ok {
		return nil, fmt.Errorf("cron_expr is required")
	}

	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command is required")
	}

	description, _ := args["description"].(string)
	taskType, _ := args["type"].(string)
	if taskType == "" {
		taskType = "shell"
	}

	enabled, _ := args["enabled"].(bool)

	// 验证cron表达式
	if _, err := cron.ParseStandard(cronExpr); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %v", err)
	}

	// 创建任务
	taskID := p.generateID()
	task := &TaskInfo{
		ID:           taskID,
		Name:         name,
		Description:  description,
		CronExpr:     cronExpr,
		Command:      command,
		Type:         taskType,
		Enabled:      enabled,
		Status:       "active",
		RunCount:     0,
		SuccessCount: 0,
		FailureCount: 0,
		Metadata:     make(map[string]interface{}),
	}

	// 处理参数
	if cmdArgs, ok := args["args"].([]interface{}); ok {
		argsList := make([]string, 0, len(cmdArgs))
		for _, arg := range cmdArgs {
			if str, ok := arg.(string); ok {
				argsList = append(argsList, str)
			}
		}
		task.Args = argsList
	}

	// 添加到任务列表
	p.mu.Lock()
	p.tasks[taskID] = task
	p.mu.Unlock()

	// 如果启用，添加到调度器
	if task.Enabled {
		if err := p.addToScheduler(task); err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"id":      taskID,
		"name":    name,
		"message": "Task added successfully",
	}, nil
}

// handleUpdateTask 处理更新任务命令
func (p *SchedulerPlugin) handleUpdateTask(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	task, exists := p.tasks[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("task not found")
	}

	// 更新字段
	if name, ok := args["name"].(string); ok {
		task.Name = name
	}
	if description, ok := args["description"].(string); ok {
		task.Description = description
	}
	if cronExpr, ok := args["cron_expr"].(string); ok {
		if _, err := cron.ParseStandard(cronExpr); err != nil {
			p.mu.Unlock()
			return nil, fmt.Errorf("invalid cron expression: %v", err)
		}
		task.CronExpr = cronExpr
	}
	if command, ok := args["command"].(string); ok {
		task.Command = command
	}
	if taskType, ok := args["type"].(string); ok {
		task.Type = taskType
	}

	// 如果任务已启用，需要重新添加到调度器
	if task.Enabled && task.EntryID != 0 {
		p.scheduler.Remove(task.EntryID)
		if err := p.addToScheduler(task); err != nil {
			p.mu.Unlock()
			return nil, err
		}
	}

	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Task updated successfully",
	}, nil
}

// handleRemoveTask 处理移除任务命令
func (p *SchedulerPlugin) handleRemoveTask(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	task, exists := p.tasks[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("task not found")
	}

	// 从调度器中移除
	if task.EntryID != 0 {
		p.scheduler.Remove(task.EntryID)
	}

	// 从任务列表中移除
	delete(p.tasks, id)
	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Task removed successfully",
	}, nil
}

// handleEnableTask 处理启用任务命令
func (p *SchedulerPlugin) handleEnableTask(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	task, exists := p.tasks[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("task not found")
	}

	if !task.Enabled {
		task.Enabled = true
		task.Status = "active"
		if err := p.addToScheduler(task); err != nil {
			p.mu.Unlock()
			return nil, err
		}
	}
	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Task enabled successfully",
	}, nil
}

// handleDisableTask 处理禁用任务命令
func (p *SchedulerPlugin) handleDisableTask(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.Lock()
	task, exists := p.tasks[id]
	if !exists {
		p.mu.Unlock()
		return nil, fmt.Errorf("task not found")
	}

	if task.Enabled {
		task.Enabled = false
		task.Status = "paused"
		if task.EntryID != 0 {
			p.scheduler.Remove(task.EntryID)
			task.EntryID = 0
		}
	}
	p.mu.Unlock()

	return map[string]interface{}{
		"id":      id,
		"message": "Task disabled successfully",
	}, nil
}

// handleRunTask 处理立即运行任务命令
func (p *SchedulerPlugin) handleRunTask(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.RLock()
	task, exists := p.tasks[id]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	// 立即执行任务
	go p.executeTask(task)

	return map[string]interface{}{
		"id":      id,
		"message": "Task execution started",
	}, nil
}

// handleListTasks 处理列出任务命令
func (p *SchedulerPlugin) handleListTasks(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tasks := make([]*TaskInfo, 0, len(p.tasks))
	for _, task := range p.tasks {
		tasks = append(tasks, task)
	}

	return map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}, nil
}

// handleGetTask 处理获取任务命令
func (p *SchedulerPlugin) handleGetTask(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.RLock()
	task, exists := p.tasks[id]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	return task, nil
}

// handleGetTaskStatus 处理获取任务状态命令
func (p *SchedulerPlugin) handleGetTaskStatus(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	p.mu.RLock()
	task, exists := p.tasks[id]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	return map[string]interface{}{
		"id":            task.ID,
		"name":          task.Name,
		"status":        task.Status,
		"enabled":       task.Enabled,
		"last_run":      task.LastRun,
		"next_run":      task.NextRun,
		"run_count":     task.RunCount,
		"success_count": task.SuccessCount,
		"failure_count": task.FailureCount,
		"last_result":   task.LastResult,
	}, nil
}

// addToScheduler 添加任务到调度器
func (p *SchedulerPlugin) addToScheduler(task *TaskInfo) error {
	entryID, err := p.scheduler.AddFunc(task.CronExpr, func() {
		p.executeTask(task)
	})
	if err != nil {
		return err
	}

	task.EntryID = entryID

	// 计算下次运行时间
	entry := p.scheduler.Entry(entryID)
	task.NextRun = entry.Next

	return nil
}

// executeTask 执行任务
func (p *SchedulerPlugin) executeTask(task *TaskInfo) {
	startTime := time.Now()

	// 更新任务状态
	p.mu.Lock()
	task.LastRun = startTime
	task.RunCount++
	p.mu.Unlock()

	// 发送任务开始事件
	p.ctx.Agent.NotifyEvent("task_started", map[string]interface{}{
		"task_id":    task.ID,
		"name":       task.Name,
		"start_time": startTime,
	})

	// 执行命令
	result := &TaskResult{
		StartTime: startTime,
	}

	// 通过执行器插件执行命令
	execResult, err := p.ctx.Agent.ExecuteCommand(task.Command, task.Args, 5*time.Minute)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime).Seconds()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.ExitCode = -1

		p.mu.Lock()
		task.FailureCount++
		p.mu.Unlock()

		// 发送任务失败事件
		p.ctx.Agent.NotifyEvent("task_failed", map[string]interface{}{
			"task_id":  task.ID,
			"name":     task.Name,
			"error":    err.Error(),
			"duration": result.Duration,
		})
	} else {
		result.Success = true
		result.Output = execResult
		result.ExitCode = 0

		p.mu.Lock()
		task.SuccessCount++
		p.mu.Unlock()

		// 发送任务完成事件
		p.ctx.Agent.NotifyEvent("task_completed", map[string]interface{}{
			"task_id":  task.ID,
			"name":     task.Name,
			"output":   execResult,
			"duration": result.Duration,
		})
	}

	// 更新任务结果
	p.mu.Lock()
	task.LastResult = result
	p.mu.Unlock()

	// 计算下次运行时间
	if task.EntryID != 0 {
		entry := p.scheduler.Entry(task.EntryID)
		task.NextRun = entry.Next
	}
}

// restoreEnabledTasks 恢复已启用的任务
func (p *SchedulerPlugin) restoreEnabledTasks() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, task := range p.tasks {
		if task.Enabled {
			if err := p.addToScheduler(task); err != nil {
				p.ctx.Logger.Errorf("Failed to restore task %s: %v", task.Name, err)
			}
		}
	}
}

// setDefaultConfig 设置默认配置
func (p *SchedulerPlugin) setDefaultConfig() {
	if p.config == nil {
		p.config = make(map[string]interface{})
	}

	if _, ok := p.config["max_concurrent_tasks"]; !ok {
		p.config["max_concurrent_tasks"] = 10
	}

	if _, ok := p.config["default_timeout"]; !ok {
		p.config["default_timeout"] = 300
	}

	if _, ok := p.config["retention_days"]; !ok {
		p.config["retention_days"] = 30
	}
}

// generateID 生成唯一ID
func (p *SchedulerPlugin) generateID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// 事件处理方法
func (p *SchedulerPlugin) handleTaskCompleted(data map[string]interface{}) error {
	p.ctx.Logger.Info("Task completed event received")
	return nil
}

func (p *SchedulerPlugin) handleTaskFailed(data map[string]interface{}) error {
	p.ctx.Logger.Info("Task failed event received")
	return nil
}

func (p *SchedulerPlugin) handleTaskStarted(data map[string]interface{}) error {
	p.ctx.Logger.Info("Task started event received")
	return nil
}

// handleGetNextRuns 处理获取下次运行时间命令
func (p *SchedulerPlugin) handleGetNextRuns(args map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	nextRuns := make(map[string]time.Time)
	
	for _, task := range p.tasks {
		if task.Enabled && task.EntryID != 0 {
			entry := p.scheduler.Entry(task.EntryID)
			nextRuns[task.ID] = entry.Next
		}
	}

	return nextRuns, nil
}
