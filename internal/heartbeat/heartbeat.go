package heartbeat

import (
	"time"

	"assistant_agent/internal/logger"
)

// Heartbeat 心跳检测器
type Heartbeat struct {
	interval int
	lastBeat time.Time
	healthy  bool
}

// New 创建新的心跳检测器
func New(interval int) (*Heartbeat, error) {
	return &Heartbeat{
		interval: interval,
		lastBeat: time.Now(),
		healthy:  true,
	}, nil
}

// Beat 发送心跳
func (h *Heartbeat) Beat() {
	h.lastBeat = time.Now()
	h.healthy = true
	logger.Debug("Heartbeat sent")
}

// IsHealthy 检查是否健康
func (h *Heartbeat) IsHealthy() bool {
	// 如果间隔为负数或零，则总是健康的
	if h.interval <= 0 {
		return true
	}
	
	// 如果超过心跳间隔的2倍时间没有心跳，则认为不健康
	if time.Since(h.lastBeat) > time.Duration(h.interval*2)*time.Second {
		h.healthy = false
	}
	return h.healthy
}

// GetLastBeat 获取最后心跳时间
func (h *Heartbeat) GetLastBeat() time.Time {
	return h.lastBeat
}

// GetInterval 获取心跳间隔
func (h *Heartbeat) GetInterval() int {
	return h.interval
}

// Stop 停止心跳
func (h *Heartbeat) Stop() {
	h.healthy = false
	logger.Debug("Heartbeat stopped")
}

// Send 发送心跳（别名方法）
func (h *Heartbeat) Send() {
	h.Beat()
} 