package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Agent    AgentConfig    `mapstructure:"agent"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	URL  string `mapstructure:"url"`
}

// AgentConfig 代理配置
type AgentConfig struct {
	ID            string `mapstructure:"id"`
	Name          string `mapstructure:"name"`
	Version       string `mapstructure:"version"`
	Heartbeat     int    `mapstructure:"heartbeat"`
	MaxRetries    int    `mapstructure:"max_retries"`
	RetryDelay    int    `mapstructure:"retry_delay"`
	WorkDir       string `mapstructure:"work_dir"`
	TempDir       string `mapstructure:"temp_dir"`
	LogDir        string `mapstructure:"log_dir"`
	DataDir       string `mapstructure:"data_dir"`
	ContainerMode bool   `mapstructure:"container_mode"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	Token     string `mapstructure:"token"`
	CertFile  string `mapstructure:"cert_file"`
	KeyFile   string `mapstructure:"key_file"`
	VerifySSL bool   `mapstructure:"verify_ssl"`
}

var (
	// GlobalConfig 全局配置实例
	GlobalConfig *Config
	configFile   = "config.yaml"
)

// getSystemDirectories 获取系统标准目录
func getSystemDirectories() (tempDir, logDir, workDir, dataDir string) {
	switch runtime.GOOS {
	case "windows":
		// Windows 系统
		tempDir = os.TempDir() // 通常是 C:\Users\<username>\AppData\Local\Temp
		
		// 尝试使用 ProgramData，如果不可用则使用 AppData
		if programData := os.Getenv("PROGRAMDATA"); programData != "" {
			logDir = filepath.Join(programData, "assistant_agent", "logs")
			workDir = filepath.Join(programData, "assistant_agent", "work")
			dataDir = filepath.Join(programData, "assistant_agent", "data")
		} else {
			appData := os.Getenv("APPDATA")
			if appData == "" {
				appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
			}
			logDir = filepath.Join(appData, "assistant_agent", "logs")
			workDir = filepath.Join(appData, "assistant_agent", "work")
			dataDir = filepath.Join(appData, "assistant_agent", "data")
		}
		
	case "linux":
		// Linux 系统
		tempDir = "/tmp"
		
		// 尝试使用系统目录，如果权限不足则回退到用户目录
		if canWrite("/var/log") {
			logDir = "/var/log/assistant_agent"
		} else {
			logDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "assistant_agent", "logs")
		}
		
		if canWrite("/var/lib") {
			workDir = "/var/lib/assistant_agent"
			dataDir = "/var/lib/assistant_agent"
		} else {
			workDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "assistant_agent", "work")
			dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "assistant_agent", "data")
		}
		
	case "darwin":
		// macOS 系统
		tempDir = "/tmp"
		
		// 尝试使用系统目录，如果权限不足则回退到用户目录
		if canWrite("/var/log") {
			logDir = "/var/log/assistant_agent"
		} else {
			logDir = filepath.Join(os.Getenv("HOME"), "Library", "Logs", "assistant_agent")
		}
		
		if canWrite("/Library/Application Support") {
			workDir = "/Library/Application Support/assistant_agent/work"
			dataDir = "/Library/Application Support/assistant_agent/data"
		} else {
			workDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "assistant_agent", "work")
			dataDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "assistant_agent", "data")
		}
		
	default:
		// 其他系统，使用用户目录
		tempDir = os.TempDir()
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "."
		}
		logDir = filepath.Join(homeDir, ".assistant_agent", "logs")
		workDir = filepath.Join(homeDir, ".assistant_agent", "work")
		dataDir = filepath.Join(homeDir, ".assistant_agent", "data")
	}
	
	return
}

// canWrite 检查目录是否可写
func canWrite(dir string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 目录不存在，尝试创建
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false
		}
	}
	
	// 尝试创建临时文件来测试写权限
	testFile := filepath.Join(dir, ".test_write")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

// Init 初始化配置
func Init() error {
	// 设置默认配置
	setDefaults()

	// 设置配置文件路径
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/assistant_agent")

	// 绑定环境变量
	viper.SetEnvPrefix("ASSISTANT_AGENT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// 配置文件不存在，使用默认配置
	}

	// 解析配置
	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return err
	}

	// 创建必要的目录
	if err := createDirectories(); err != nil {
		return err
	}

	return nil
}

// setDefaults 设置默认配置
func setDefaults() {
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.url", "ws://localhost:8080/ws")

	viper.SetDefault("agent.id", "")
	viper.SetDefault("agent.name", "assistant-agent")
	viper.SetDefault("agent.version", "1.0.0")
	viper.SetDefault("agent.heartbeat", 30)
	viper.SetDefault("agent.max_retries", 3)
	viper.SetDefault("agent.retry_delay", 5)
	viper.SetDefault("agent.container_mode", false)

	// 使用系统标准目录
	tempDir, logDir, workDir, dataDir := getSystemDirectories()
	viper.SetDefault("agent.temp_dir", tempDir)
	viper.SetDefault("agent.log_dir", logDir)
	viper.SetDefault("agent.work_dir", workDir)
	viper.SetDefault("agent.data_dir", dataDir)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.file", "assistant_agent.log")

	viper.SetDefault("security.token", "")
	viper.SetDefault("security.cert_file", "")
	viper.SetDefault("security.key_file", "")
	viper.SetDefault("security.verify_ssl", true)
}

// createDirectories 创建必要的目录
func createDirectories() error {
	dirs := []string{
		GlobalConfig.Agent.WorkDir,
		GlobalConfig.Agent.TempDir,
		GlobalConfig.Agent.LogDir,
		GlobalConfig.Agent.DataDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return GlobalConfig
}
