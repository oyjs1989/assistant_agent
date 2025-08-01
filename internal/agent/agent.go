package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"assistant_agent/internal/config"
	"assistant_agent/internal/executor"
	"assistant_agent/internal/heartbeat"
	"assistant_agent/internal/logger"
	"assistant_agent/internal/plugin"
	"assistant_agent/internal/plugin/filetransfer"
	"assistant_agent/internal/plugin/monitor"
	"assistant_agent/internal/plugin/password"
	"assistant_agent/internal/plugin/scheduler"
	"assistant_agent/internal/plugin/software"
	"assistant_agent/internal/plugin/updater"
	"assistant_agent/internal/state"
	"assistant_agent/internal/sysinfo"
	"assistant_agent/internal/websocket"
)

// Agent 主代理结构
type Agent struct {
	config *config.Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 核心组件
	stateMgr  *state.Manager
	heartbeat *heartbeat.Heartbeat
	wsClient  *websocket.Client
	pluginMgr *plugin.Manager
	sysinfo   *sysinfo.Collector
	executor  *executor.Executor

	// 状态
	running bool
	mu      sync.RWMutex
}

// New 创建新的 Agent 实例
func New() (*Agent, error) {
	cfg := config.GetConfig()
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	// 初始化组件
	if err := agent.initComponents(); err != nil {
		cancel()
		return nil, err
	}

	return agent, nil
}

// initComponents 初始化所有组件
func (a *Agent) initComponents() error {
	var err error

	// 初始化状态管理器
	a.stateMgr, err = state.NewManager(a.config.Agent.DataDir)
	if err != nil {
		return err
	}

	// 初始化心跳检测
	a.heartbeat, err = heartbeat.New(a.config.Agent.Heartbeat)
	if err != nil {
		return err
	}

	// 初始化 WebSocket 客户端
	a.wsClient, err = websocket.NewClient(a.config.Server.URL, a.config.Security.Token)
	if err != nil {
		return err
	}

	// 初始化系统信息收集器
	a.sysinfo, err = sysinfo.NewCollector()
	if err != nil {
		return err
	}

	// 初始化命令执行器
	a.executor, err = executor.New(a.config.Agent.WorkDir, a.config.Agent.TempDir)
	if err != nil {
		return err
	}

	// 初始化插件管理器
	a.pluginMgr = plugin.NewManager(a, a.config)

	// 注册内置插件
	if err := a.registerBuiltinPlugins(); err != nil {
		logger.Warnf("Failed to register builtin plugins: %v", err)
	}

	return nil
}

// Start 启动 Agent
func (a *Agent) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	logger.Info("Starting Assistant Agent...")

	// 启动状态管理器
	if err := a.stateMgr.Start(); err != nil {
		return err
	}

	// 启动心跳检测
	a.wg.Add(1)
	go a.runHeartbeat()

	// 启动 WebSocket 连接
	a.wg.Add(1)
	go a.runWebSocketClient()

	// 启动命令执行器
	if err := a.executor.Start(); err != nil {
		return err
	}

	// 启动插件管理器
	if err := a.pluginMgr.StartAll(); err != nil {
		logger.Warnf("Failed to start some plugins: %v", err)
	}

	a.running = true
	logger.Info("Assistant Agent started successfully")

	return nil
}

// Stop 停止 Agent
func (a *Agent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return
	}

	logger.Info("Stopping Assistant Agent...")

	// 取消上下文
	a.cancel()

	// 停止 WebSocket 客户端
	if a.wsClient != nil {
		a.wsClient.Stop()
	}

	// 停止心跳检测
	if a.heartbeat != nil {
		a.heartbeat.Stop()
	}

	// 停止状态管理器
	if a.stateMgr != nil {
		a.stateMgr.Stop()
	}

	// 停止命令执行器
	if a.executor != nil {
		a.executor.Stop()
	}

	// 停止插件管理器
	if a.pluginMgr != nil {
		a.pluginMgr.Stop()
	}

	// 等待所有 goroutine 结束
	a.wg.Wait()

	a.running = false
	logger.Info("Assistant Agent stopped")
}

// runHeartbeat 运行心跳检测
func (a *Agent) runHeartbeat() {
	defer a.wg.Done()

	ticker := time.NewTicker(time.Duration(a.config.Agent.Heartbeat) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.sendHeartbeat()
		case <-a.ctx.Done():
			return
		}
	}
}

// sendHeartbeat 发送心跳
func (a *Agent) sendHeartbeat() {
	if a.heartbeat != nil {
		a.heartbeat.Send()
	}
}

// runWebSocketClient 运行 WebSocket 客户端
func (a *Agent) runWebSocketClient() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if err := a.wsClient.Connect(); err != nil {
				logger.Errorf("Failed to connect to WebSocket server: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// 处理消息
			for {
				select {
				case <-a.ctx.Done():
					return
				default:
					msgType, data, err := a.wsClient.Receive()
					if err != nil {
						logger.Errorf("Failed to receive message: %v", err)
						break
					}

					if err := a.handleMessage(msgType, data); err != nil {
						logger.Errorf("Failed to handle message: %v", err)
					}
				}
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (a *Agent) handleMessage(msgType string, data interface{}) error {
	switch msgType {
	case "command":
		return a.handleCommand(data)
	case "schedule":
		return a.handleSchedule(data)
	case "file_transfer":
		return a.handleFileTransfer(data)
	case "update":
		return a.handleUpdate(data)
	case "plugin":
		return a.handlePluginCommand(data)
	default:
		logger.Warnf("Unknown message type: %s", msgType)
		return nil
	}
}

// handleCommand 处理命令消息
func (a *Agent) handleCommand(data interface{}) error {
	// 直接使用命令执行器处理命令
	if a.executor != nil {
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid command data format")
		}

		// 构建命令
		cmd := &executor.Command{
			Type:       executor.CommandTypeShell,
			Script:     dataMap["command"].(string),
			Args:       []string{},
			WorkingDir: a.config.Agent.WorkDir,
			Timeout:    300, // 默认5分钟超时
		}

		// 如果有参数，添加到Args中
		if args, ok := dataMap["args"].([]interface{}); ok {
			for _, arg := range args {
				if str, ok := arg.(string); ok {
					cmd.Args = append(cmd.Args, str)
				}
			}
		}

		// 执行命令
		result := a.executor.Execute(cmd)
		if !result.Success {
			return fmt.Errorf("command execution failed: %s", result.Error)
		}

		return nil
	}
	return fmt.Errorf("executor not available")
}

// handleSchedule 处理定时任务消息
func (a *Agent) handleSchedule(data interface{}) error {
	// 通过调度器插件处理定时任务
	if a.pluginMgr != nil {
		schedulerPlugin, exists := a.pluginMgr.GetPlugin("scheduler")
		if exists {
			dataMap, ok := data.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid schedule data format")
			}

			// 获取命令类型，默认为 add_task
			command, ok := dataMap["command"].(string)
			if !ok {
				command = "add_task"
			}

			// 移除 command 字段，其余作为参数传递
			args := make(map[string]interface{})
			for key, value := range dataMap {
				if key != "command" {
					args[key] = value
				}
			}

			result, err := schedulerPlugin.HandleCommand(command, args)
			if err != nil {
				return err
			}

			// 发送结果回服务器
			return a.wsClient.Send("schedule_result", map[string]interface{}{
				"command": command,
				"result":  result,
			})
		}
	}
	return fmt.Errorf("scheduler plugin not available")
}

// handleFileTransfer 处理文件传输消息
func (a *Agent) handleFileTransfer(data interface{}) error {
	// 通过文件传输插件处理文件传输
	if a.pluginMgr != nil {
		filetransferPlugin, exists := a.pluginMgr.GetPlugin("filetransfer")
		if exists {
			_, err := filetransferPlugin.HandleCommand("upload", data.(map[string]interface{}))
			return err
		}
	}
	return fmt.Errorf("filetransfer plugin not available")
}

// handleUpdate 处理更新消息
func (a *Agent) handleUpdate(data interface{}) error {
	// 通过更新插件处理更新
	if a.pluginMgr != nil {
		updaterPlugin, exists := a.pluginMgr.GetPlugin("updater")
		if exists {
			dataMap, ok := data.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid update data format")
			}

			// 获取命令类型，默认为 check_update
			command, ok := dataMap["command"].(string)
			if !ok {
				command = "check_update"
			}

			// 移除 command 字段，其余作为参数传递
			args := make(map[string]interface{})
			for key, value := range dataMap {
				if key != "command" {
					args[key] = value
				}
			}

			result, err := updaterPlugin.HandleCommand(command, args)
			if err != nil {
				return err
			}

			// 发送结果回服务器
			return a.wsClient.Send("update_result", map[string]interface{}{
				"command": command,
				"result":  result,
			})
		}
	}
	return fmt.Errorf("updater plugin not available")
}

// handlePluginCommand 处理插件命令
func (a *Agent) handlePluginCommand(data interface{}) error {
	if a.pluginMgr == nil {
		return fmt.Errorf("plugin manager not available")
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid plugin command data")
	}

	pluginName, ok := dataMap["plugin"].(string)
	if !ok {
		return fmt.Errorf("plugin name not specified")
	}

	command, ok := dataMap["command"].(string)
	if !ok {
		return fmt.Errorf("plugin command not specified")
	}

	args, _ := dataMap["args"].(map[string]interface{})

	plugin, exists := a.pluginMgr.GetPlugin(pluginName)
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	result, err := plugin.HandleCommand(command, args)
	if err != nil {
		return err
	}

	// 发送结果回服务器
	return a.wsClient.Send("plugin_result", map[string]interface{}{
		"plugin":  pluginName,
		"command": command,
		"result":  result,
	})
}

// IsRunning 检查 Agent 是否正在运行
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// registerBuiltinPlugins 注册内置插件
func (a *Agent) registerBuiltinPlugins() error {
	// 注册软件管理插件
	softwarePlugin := software.NewSoftwarePlugin()
	if err := a.pluginMgr.Register(softwarePlugin); err != nil {
		return err
	}

	// 注册密码管理插件
	passwordPlugin := password.NewPasswordPlugin()
	if err := a.pluginMgr.Register(passwordPlugin); err != nil {
		return err
	}

	// 注册文件传输插件
	filetransferPlugin := filetransfer.NewFileTransferPlugin()
	if err := a.pluginMgr.Register(filetransferPlugin); err != nil {
		return err
	}

	// 注册系统监控插件
	monitorPlugin := monitor.NewMonitorPlugin()
	if err := a.pluginMgr.Register(monitorPlugin); err != nil {
		return err
	}

	// 注册定时任务调度器插件
	schedulerPlugin := scheduler.NewSchedulerPlugin()
	if err := a.pluginMgr.Register(schedulerPlugin); err != nil {
		return err
	}

	// 注册自动更新插件
	updaterPlugin := updater.NewUpdaterPlugin()
	if err := a.pluginMgr.Register(updaterPlugin); err != nil {
		return err
	}

	return nil
}

// AgentInterface 实现
func (a *Agent) GetSystemInfo() (map[string]interface{}, error) {
	// 直接使用系统信息收集器获取系统信息
	if a.sysinfo != nil {
		return a.sysinfo.Collect()
	}

	// 返回基本信息
	return map[string]interface{}{
		"hostname":     "unknown",
		"os":           "unknown",
		"arch":         "unknown",
		"cpu_count":    0,
		"memory_total": int64(0),
	}, nil
}

func (a *Agent) ExecuteCommand(command string, args []string, timeout time.Duration) (string, error) {
	// 直接使用命令执行器执行命令
	if a.executor != nil {
		cmd := &executor.Command{
			Type:       executor.CommandTypeShell,
			Script:     command,
			Args:       args,
			WorkingDir: a.config.Agent.WorkDir,
			Timeout:    int(timeout.Seconds()),
		}

		result := a.executor.Execute(cmd)
		if !result.Success {
			return "", fmt.Errorf("command execution failed: %s", result.Error)
		}

		return result.Output, nil
	}

	// 如果执行器不可用，返回错误
	return "", fmt.Errorf("executor not available")
}

func (a *Agent) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (a *Agent) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (a *Agent) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a *Agent) GetConfig(key string) interface{} {
	// 从配置中获取值
	switch key {
	case "server.host":
		return a.config.Server.Host
	case "server.port":
		return a.config.Server.Port
	case "agent.name":
		return a.config.Agent.Name
	case "agent.work_dir":
		return a.config.Agent.WorkDir
	case "agent.data_dir":
		return a.config.Agent.DataDir
	case "agent.temp_dir":
		return a.config.Agent.TempDir
	case "logging.level":
		return a.config.Logging.Level
	case "logging.file":
		return a.config.Logging.File
	case "security.token":
		return a.config.Security.Token
	default:
		return nil
	}
}

func (a *Agent) SetConfig(key string, value interface{}) error {
	// 这里可以实现动态配置更新
	// 暂时返回不支持的错误
	return fmt.Errorf("dynamic config update not supported")
}

func (a *Agent) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"running": a.running,
		"uptime":  time.Since(a.stateMgr.GetStartTime()).Seconds(),
	}

	// 添加插件状态
	if a.pluginMgr != nil {
		pluginStatuses := a.pluginMgr.GetAllPluginStatus()
		status["plugins"] = pluginStatuses
	}

	return status
}

func (a *Agent) SetStatus(key string, value interface{}) error {
	// 这里可以实现状态更新
	// 暂时返回不支持的错误
	return fmt.Errorf("status update not supported")
}

func (a *Agent) NotifyEvent(eventType string, data map[string]interface{}) error {
	// 通过 WebSocket 发送事件到服务器
	return a.wsClient.Send("event", map[string]interface{}{
		"type": eventType,
		"data": data,
	})
}
