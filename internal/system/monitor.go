package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

// Monitor collects system metrics directly from Linux procfs and syscalls.
type Monitor struct {
	runner domain.AgentRunner
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(runner domain.AgentRunner) *Monitor {
	return &Monitor{runner: runner}
}

// Collect reads and formats host system metrics.
func (m *Monitor) Collect() (*domain.SystemStatus, error) {
	uptimeStr := m.getUptime()
	cpuModel := m.getCPUModel()
	memUsage, swapUsage := m.getMemoryAndSwap()
	diskUsage := m.getDiskUsage("/")

	agentStatus := "Idle 🟢"
	if m.runner != nil && m.runner.IsRunning() {
		agentStatus = "Processing request 🟡"
	}

	return &domain.SystemStatus{
		Uptime:      uptimeStr,
		CPUModel:    cpuModel,
		MemoryUsage: memUsage,
		SwapUsage:   swapUsage,
		DiskUsage:   diskUsage,
		AgentStatus: agentStatus,
	}, nil
}

func (m *Monitor) getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "unknown"
	}

	totalSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "unknown"
	}

	d := time.Duration(totalSeconds) * time.Second
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func (m *Monitor) getCPUModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Unknown CPU"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown CPU"
}

func (m *Monitor) getMemoryAndSwap() (string, string) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return "unknown", "unknown"
	}
	defer file.Close()

	var memTotal, memAvailable, swapTotal, swapFree uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)

		switch key {
		case "MemTotal":
			memTotal = val
		case "MemAvailable":
			memAvailable = val
		case "SwapTotal":
			swapTotal = val
		case "SwapFree":
			swapFree = val
		}
	}

	memUsed := memTotal - memAvailable
	memUsage := fmt.Sprintf("%d / %d MB (%.1f%%)",
		memUsed/1024, memTotal/1024,
		float64(memUsed)/float64(memTotal)*100,
	)

	swapUsed := swapTotal - swapFree
	var swapPct float64
	if swapTotal > 0 {
		swapPct = float64(swapUsed) / float64(swapTotal) * 100
	}
	swapUsage := fmt.Sprintf("%d / %d MB (%.1f%%)",
		swapUsed/1024, swapTotal/1024, swapPct,
	)

	return memUsage, swapUsage
}

func (m *Monitor) getDiskUsage(path string) string {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "unknown"
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	toGB := float64(1024 * 1024 * 1024)
	return fmt.Sprintf("%.1f / %.1f GB (%.1f%%)",
		float64(used)/toGB,
		float64(total)/toGB,
		float64(used)/float64(total)*100,
	)
}

var _ domain.MetricsCollector = (*Monitor)(nil)
