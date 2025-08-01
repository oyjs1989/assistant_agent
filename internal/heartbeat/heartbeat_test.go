package heartbeat

import (
	"testing"
	"time"

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

func TestNew(t *testing.T) {
	// 测试创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)
	assert.NotNil(t, heartbeat)
	assert.Equal(t, interval, heartbeat.interval)
	assert.True(t, heartbeat.healthy)
}

func TestHeartbeatBeat(t *testing.T) {
	// 创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 等待一小段时间确保时间差异
	time.Sleep(10 * time.Millisecond)

	// 记录当前时间
	beforeBeat := time.Now()

	// 发送心跳
	heartbeat.Beat()

	// 验证心跳时间已更新
	assert.True(t, heartbeat.lastBeat.After(beforeBeat) || heartbeat.lastBeat.Equal(beforeBeat))
}

func TestHeartbeatIsHealthy(t *testing.T) {
	// 创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 初始状态应该是健康的
	assert.True(t, heartbeat.IsHealthy())

	// 发送心跳
	heartbeat.Beat()

	// 发送心跳后应该仍然是健康的
	assert.True(t, heartbeat.IsHealthy())
}

func TestHeartbeatIsHealthyAfterInterval(t *testing.T) {
	// 创建心跳检测器，使用较短的间隔进行测试
	interval := 1 // 1秒
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 立即检查应该是健康的
	assert.True(t, heartbeat.IsHealthy())

	// 等待超过2倍间隔时间（因为IsHealthy检查的是2*interval）
	time.Sleep(2500 * time.Millisecond)

	// 现在应该是不健康的
	assert.False(t, heartbeat.IsHealthy())
}

func TestHeartbeatIsHealthyAfterDoubleInterval(t *testing.T) {
	// 创建心跳检测器，使用较短的间隔进行测试
	interval := 1 // 1秒
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 等待超过两倍间隔时间
	time.Sleep(2500 * time.Millisecond)

	// 应该是不健康的
	assert.False(t, heartbeat.IsHealthy())
}

func TestHeartbeatGetLastBeat(t *testing.T) {
	// 创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 初始时应该有当前时间
	initialBeat := heartbeat.GetLastBeat()
	assert.NotZero(t, initialBeat)

	// 发送心跳
	heartbeat.Beat()

	// 现在应该有最后心跳时间
	lastBeat := heartbeat.GetLastBeat()
	assert.NotZero(t, lastBeat)
	assert.True(t, lastBeat.After(time.Time{}))
}

func TestHeartbeatGetInterval(t *testing.T) {
	// 创建心跳检测器
	interval := 45
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 验证间隔时间
	assert.Equal(t, interval, heartbeat.GetInterval())
}

func TestHeartbeatMultipleBeats(t *testing.T) {
	// 创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 多次发送心跳
	for i := 0; i < 5; i++ {
		heartbeat.Beat()
		time.Sleep(10 * time.Millisecond)
	}

	// 应该保持健康状态
	assert.True(t, heartbeat.IsHealthy())

	// 最后心跳时间应该是最新的
	lastBeat := heartbeat.GetLastBeat()
	assert.True(t, lastBeat.After(time.Time{}))
}

func TestHeartbeatConcurrentBeats(t *testing.T) {
	// 创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 并发发送心跳
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			heartbeat.Beat()
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 应该保持健康状态
	assert.True(t, heartbeat.IsHealthy())
}

func TestHeartbeatZeroInterval(t *testing.T) {
	// 测试零间隔的情况
	interval := 0
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 零间隔时应该总是健康的
	assert.True(t, heartbeat.IsHealthy())
}

func TestHeartbeatNegativeInterval(t *testing.T) {
	// 测试负间隔的情况
	interval := -10
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 负间隔时应该总是健康的（因为负间隔乘以2仍然是负数，time.Since总是正数）
	assert.True(t, heartbeat.IsHealthy())

	// 等待一段时间后仍然应该是健康的
	time.Sleep(100 * time.Millisecond)
	assert.True(t, heartbeat.IsHealthy())
}

func TestHeartbeatVeryShortInterval(t *testing.T) {
	// 测试非常短的间隔
	interval := 1 // 1秒
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 立即检查应该是健康的
	assert.True(t, heartbeat.IsHealthy())

	// 等待超过2倍间隔时间
	time.Sleep(2500 * time.Millisecond)

	// 现在应该是不健康的
	assert.False(t, heartbeat.IsHealthy())
}

func TestHeartbeatVeryLongInterval(t *testing.T) {
	// 测试非常长的间隔
	interval := 86400 // 24小时（秒）
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 应该保持健康状态
	assert.True(t, heartbeat.IsHealthy())

	// 等待一段时间（但不超过间隔）
	time.Sleep(100 * time.Millisecond)

	// 应该仍然是健康的
	assert.True(t, heartbeat.IsHealthy())
}

func TestHeartbeatReset(t *testing.T) {
	// 创建心跳检测器
	interval := 30
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()
	assert.True(t, heartbeat.IsHealthy())

	// 重置心跳检测器
	heartbeat, err = New(interval)
	require.NoError(t, err)

	// 重置后应该回到初始状态
	assert.True(t, heartbeat.healthy)
}

func TestHeartbeatEdgeCase(t *testing.T) {
	// 测试边界情况
	interval := 1 // 1秒
	heartbeat, err := New(interval)
	require.NoError(t, err)

	// 发送心跳
	heartbeat.Beat()

	// 等待恰好等于间隔的时间
	time.Sleep(time.Duration(interval) * time.Second)

	// 应该仍然健康（因为检查是 2*interval）
	assert.True(t, heartbeat.IsHealthy())

	// 等待超过 2*interval 的时间
	time.Sleep(time.Duration(interval) * time.Second)

	// 现在应该是不健康的
	assert.False(t, heartbeat.IsHealthy())
} 