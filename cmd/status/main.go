package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

const (
	version       = "1.0.0"
	brandName     = "🐹 WiMo Status"
	refreshRate   = 2 * time.Second
	gaugeWidth    = 14
	maxProcesses  = 5
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	greenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	redStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type tickMsg time.Time

type systemStats struct {
	// CPU
	cpuPercent  float64
	cpuModel    string
	cpuCores    int
	cpuThreads  int
	cpuFreq     float64

	// Memory
	memTotal    uint64
	memUsed     uint64
	memFree     uint64
	memPercent  float64
	swapTotal   uint64
	swapUsed    uint64

	// Disk
	diskTotal   uint64
	diskUsed    uint64
	diskFree    uint64
	diskPercent float64
	diskLetter  string

	// Network
	netSent     uint64
	netRecv     uint64
	prevSent    uint64
	prevRecv    uint64
	netUpRate   float64
	netDownRate float64

	// System
	hostname    string
	osVersion   string
	uptime      uint64

	// Top processes
	topProcs    []procInfo

	// Health score
	health      int
}

type procInfo struct {
	name       string
	cpuPercent float64
	memPercent float32
}

type model struct {
	stats  systemStats
	width  int
	height int
	err    error
}

func initialModel() model {
	return model{
		width:  80,
		height: 24,
		stats: systemStats{
			diskLetter: "C:",
		},
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func gatherStats(prev systemStats) systemStats {
	s := systemStats{
		prevSent: prev.netSent,
		prevRecv: prev.netRecv,
	}

	// CPU
	percents, err := cpu.Percent(time.Second, false)
	if err == nil && len(percents) > 0 {
		s.cpuPercent = percents[0]
	}

	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		s.cpuModel = cpuInfo[0].ModelName
		s.cpuFreq = cpuInfo[0].Mhz
	}
	s.cpuCores = runtime.NumCPU()
	counts, err := cpu.Counts(true)
	if err == nil {
		s.cpuThreads = counts
	}

	// Memory
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		s.memTotal = memInfo.Total
		s.memUsed = memInfo.Used
		s.memFree = memInfo.Free
		s.memPercent = memInfo.UsedPercent
	}

	swapInfo, err := mem.SwapMemory()
	if err == nil {
		s.swapTotal = swapInfo.Total
		s.swapUsed = swapInfo.Used
	}

	// Disk
	diskUsage, err := disk.Usage("C:\\")
	if err == nil {
		s.diskTotal = diskUsage.Total
		s.diskUsed = diskUsage.Used
		s.diskFree = diskUsage.Free
		s.diskPercent = diskUsage.UsedPercent
		s.diskLetter = "C:"
	}

	// Network
	netIO, err := net.IOCounters(false)
	if err == nil && len(netIO) > 0 {
		s.netSent = netIO[0].BytesSent
		s.netRecv = netIO[0].BytesRecv

		if s.prevSent > 0 {
			s.netUpRate = float64(s.netSent-s.prevSent) / refreshRate.Seconds()
			s.netDownRate = float64(s.netRecv-s.prevRecv) / refreshRate.Seconds()
		}
	}

	// System
	hostInfo, err := host.Info()
	if err == nil {
		s.hostname = hostInfo.Hostname
		s.osVersion = fmt.Sprintf("%s %s", hostInfo.Platform, hostInfo.PlatformVersion)
		s.uptime = hostInfo.Uptime
	}

	// Top processes
	procs, err := process.Processes()
	if err == nil {
		var procList []procInfo
		for _, p := range procs {
			name, _ := p.Name()
			cpuPct, _ := p.CPUPercent()
			memPct, _ := p.MemoryPercent()

			if cpuPct > 0 || memPct > 0 {
				procList = append(procList, procInfo{
					name:       name,
					cpuPercent: cpuPct,
					memPercent: memPct,
				})
			}
		}

		// Sort by CPU usage
		sort.Slice(procList, func(i, j int) bool {
			return procList[i].cpuPercent > procList[j].cpuPercent
		})

		if len(procList) > maxProcesses {
			procList = procList[:maxProcesses]
		}
		s.topProcs = procList
	}

	// Health score
	s.health = calculateHealth(s)

	return s
}

func calculateHealth(s systemStats) int {
	score := 100

	// CPU penalty
	if s.cpuPercent > 90 {
		score -= 30
	} else if s.cpuPercent > 80 {
		score -= 20
	} else if s.cpuPercent > 60 {
		score -= 10
	}

	// Memory penalty
	if s.memPercent > 90 {
		score -= 30
	} else if s.memPercent > 85 {
		score -= 20
	} else if s.memPercent > 70 {
		score -= 10
	}

	// Disk penalty
	if s.diskPercent > 95 {
		score -= 30
	} else if s.diskPercent > 90 {
		score -= 20
	} else if s.diskPercent > 80 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}
	return score
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return tickMsg(time.Now())
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.stats = gatherStats(m.stats)
		return m, tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.stats = gatherStats(m.stats)
			return m, tick()
		}
	}

	return m, nil
}

func renderGauge(percent float64, width int) string {
	filled := int(math.Round(percent / 100 * float64(width)))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	empty := width - filled

	var colorFn func(strs ...string) string
	if percent < 60 {
		colorFn = greenStyle.Render
	} else if percent < 80 {
		colorFn = yellowStyle.Render
	} else {
		colorFn = redStyle.Render
	}

	return colorFn(strings.Repeat("█", filled)) + dimStyle.Render(strings.Repeat("░", empty))
}

func renderProcBar(percent float64, width int) string {
	filled := int(math.Round(percent / 100 * float64(width)))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	empty := width - filled
	return greenStyle.Render(strings.Repeat("█", filled)) + dimStyle.Render(strings.Repeat("░", empty))
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatRate(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	} else {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	}
}

func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func healthColor(health int) func(strs ...string) string {
	if health >= 80 {
		return greenStyle.Render
	} else if health >= 60 {
		return yellowStyle.Render
	}
	return redStyle.Render
}

func (m model) View() string {
	s := m.stats
	var b strings.Builder

	w := m.width
	if w < 40 {
		w = 80
	}
	halfW := (w - 5) / 2

	// Top border
	b.WriteString(borderStyle.Render("╭" + strings.Repeat("─", w-2) + "╮") + "\n")

	// Header
	healthStr := healthColor(s.health)(fmt.Sprintf("● %d", s.health))
	header := fmt.Sprintf("  %s  ·  Health %s  ·  %s  ·  %s",
		titleStyle.Render(brandName), healthStr,
		dimStyle.Render(s.hostname),
		dimStyle.Render(s.osVersion))
	b.WriteString(borderStyle.Render("│") + header + "\n")

	// Mid separator
	mid := borderStyle.Render("├" + strings.Repeat("─", halfW) + "┬" + strings.Repeat("─", w-halfW-3) + "┤")
	b.WriteString(mid + "\n")

	// CPU section
	cpuLabel := headerStyle.Render("  ⚙  CPU")
	memLabel := headerStyle.Render("▦  Memory")
	b.WriteString(borderStyle.Render("│") + cpuLabel +
		strings.Repeat(" ", maxInt(1, halfW-10)) + borderStyle.Render("│") +
		" " + memLabel + "\n")

	cpuGauge := fmt.Sprintf("  Total  %s  %.1f%%", renderGauge(s.cpuPercent, gaugeWidth), s.cpuPercent)
	memGauge := fmt.Sprintf(" Used   %s  %s/%s", renderGauge(s.memPercent, gaugeWidth),
		formatBytes(s.memUsed), formatBytes(s.memTotal))
	b.WriteString(borderStyle.Render("│") + cpuGauge +
		strings.Repeat(" ", maxInt(1, halfW-lipgloss.Width(cpuGauge))) + borderStyle.Render("│") +
		memGauge + "\n")

	cpuInfo := fmt.Sprintf("  Cores  %.1f GHz · %d threads", s.cpuFreq/1000, s.cpuThreads)
	memFree := fmt.Sprintf(" Free   %s  %s", renderGauge(100-s.memPercent, gaugeWidth), formatBytes(s.memFree))
	b.WriteString(borderStyle.Render("│") + labelStyle.Render(cpuInfo) +
		strings.Repeat(" ", maxInt(1, halfW-len(cpuInfo))) + borderStyle.Render("│") +
		memFree + "\n")

	swapPct := float64(0)
	if s.swapTotal > 0 {
		swapPct = float64(s.swapUsed) / float64(s.swapTotal) * 100
	}
	swapLine := fmt.Sprintf(" Swap   %s  %s", renderGauge(swapPct, gaugeWidth), formatBytes(s.swapUsed))
	b.WriteString(borderStyle.Render("│") +
		strings.Repeat(" ", halfW) + borderStyle.Render("│") +
		swapLine + "\n")

	// Mid separator
	mid2 := borderStyle.Render("├" + strings.Repeat("─", halfW) + "┼" + strings.Repeat("─", w-halfW-3) + "┤")
	b.WriteString(mid2 + "\n")

	// Disk + Power section
	diskLabel := headerStyle.Render(fmt.Sprintf("  ▤  Disk (%s)", s.diskLetter))
	netLabel := headerStyle.Render("⇅  Network")
	b.WriteString(borderStyle.Render("│") + diskLabel +
		strings.Repeat(" ", maxInt(1, halfW-lipgloss.Width(diskLabel))) + borderStyle.Render("│") +
		" " + netLabel + "\n")

	diskGauge := fmt.Sprintf("  Used   %s  %.1f%%", renderGauge(s.diskPercent, gaugeWidth), s.diskPercent)
	netDown := fmt.Sprintf(" Down  %s", formatRate(s.netDownRate))
	b.WriteString(borderStyle.Render("│") + diskGauge +
		strings.Repeat(" ", maxInt(1, halfW-lipgloss.Width(diskGauge))) + borderStyle.Render("│") +
		" " + netDown + "\n")

	diskFree := fmt.Sprintf("  Free   %s", formatBytes(s.diskFree))
	netUp := fmt.Sprintf(" Up    %s", formatRate(s.netUpRate))
	b.WriteString(borderStyle.Render("│") + labelStyle.Render(diskFree) +
		strings.Repeat(" ", maxInt(1, halfW-len(diskFree))) + borderStyle.Render("│") +
		" " + netUp + "\n")

	// Mid separator
	mid3 := borderStyle.Render("├" + strings.Repeat("─", halfW) + "┼" + strings.Repeat("─", w-halfW-3) + "┤")
	b.WriteString(mid3 + "\n")

	// Top Processes + System Info
	procLabel := headerStyle.Render("  ▶  Top Processes")
	sysLabel := headerStyle.Render("ℹ  System")
	b.WriteString(borderStyle.Render("│") + procLabel +
		strings.Repeat(" ", maxInt(1, halfW-lipgloss.Width(procLabel))) + borderStyle.Render("│") +
		" " + sysLabel + "\n")

	for i := 0; i < maxProcesses; i++ {
		var procLine string
		if i < len(s.topProcs) {
			p := s.topProcs[i]
			name := p.name
			if len(name) > 18 {
				name = name[:15] + "..."
			}
			procLine = fmt.Sprintf("  %-18s %s  %5.1f%%", name, renderProcBar(p.cpuPercent, 6), p.cpuPercent)
		} else {
			procLine = ""
		}

		var sysLine string
		switch i {
		case 0:
			sysLine = fmt.Sprintf(" Uptime    %s", formatUptime(s.uptime))
		case 1:
			sysLine = fmt.Sprintf(" Arch      %s", runtime.GOARCH)
		case 2:
			sysLine = fmt.Sprintf(" GoVer     %s", runtime.Version())
		default:
			sysLine = ""
		}

		b.WriteString(borderStyle.Render("│") + procLine +
			strings.Repeat(" ", maxInt(1, halfW-lipgloss.Width(procLine))) + borderStyle.Render("│") +
			labelStyle.Render(sysLine) + "\n")
	}

	// Bottom border
	bottom := borderStyle.Render("╰" + strings.Repeat("─", halfW) + "┴" + strings.Repeat("─", w-halfW-3) + "╯")
	b.WriteString(bottom + "\n")

	// Help
	b.WriteString(helpStyle.Render("  Press Q to quit · R to refresh · Refresh: 2s") + "\n")

	return b.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
