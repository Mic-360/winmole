package services

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/pkg/util"
)

type RuntimeService struct {
	logger *Logger

	mu               sync.Mutex
	cpuHistory       []float64
	memoryHistory    []float64
	diskHistory      []float64
	networkHistory   []float64
	prevSent         uint64
	prevRecv         uint64
	cachedBuild      string
	cachedPowerShell string
	cachedGPUs       []state.GPUInfo
	cachedBattery    state.BatteryInfo
	cachedStaticAt   time.Time
	isAdmin          bool
}

func NewRuntimeService(logger *Logger) *RuntimeService {
	service := &RuntimeService{logger: logger, isAdmin: probeAdmin()}
	service.cachedPowerShell = probePowerShellVersion()
	service.cachedBuild = probeWindowsBuild()
	service.cachedGPUs = probeGPUs()
	service.cachedBattery = probeBattery()
	service.cachedStaticAt = time.Now()
	return service
}

func (r *RuntimeService) Snapshot(ctx context.Context) (state.RuntimeSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.cachedStaticAt) > 30*time.Second {
		r.cachedBuild = probeWindowsBuild()
		r.cachedPowerShell = probePowerShellVersion()
		r.cachedGPUs = probeGPUs()
		r.cachedBattery = probeBattery()
		r.cachedStaticAt = time.Now()
	}

	snapshot := state.RuntimeSnapshot{Timestamp: time.Now(), Build: r.cachedBuild, PowerShell: r.cachedPowerShell, GoVersion: runtime.Version(), GPUs: append([]state.GPUInfo{}, r.cachedGPUs...), Battery: r.cachedBattery, IsAdmin: r.isAdmin}

	if info, err := host.InfoWithContext(ctx); err == nil {
		snapshot.Hostname = info.Hostname
		snapshot.Platform = fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)
		snapshot.Uptime = time.Duration(info.Uptime) * time.Second
		snapshot.BootTime = time.Unix(int64(info.BootTime), 0)
	}

	if percents, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(percents) > 0 {
		snapshot.CPU = state.Metric{Label: "CPU", Current: percents[0], Max: 100, Unit: "%", Text: util.FormatPercent(percents[0])}
	}
	if totals, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		snapshot.Memory = state.Metric{Label: "Memory", Current: totals.UsedPercent, Max: 100, Unit: "%", Text: fmt.Sprintf("%s / %s", util.FormatBytes(int64(totals.Used)), util.FormatBytes(int64(totals.Total)))}
	}
	if usage, err := disk.UsageWithContext(ctx, `C:\`); err == nil {
		snapshot.Disk = state.Metric{Label: "System Disk", Current: usage.UsedPercent, Max: 100, Unit: "%", Text: fmt.Sprintf("%s free", util.FormatBytes(int64(usage.Free)))}
	}

	if partitions, err := disk.PartitionsWithContext(ctx, false); err == nil {
		for _, partition := range partitions {
			usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
			if err != nil || usage.Total == 0 {
				continue
			}
			snapshot.Disks = append(snapshot.Disks, state.DiskInfo{Mount: partition.Mountpoint, FSType: partition.Fstype, Used: int64(usage.Used), Free: int64(usage.Free), Total: int64(usage.Total), Percent: usage.UsedPercent})
		}
	}

	if counters, err := gnet.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		counter := counters[0]
		upRate := 0.0
		downRate := 0.0
		if r.prevSent > 0 {
			upRate = float64(counter.BytesSent-r.prevSent) / 3.0
			downRate = float64(counter.BytesRecv-r.prevRecv) / 3.0
		}
		r.prevSent = counter.BytesSent
		r.prevRecv = counter.BytesRecv
		snapshot.Network = state.NetworkInfo{SentBytes: int64(counter.BytesSent), RecvBytes: int64(counter.BytesRecv), UploadRate: upRate, DownloadRate: downRate}
	}
	if perIface, err := gnet.IOCountersWithContext(ctx, true); err == nil {
		for _, iface := range perIface {
			if iface.BytesSent == 0 && iface.BytesRecv == 0 {
				continue
			}
			snapshot.Network.InterfaceSummaries = append(snapshot.Network.InterfaceSummaries, state.InterfaceSummary{Name: iface.Name, SentBytes: int64(iface.BytesSent), RecvBytes: int64(iface.BytesRecv)})
		}
		sort.Slice(snapshot.Network.InterfaceSummaries, func(i, j int) bool {
			left := snapshot.Network.InterfaceSummaries[i].SentBytes + snapshot.Network.InterfaceSummaries[i].RecvBytes
			right := snapshot.Network.InterfaceSummaries[j].SentBytes + snapshot.Network.InterfaceSummaries[j].RecvBytes
			return left > right
		})
		if len(snapshot.Network.InterfaceSummaries) > 4 {
			snapshot.Network.InterfaceSummaries = snapshot.Network.InterfaceSummaries[:4]
		}
	}

	if procs, err := process.ProcessesWithContext(ctx); err == nil {
		items := make([]state.ProcessInfo, 0, len(procs))
		for _, proc := range procs {
			name, err := proc.NameWithContext(ctx)
			if err != nil || strings.TrimSpace(name) == "" {
				continue
			}
			cpuPercent, _ := proc.CPUPercentWithContext(ctx)
			memPercent, _ := proc.MemoryPercentWithContext(ctx)
			items = append(items, state.ProcessInfo{Name: name, CPUPercent: cpuPercent, MemPercent: float64(memPercent)})
		}
		sort.Slice(items, func(i, j int) bool {
			left := items[i].CPUPercent + items[i].MemPercent
			right := items[j].CPUPercent + items[j].MemPercent
			return left > right
		})
		if len(items) > 7 {
			items = items[:7]
		}
		snapshot.Processes = items
	}

	r.cpuHistory = appendHistory(r.cpuHistory, snapshot.CPU.Current)
	r.memoryHistory = appendHistory(r.memoryHistory, snapshot.Memory.Current)
	r.diskHistory = appendHistory(r.diskHistory, snapshot.Disk.Current)
	r.networkHistory = appendHistory(r.networkHistory, max(snapshot.Network.UploadRate, snapshot.Network.DownloadRate)/(1024*1024))
	snapshot.CPU.History = append([]float64{}, r.cpuHistory...)
	snapshot.Memory.History = append([]float64{}, r.memoryHistory...)
	snapshot.Disk.History = append([]float64{}, r.diskHistory...)
	snapshot.Network.DownloadRate = util.Round(snapshot.Network.DownloadRate, 1)
	snapshot.Network.UploadRate = util.Round(snapshot.Network.UploadRate, 1)
	snapshot.Health = healthScore(snapshot)
	snapshot.Alerts = buildAlerts(snapshot)
	return snapshot, nil
}

func healthScore(snapshot state.RuntimeSnapshot) int {
	score := 100
	if snapshot.CPU.Current > 90 {
		score -= 28
	} else if snapshot.CPU.Current > 75 {
		score -= 18
	}
	if snapshot.Memory.Current > 90 {
		score -= 28
	} else if snapshot.Memory.Current > 80 {
		score -= 15
	}
	if snapshot.Disk.Current > 92 {
		score -= 24
	} else if snapshot.Disk.Current > 80 {
		score -= 12
	}
	if snapshot.Network.DownloadRate > 150*1024*1024 || snapshot.Network.UploadRate > 150*1024*1024 {
		score -= 8
	}
	if score < 0 {
		return 0
	}
	return score
}

func buildAlerts(snapshot state.RuntimeSnapshot) []string {
	alerts := []string{}
	if snapshot.CPU.Current > 85 {
		alerts = append(alerts, "CPU is running hot")
	}
	if snapshot.Memory.Current > 85 {
		alerts = append(alerts, "Memory pressure is elevated")
	}
	if snapshot.Disk.Current > 88 {
		alerts = append(alerts, "System drive is close to full")
	}
	if !snapshot.IsAdmin {
		alerts = append(alerts, "Admin-only tasks will be previewed but not executed")
	}
	if len(alerts) == 0 {
		alerts = append(alerts, "System health is stable")
	}
	return alerts
}

func appendHistory(history []float64, value float64) []float64 {
	history = append(history, value)
	if len(history) > 24 {
		history = append([]float64{}, history[len(history)-24:]...)
	}
	return history
}

func probePowerShellVersion() string {
	output, err := exec.Command("powershell", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func probeWindowsBuild() string {
	output, err := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_OperatingSystem).BuildNumber").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func probeGPUs() []state.GPUInfo {
	script := "Get-CimInstance Win32_VideoController -ErrorAction SilentlyContinue | ForEach-Object { \"{0}|{1}|{2}\" -f $_.Name, $_.AdapterRAM, $_.DriverVersion }"
	output, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	gpus := make([]state.GPUInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		vram, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		gpus = append(gpus, state.GPUInfo{Name: strings.TrimSpace(parts[0]), VRAM: util.FormatBytes(vram), Driver: strings.TrimSpace(parts[2])})
	}
	return gpus
}

func probeBattery() state.BatteryInfo {
	script := "$battery = Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue | Select-Object -First 1 EstimatedChargeRemaining, BatteryStatus; if ($null -eq $battery) { '' } else { \"{0}|{1}\" -f $battery.EstimatedChargeRemaining, $battery.BatteryStatus }"
	output, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return state.BatteryInfo{}
	}
	line := strings.TrimSpace(string(output))
	if line == "" {
		return state.BatteryInfo{}
	}
	parts := strings.Split(line, "|")
	if len(parts) != 2 {
		return state.BatteryInfo{}
	}
	percent, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	statusCode, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	status := "Connected"
	charging := false
	switch statusCode {
	case 6, 7, 8, 9:
		status = "Charging"
		charging = true
	case 3:
		status = "Full"
	case 1, 4, 5, 11:
		status = "Discharging"
	}
	return state.BatteryInfo{Present: true, Percent: util.Clamp(percent, 0, 100), Charging: charging, Status: status}
}

func probeAdmin() bool {
	command := exec.Command("powershell", "-NoProfile", "-Command", "([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)")
	output, err := command.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(output)), "True")
}

func max(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
