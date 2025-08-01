package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"assistant_agent/internal/logger"
)

// CommandType 命令类型
type CommandType string

const (
	CommandTypeShell      CommandType = "shell"
	CommandTypePowerShell CommandType = "powershell"
	CommandTypeContainer  CommandType = "container"
)

// Command 命令结构
type Command struct {
	ID          string      `json:"id"`
	Type        CommandType `json:"type"`
	Script      string      `json:"script"`
	Args        []string    `json:"args"`
	WorkingDir  string      `json:"working_dir"`
	Timeout     int         `json:"timeout"`
	ContainerID string      `json:"container_id,omitempty"`
	User        string      `json:"user,omitempty"`
	Env         []string    `json:"env,omitempty"`
}

// Result 执行结果
type Result struct {
	ID        string    `json:"id"`
	Success   bool      `json:"success"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output"`
	Error     string    `json:"error"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration"`
}

// Executor 命令执行器
type Executor struct {
	workDir string
	tempDir string
	mu      sync.RWMutex
	running map[string]*exec.Cmd
}

// New 创建新的执行器
func New(workDir, tempDir string) (*Executor, error) {
	// 创建目录
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	return &Executor{
		workDir: workDir,
		tempDir: tempDir,
		running: make(map[string]*exec.Cmd),
	}, nil
}

// Start 启动执行器
func (e *Executor) Start() error {
	logger.Info("Command executor started")
	return nil
}

// Stop 停止执行器
func (e *Executor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 停止所有运行中的命令
	for id, cmd := range e.running {
		logger.Infof("Stopping command: %s", id)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		delete(e.running, id)
	}

	logger.Info("Command executor stopped")
}

// Execute 执行命令
func (e *Executor) Execute(cmd *Command) *Result {
	result := &Result{
		ID:        cmd.ID,
		StartTime: time.Now(),
	}

	logger.Infof("Executing command: %s, type: %s", cmd.ID, cmd.Type)

	switch cmd.Type {
	case CommandTypeShell:
		result = e.executeShell(cmd)
	case CommandTypePowerShell:
		result = e.executePowerShell(cmd)
	case CommandTypeContainer:
		result = e.executeContainer(cmd)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported command type: %s", cmd.Type)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()

	logger.Infof("Command %s completed, success: %v, exit code: %d",
		cmd.ID, result.Success, result.ExitCode)

	return result
}

// executeShell 执行 Shell 命令
func (e *Executor) executeShell(cmd *Command) *Result {
	result := &Result{
		ID:        cmd.ID,
		StartTime: time.Now(),
	}

	// 创建临时脚本文件
	scriptFile, err := e.createScriptFile(cmd.Script, "sh")
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}
	defer os.Remove(scriptFile)

	// 设置执行权限
	if err := os.Chmod(scriptFile, 0755); err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	// 创建命令
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows 上使用 Git Bash 或 WSL
		execCmd = exec.Command("bash", scriptFile)
	} else {
		execCmd = exec.Command("bash", scriptFile)
	}

	// 设置工作目录
	if cmd.WorkingDir != "" {
		execCmd.Dir = cmd.WorkingDir
	} else {
		execCmd.Dir = e.workDir
	}

	// 设置环境变量
	execCmd.Env = append(os.Environ(), cmd.Env...)

	// 设置超时
	ctx := context.Background()
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cmd.Timeout)*time.Second)
		defer cancel()
		execCmd = exec.CommandContext(ctx, execCmd.Path, execCmd.Args[1:]...)
	}

	// 捕获输出
	output, err := execCmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if execCmd.ProcessState != nil {
			result.ExitCode = execCmd.ProcessState.ExitCode()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}

// executePowerShell 执行 PowerShell 命令
func (e *Executor) executePowerShell(cmd *Command) *Result {
	result := &Result{
		ID:        cmd.ID,
		StartTime: time.Now(),
	}

	// 创建临时脚本文件
	scriptFile, err := e.createScriptFile(cmd.Script, "ps1")
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}
	defer os.Remove(scriptFile)

	// 创建 PowerShell 命令
	execCmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptFile)

	// 设置工作目录
	if cmd.WorkingDir != "" {
		execCmd.Dir = cmd.WorkingDir
	} else {
		execCmd.Dir = e.workDir
	}

	// 设置环境变量
	execCmd.Env = append(os.Environ(), cmd.Env...)

	// 设置超时
	ctx := context.Background()
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cmd.Timeout)*time.Second)
		defer cancel()
		execCmd = exec.CommandContext(ctx, execCmd.Path, execCmd.Args[1:]...)
	}

	// 捕获输出
	output, err := execCmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if execCmd.ProcessState != nil {
			result.ExitCode = execCmd.ProcessState.ExitCode()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}

// executeContainer 在容器内执行命令
func (e *Executor) executeContainer(cmd *Command) *Result {
	result := &Result{
		ID:        cmd.ID,
		StartTime: time.Now(),
	}

	// 检查容器 ID
	if cmd.ContainerID == "" {
		result.Success = false
		result.Error = "container ID is required for container commands"
		return result
	}

	// 创建临时脚本文件
	scriptFile, err := e.createScriptFile(cmd.Script, "sh")
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}
	defer os.Remove(scriptFile)

	// 构建 docker exec 命令
	dockerArgs := []string{"exec"}

	// 添加用户参数
	if cmd.User != "" {
		dockerArgs = append(dockerArgs, "-u", cmd.User)
	}

	// 添加工作目录
	if cmd.WorkingDir != "" {
		dockerArgs = append(dockerArgs, "-w", cmd.WorkingDir)
	}

	// 添加环境变量
	for _, env := range cmd.Env {
		dockerArgs = append(dockerArgs, "-e", env)
	}

	dockerArgs = append(dockerArgs, cmd.ContainerID, "bash", scriptFile)

	// 创建命令
	execCmd := exec.Command("docker", dockerArgs...)

	// 设置超时
	ctx := context.Background()
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cmd.Timeout)*time.Second)
		defer cancel()
		execCmd = exec.CommandContext(ctx, execCmd.Path, execCmd.Args[1:]...)
	}

	// 捕获输出
	output, err := execCmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		if execCmd.ProcessState != nil {
			result.ExitCode = execCmd.ProcessState.ExitCode()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}

// createScriptFile 创建临时脚本文件
func (e *Executor) createScriptFile(script, ext string) (string, error) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp(e.tempDir, fmt.Sprintf("script_*.%s", ext))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// 写入脚本内容
	if _, err := io.WriteString(tmpFile, script); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// StopCommand 停止指定的命令
func (e *Executor) StopCommand(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if cmd, exists := e.running[id]; exists {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				return err
			}
		}
		delete(e.running, id)
		logger.Infof("Command %s stopped", id)
	}

	return nil
}

// ListRunningCommands 列出正在运行的命令
func (e *Executor) ListRunningCommands() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	commands := make([]string, 0, len(e.running))
	for id := range e.running {
		commands = append(commands, id)
	}

	return commands
}
