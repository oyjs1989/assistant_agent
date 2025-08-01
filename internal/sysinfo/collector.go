package sysinfo

import (
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemInfo 系统信息结构（简化版）
type SystemInfo struct {
	// 基本信息
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Platform     string `json:"platform"`
	Kernel       string `json:"kernel"`

	// 硬件信息
	CPU    CPUInfo    `json:"cpu"`
	Memory MemoryInfo `json:"memory"`
	Disk   DiskInfo   `json:"disk"`

	// 网络信息
	Network NetworkInfo `json:"network"`

	// 系统状态
	Uptime      float64   `json:"uptime"`
	BootTime    time.Time `json:"boot_time"`
	Processes   int       `json:"processes"`
	LoadAverage []float64 `json:"load_average"`
}

// CPUInfo CPU 信息（简化版）
type CPUInfo struct {
	Model       string  `json:"model"`
	Cores       int     `json:"cores"`
	LogicalCPUs int     `json:"logical_cpus"`
	Usage       float64 `json:"usage"`
}

// MemoryInfo 内存信息（简化版）
type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
}

// DiskInfo 磁盘信息（简化版）
type DiskInfo struct {
	Total      uint64          `json:"total"`
	Used       uint64          `json:"used"`
	Free       uint64          `json:"free"`
	Usage      float64         `json:"usage"`
	Partitions []PartitionInfo `json:"partitions"`
}

// PartitionInfo 分区信息（简化版）
type PartitionInfo struct {
	Device     string  `json:"device"`
	MountPoint string  `json:"mount_point"`
	FileSystem string  `json:"file_system"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	Usage      float64 `json:"usage"`
}

// NetworkInfo 网络信息（简化版）
type NetworkInfo struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
}

// InterfaceInfo 网络接口信息（简化版）
type InterfaceInfo struct {
	Name       string   `json:"name"`
	Addresses  []string `json:"addresses"`
	MACAddress string   `json:"mac_address"`
	MTU        int      `json:"mtu"`
}

// Collector 系统信息收集器
type Collector struct {
	lastCPUUsage float64
	lastCPUTime  time.Time
}

// NewCollector 创建新的收集器
func NewCollector() (*Collector, error) {
	return &Collector{
		lastCPUTime: time.Now(),
	}, nil
}

// Collect 收集系统信息
func (c *Collector) Collect() (map[string]interface{}, error) {
	info := &SystemInfo{}

	// 收集基本信息
	if err := c.collectBasicInfo(info); err != nil {
		return nil, err
	}

	// 收集 CPU 信息
	if err := c.collectCPUInfo(info); err != nil {
		return nil, err
	}

	// 收集内存信息
	if err := c.collectMemoryInfo(info); err != nil {
		return nil, err
	}

	// 收集磁盘信息
	if err := c.collectDiskInfo(info); err != nil {
		return nil, err
	}

	// 收集网络信息
	if err := c.collectNetworkInfo(info); err != nil {
		return nil, err
	}

	// 转换为 map（简化输出）
	result := map[string]interface{}{
		"hostname":     info.Hostname,
		"os":           info.OS,
		"architecture": info.Architecture,
		"platform":     info.Platform,
		"kernel":       info.Kernel,
		"cpu_usage":    info.CPU.Usage,
		"memory_usage": info.Memory.Usage,
		"disk_usage":   info.Disk.Usage,
		"uptime":       info.Uptime,
		"processes":    info.Processes,
		"boot_time":    info.BootTime,
		"load_average": info.LoadAverage,
		"cpu_info":     info.CPU,
		"memory_info":  info.Memory,
		"disk_info":    info.Disk,
		"network_info": info.Network,
	}

	return result, nil
}

// collectBasicInfo 收集基本信息
func (c *Collector) collectBasicInfo(info *SystemInfo) error {
	// 主机名
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	info.Hostname = hostname

	// 操作系统信息
	info.OS = runtime.GOOS
	info.Architecture = runtime.GOARCH
	info.Platform = runtime.GOOS + "/" + runtime.GOARCH

	// 内核版本
	if kernel, err := c.getKernelVersion(); err == nil {
		info.Kernel = kernel
	}

	// 进程数
	if processes, err := c.getProcessCount(); err == nil {
		info.Processes = processes
	}

	// 系统启动时间
	if hostInfo, err := host.Info(); err == nil {
		info.Uptime = float64(hostInfo.Uptime)
		info.BootTime = time.Unix(int64(hostInfo.BootTime), 0)
	}

	// 负载平均值
	if loadAvg, err := c.getLoadAverage(); err == nil {
		info.LoadAverage = loadAvg
	}

	return nil
}

// collectCPUInfo 收集 CPU 信息
func (c *Collector) collectCPUInfo(info *SystemInfo) error {
	// CPU 使用率
	usage, err := cpu.Percent(0, false)
	if err != nil {
		return err
	}
	if len(usage) > 0 {
		info.CPU.Usage = usage[0]
	}

	// CPU 信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		return err
	}
	if len(cpuInfo) > 0 {
		info.CPU.Model = cpuInfo[0].ModelName
		info.CPU.Cores = int(cpuInfo[0].Cores)
		info.CPU.LogicalCPUs = runtime.NumCPU()
	}

	return nil
}

// collectMemoryInfo 收集内存信息
func (c *Collector) collectMemoryInfo(info *SystemInfo) error {
	// 虚拟内存
	if vmstat, err := mem.VirtualMemory(); err == nil {
		info.Memory.Total = vmstat.Total
		info.Memory.Used = vmstat.Used
		info.Memory.Free = vmstat.Free
		info.Memory.Available = vmstat.Available
		info.Memory.Usage = vmstat.UsedPercent
	}

	return nil
}

// collectDiskInfo 收集磁盘信息
func (c *Collector) collectDiskInfo(info *SystemInfo) error {
	// 磁盘使用情况
	if diskStat, err := disk.Usage("/"); err == nil {
		info.Disk.Total = diskStat.Total
		info.Disk.Used = diskStat.Used
		info.Disk.Free = diskStat.Free
		info.Disk.Usage = diskStat.UsedPercent
	}

	// 分区信息（只收集主要分区）
	if partitions, err := disk.Partitions(false); err == nil {
		for _, partition := range partitions {
			// 只收集根分区和主要数据分区
			if partition.Mountpoint == "/" ||
				partition.Mountpoint == "/home" ||
				partition.Mountpoint == "/data" ||
				partition.Mountpoint == "C:" ||
				partition.Mountpoint == "D:" {
				if usage, err := disk.Usage(partition.Mountpoint); err == nil {
					info.Disk.Partitions = append(info.Disk.Partitions, PartitionInfo{
						Device:     partition.Device,
						MountPoint: partition.Mountpoint,
						FileSystem: partition.Fstype,
						Total:      usage.Total,
						Used:       usage.Used,
						Free:       usage.Free,
						Usage:      usage.UsedPercent,
					})
				}
			}
		}
	}

	return nil
}

// collectNetworkInfo 收集网络信息
func (c *Collector) collectNetworkInfo(info *SystemInfo) error {
	// 网络接口
	if interfaces, err := net.Interfaces(); err == nil {
		for _, iface := range interfaces {
			// 只收集活跃的网络接口
			if len(iface.Addrs) > 0 {
				interfaceInfo := InterfaceInfo{
					Name:       iface.Name,
					MACAddress: iface.HardwareAddr,
					MTU:        iface.MTU,
				}

				// 获取 IP 地址
				for _, addr := range iface.Addrs {
					interfaceInfo.Addresses = append(interfaceInfo.Addresses, addr.Addr)
				}

				info.Network.Interfaces = append(info.Network.Interfaces, interfaceInfo)
			}
		}
	}

	return nil
}

// getKernelVersion 获取内核版本
func (c *Collector) getKernelVersion() (string, error) {
	// 这里可以实现获取内核版本的逻辑
	// 不同操作系统有不同的实现方式
	return runtime.GOOS, nil
}

// getProcessCount 获取进程数
func (c *Collector) getProcessCount() (int, error) {
	// 这里可以实现获取进程数的逻辑
	return 0, nil
}

// getLoadAverage 获取负载平均值
func (c *Collector) getLoadAverage() ([]float64, error) {
	// 这里可以实现获取负载平均值的逻辑
	return []float64{0, 0, 0}, nil
}
