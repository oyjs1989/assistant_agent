package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	// 测试创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)
	assert.NotNil(t, collector)
}

func TestCollectorCollect(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集系统信息
	info, err := collector.Collect()
	require.NoError(t, err)
	assert.NotNil(t, info)

	// 验证基本信息
	assert.NotEmpty(t, info["hostname"])
	assert.NotEmpty(t, info["os"])
	assert.NotEmpty(t, info["architecture"])
	assert.NotEmpty(t, info["platform"])
	assert.NotEmpty(t, info["kernel"])
	assert.NotZero(t, info["uptime"])

	// 验证 CPU 信息
	cpuInfo, ok := info["cpu_info"].(CPUInfo)
	assert.True(t, ok)
	assert.Greater(t, cpuInfo.Cores, 0)
	assert.GreaterOrEqual(t, cpuInfo.Usage, 0.0)
	assert.LessOrEqual(t, cpuInfo.Usage, 100.0)

	// 验证内存信息
	memoryInfo, ok := info["memory_info"].(MemoryInfo)
	assert.True(t, ok)
	assert.Greater(t, memoryInfo.Total, uint64(0))
	assert.GreaterOrEqual(t, memoryInfo.Used, uint64(0))
	assert.GreaterOrEqual(t, memoryInfo.Available, uint64(0))
	assert.GreaterOrEqual(t, memoryInfo.Usage, 0.0)
	assert.LessOrEqual(t, memoryInfo.Usage, 100.0)

	// 验证磁盘信息
	diskInfo, ok := info["disk_info"].(DiskInfo)
	assert.True(t, ok)
	assert.Greater(t, diskInfo.Total, uint64(0))
	assert.GreaterOrEqual(t, diskInfo.Used, uint64(0))
	assert.GreaterOrEqual(t, diskInfo.Free, uint64(0))
	assert.GreaterOrEqual(t, diskInfo.Usage, 0.0)
	assert.LessOrEqual(t, diskInfo.Usage, 100.0)

	// 验证网络信息
	networkInfo, ok := info["network_info"].(NetworkInfo)
	assert.True(t, ok)
	assert.NotNil(t, networkInfo.Interfaces)
}

func TestCollectorCollectBasicInfo(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集基本信息
	info := &SystemInfo{}
	err = collector.collectBasicInfo(info)
	require.NoError(t, err)

	// 验证基本信息
	assert.NotEmpty(t, info.Hostname)
	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.Architecture)
	assert.NotEmpty(t, info.Platform)
	assert.NotEmpty(t, info.Kernel)
	assert.GreaterOrEqual(t, info.Uptime, 0.0)
}

func TestCollectorCollectCPUInfo(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集 CPU 信息
	info := &SystemInfo{}
	err = collector.collectCPUInfo(info)
	require.NoError(t, err)

	// 验证 CPU 信息
	assert.Greater(t, info.CPU.Cores, 0)
	assert.GreaterOrEqual(t, info.CPU.Usage, 0.0)
	assert.LessOrEqual(t, info.CPU.Usage, 100.0)
}

func TestCollectorCollectMemoryInfo(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集内存信息
	info := &SystemInfo{}
	err = collector.collectMemoryInfo(info)
	require.NoError(t, err)

	// 验证内存信息
	assert.Greater(t, info.Memory.Total, uint64(0))
	assert.GreaterOrEqual(t, info.Memory.Used, uint64(0))
	assert.GreaterOrEqual(t, info.Memory.Available, uint64(0))
	assert.GreaterOrEqual(t, info.Memory.Usage, 0.0)
	assert.LessOrEqual(t, info.Memory.Usage, 100.0)
}

func TestCollectorCollectDiskInfo(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集磁盘信息
	info := &SystemInfo{}
	err = collector.collectDiskInfo(info)
	require.NoError(t, err)

	// 验证磁盘信息
	assert.Greater(t, info.Disk.Total, uint64(0))
	assert.GreaterOrEqual(t, info.Disk.Used, uint64(0))
	assert.GreaterOrEqual(t, info.Disk.Free, uint64(0))
	assert.GreaterOrEqual(t, info.Disk.Usage, 0.0)
	assert.LessOrEqual(t, info.Disk.Usage, 100.0)
}

func TestCollectorCollectNetworkInfo(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集网络信息
	info := &SystemInfo{}
	err = collector.collectNetworkInfo(info)
	require.NoError(t, err)

	// 验证网络信息
	assert.NotNil(t, info.Network.Interfaces)
}

func TestCollectorMultipleCollections(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 多次收集系统信息
	for i := 0; i < 3; i++ {
		info, err := collector.Collect()
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.NotEmpty(t, info["hostname"])

		cpuInfo, ok := info["cpu_info"].(CPUInfo)
		assert.True(t, ok)
		assert.Greater(t, cpuInfo.Cores, 0)
	}
}

func TestCollectorConcurrentCollection(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 并发收集系统信息
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			info, err := collector.Collect()
			assert.NoError(t, err)
			assert.NotNil(t, info)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestCollectorSystemInfoValidation(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 收集系统信息
	info, err := collector.Collect()
	require.NoError(t, err)

	// 验证系统信息的完整性
	assert.NotNil(t, info)
	assert.NotEmpty(t, info["hostname"])
	assert.NotEmpty(t, info["os"])
	assert.NotEmpty(t, info["architecture"])

	// 验证 CPU 信息
	cpuInfo, ok := info["cpu_info"].(CPUInfo)
	assert.True(t, ok)
	assert.Greater(t, cpuInfo.Cores, 0)
	assert.GreaterOrEqual(t, cpuInfo.Usage, 0.0)
	assert.LessOrEqual(t, cpuInfo.Usage, 100.0)

	// 验证内存信息
	memoryInfo, ok := info["memory_info"].(MemoryInfo)
	assert.True(t, ok)
	assert.Greater(t, memoryInfo.Total, uint64(0))
	assert.GreaterOrEqual(t, memoryInfo.Used, uint64(0))
	assert.GreaterOrEqual(t, memoryInfo.Available, uint64(0))
	assert.GreaterOrEqual(t, memoryInfo.Usage, 0.0)
	assert.LessOrEqual(t, memoryInfo.Usage, 100.0)

	// 验证磁盘信息
	diskInfo, ok := info["disk_info"].(DiskInfo)
	assert.True(t, ok)
	assert.Greater(t, diskInfo.Total, uint64(0))
	assert.GreaterOrEqual(t, diskInfo.Used, uint64(0))
	assert.GreaterOrEqual(t, diskInfo.Free, uint64(0))
	assert.GreaterOrEqual(t, diskInfo.Usage, 0.0)
	assert.LessOrEqual(t, diskInfo.Usage, 100.0)

	// 验证网络信息
	networkInfo, ok := info["network_info"].(NetworkInfo)
	assert.True(t, ok)
	assert.NotNil(t, networkInfo.Interfaces)
}

func TestCollectorErrorHandling(t *testing.T) {
	// 创建收集器
	collector, err := NewCollector()
	require.NoError(t, err)

	// 测试收集器在错误情况下的行为
	// 这里主要测试收集器不会因为部分信息收集失败而完全失败
	info := &SystemInfo{}

	// 即使某个收集方法失败，其他方法仍应正常工作
	collector.collectBasicInfo(info)
	collector.collectCPUInfo(info)
	collector.collectMemoryInfo(info)
	collector.collectDiskInfo(info)
	collector.collectNetworkInfo(info)

	// 验证至少有一些基本信息被收集
	assert.NotEmpty(t, info.Hostname)
	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.Architecture)
}
