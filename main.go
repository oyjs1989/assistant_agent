package main

import (
	"os"
	"os/signal"
	"syscall"

	"assistant_agent/internal/agent"
	"assistant_agent/internal/config"
	"assistant_agent/internal/logger"

	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化配置
	if err := config.Init(); err != nil {
		logrus.Fatalf("Failed to initialize config: %v", err)
	}

	// 初始化日志
	if err := logger.Init(); err != nil {
		logrus.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Info("Assistant Agent starting...")

	// 创建并启动 agent
	a, err := agent.New()
	if err != nil {
		logger.Fatalf("Failed to create agent: %v", err)
	}

	// 启动 agent
	if err := a.Start(); err != nil {
		logger.Fatalf("Failed to start agent: %v", err)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Assistant Agent...")
	a.Stop()
	logger.Info("Assistant Agent stopped")
} 