package logger

import (
	"os"
	"path/filepath"

	"assistant_agent/internal/config"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init 初始化日志
func Init() error {
	log = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(config.GetConfig().Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// 设置日志格式
	if config.GetConfig().Logging.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// 设置日志文件
	if config.GetConfig().Logging.File != "" {
		logFile := filepath.Join(config.GetConfig().Agent.LogDir, config.GetConfig().Logging.File)
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stdout)
	}

	return nil
}

// Debug 调试日志
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Info 信息日志
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Warn 警告日志
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

// Error 错误日志
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Fatal 致命错误日志
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// WithField 添加字段
func WithField(key string, value interface{}) *logrus.Entry {
	return log.WithField(key, value)
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	return log.WithFields(fields)
} 