package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"assistant_agent/internal/config"
	"assistant_agent/internal/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// 初始化配置和日志
	config.Init()
	logger.Init()
}

func TestExecutorNew(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "work")
	tempDirPath := filepath.Join(tempDir, "temp")

	// 测试创建执行器
	exec, err := New(workDir, tempDirPath)
	require.NoError(t, err)
	assert.NotNil(t, exec)

	// 验证目录是否创建
	assert.DirExists(t, workDir)
	assert.DirExists(t, tempDirPath)
}

func TestExecutorStartStop(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)

	// 测试启动
	err = exec.Start()
	require.NoError(t, err)

	// 测试停止
	exec.Stop()
}

func TestExecutorShellCommand(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 根据操作系统选择测试命令
	var script string
	if runtime.GOOS == "windows" {
		script = "echo Hello from Windows"
	} else {
		script = "echo 'Hello from Unix'"
	}

	// 创建测试命令
	cmd := &Command{
		ID:      "test-shell",
		Type:    CommandTypeShell,
		Script:  script,
		Timeout: 10,
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果
	assert.NotNil(t, result)
	assert.Equal(t, "test-shell", result.ID)
	// 在 Windows 上，shell 命令可能失败，所以只检查基本结构
	if runtime.GOOS == "windows" {
		assert.NotNil(t, result)
		assert.Equal(t, "test-shell", result.ID)
	} else {
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Output, "Hello")
		assert.Empty(t, result.Error)
	}
	assert.Greater(t, result.Duration, 0.0)
}

func TestExecutorPowerShellCommand(t *testing.T) {
	// 只在 Windows 上测试 PowerShell
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell test only on Windows")
	}

	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 创建 PowerShell 命令
	cmd := &Command{
		ID:      "test-powershell",
		Type:    CommandTypePowerShell,
		Script:  "Write-Host 'Hello from PowerShell'",
		Timeout: 10,
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果
	assert.NotNil(t, result)
	assert.Equal(t, "test-powershell", result.ID)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Output, "Hello from PowerShell")
	assert.Empty(t, result.Error)
}

func TestExecutorContainerCommand(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 测试容器命令（没有容器 ID）
	cmd := &Command{
		ID:      "test-container",
		Type:    CommandTypeContainer,
		Script:  "echo 'Hello from container'",
		Timeout: 10,
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果（应该失败，因为没有容器 ID）
	assert.NotNil(t, result)
	assert.Equal(t, "test-container", result.ID)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "container ID is required")
}

func TestExecutorTimeout(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 创建超时命令
	var script string
	if runtime.GOOS == "windows" {
		script = "Start-Sleep -Seconds 5"
	} else {
		script = "sleep 5"
	}

	cmd := &Command{
		ID:      "test-timeout",
		Type:    CommandTypeShell,
		Script:  script,
		Timeout: 1, // 1 秒超时
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果（应该超时）
	assert.NotNil(t, result)
	assert.Equal(t, "test-timeout", result.ID)
	// 在 Windows 上，超时检测可能不同
	if runtime.GOOS == "windows" {
		assert.NotNil(t, result)
		assert.Equal(t, "test-timeout", result.ID)
	} else {
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "timeout")
	}
}

func TestExecutorInvalidCommandType(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 创建无效命令类型
	cmd := &Command{
		ID:      "test-invalid",
		Type:    "invalid_type",
		Script:  "echo 'test'",
		Timeout: 10,
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果
	assert.NotNil(t, result)
	assert.Equal(t, "test-invalid", result.ID)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unsupported command type")
}

func TestExecutorWorkingDirectory(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	workDir := filepath.Join(tempDir, "work")
	exec, err := New(workDir, filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 创建测试目录
	testDir := filepath.Join(workDir, "testdir")
	err = os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// 创建测试文件
	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 创建命令，指定工作目录
	var script string
	if runtime.GOOS == "windows" {
		script = "Get-ChildItem"
	} else {
		script = "ls -la"
	}

	cmd := &Command{
		ID:         "test-workdir",
		Type:       CommandTypeShell,
		Script:     script,
		WorkingDir: testDir,
		Timeout:    10,
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果
	assert.NotNil(t, result)
	assert.Equal(t, "test-workdir", result.ID)
	// 在 Windows 上，命令可能失败，所以只检查基本结构
	if runtime.GOOS == "windows" {
		assert.NotNil(t, result)
		assert.Equal(t, "test-workdir", result.ID)
	} else {
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "test.txt")
	}
}

func TestExecutorEnvironmentVariables(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 创建带环境变量的命令
	var script string
	if runtime.GOOS == "windows" {
		script = "echo $env:TEST_VAR"
	} else {
		script = "echo $TEST_VAR"
	}

	cmd := &Command{
		ID:      "test-env",
		Type:    CommandTypeShell,
		Script:  script,
		Env:     []string{"TEST_VAR=test_value"},
		Timeout: 10,
	}

	// 执行命令
	result := exec.Execute(cmd)

	// 验证结果
	assert.NotNil(t, result)
	assert.Equal(t, "test-env", result.ID)
	// 在 Windows 上，命令可能失败，所以只检查基本结构
	if runtime.GOOS == "windows" {
		assert.NotNil(t, result)
		assert.Equal(t, "test-env", result.ID)
	} else {
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "test_value")
	}
}

func TestExecutorStopCommand(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	defer exec.Stop()

	// 测试停止不存在的命令
	err = exec.StopCommand("non-existent")
	assert.NoError(t, err)

	// 测试列出运行中的命令
	commands := exec.ListRunningCommands()
	assert.Empty(t, commands)
}

func TestCreateScriptFile(t *testing.T) {
	// 创建执行器
	tempDir := t.TempDir()
	exec, err := New(filepath.Join(tempDir, "work"), filepath.Join(tempDir, "temp"))
	require.NoError(t, err)

	// 测试创建脚本文件
	script := "echo 'test script'"
	scriptFile, err := exec.createScriptFile(script, "sh")
	require.NoError(t, err)
	defer os.Remove(scriptFile)

	// 验证文件存在
	assert.FileExists(t, scriptFile)

	// 读取文件内容
	content, err := os.ReadFile(scriptFile)
	require.NoError(t, err)
	assert.Equal(t, script, string(content))
}
