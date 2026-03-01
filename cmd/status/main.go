package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
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
	version      = "1.0.0"
	refreshRate  = 2 * time.Second
	maxProcesses = 10
)

// ━━━ Color Palette (Catppuccin Mocha) ━━━━━━━━━━━━━━━━━━

var (
	clrBrand   = lipgloss.Color("#9CB98B") // sage green
	clrAccent  = lipgloss.Color("#B5CDA3") // light sage
	clrGreen   = lipgloss.Color("#A6E3A1")
	clrYellow  = lipgloss.Color("#F9E2AF")
	clrRed     = lipgloss.Color("#F38BA8")
	clrPeach   = lipgloss.Color("#DBC5A0")
	clrTeal    = lipgloss.Color("#8FBCA3") // muted sage-teal
	clrText    = lipgloss.Color("#D5DDD0") // warm off-white
	clrSubtext = lipgloss.Color("#A3AE9E")
	clrOverlay = lipgloss.Color("#6B7466")
	clrSurface = lipgloss.Color("#3A4035") // dark sage
	clrDark    = lipgloss.Color("#1C211A")
)

// ━━━ Styles ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

var (
	brandStyle  = lipgloss.NewStyle().Bold(true).Foreground(clrBrand)
	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(clrBrand)
	labelStyle  = lipgloss.NewStyle().Foreground(clrSubtext)
	valueStyle  = lipgloss.NewStyle().Foreground(clrText).Bold(true)
	greenStyle  = lipgloss.NewStyle().Foreground(clrGreen)
	yellowStyle = lipgloss.NewStyle().Foreground(clrYellow)
	redStyle    = lipgloss.NewStyle().Foreground(clrRed)
	dimStyle    = lipgloss.NewStyle().Foreground(clrOverlay)
	peachStyle  = lipgloss.NewStyle().Foreground(clrPeach)
	tealStyle   = lipgloss.NewStyle().Foreground(clrTeal)
)

func newCardStyle(w int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrSurface).
		Padding(0, 1).
		Width(w)
}

// ━━━ Data Types ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

type tickMsg time.Time

type gpuInfo struct {
	name   string
	vram   string
	driver string
}

type batteryInfo struct {
	percent   int
	charging  bool
	present   bool
}

type diskPartInfo struct {
	mount   string
	total   uint64
	used    uint64
	free    uint64
	percent float64
	fstype  string
}

type netIfaceInfo struct {
	name     string
	bytesSent uint64
	bytesRecv uint64
}

type systemStats struct {
	cpuPercent  float64
	cpuModel    string
	cpuCores    int
	cpuThreads  int
	cpuFreq     float64
	memTotal    uint64
	memUsed     uint64
	memFree     uint64
	memPercent  float64
	swapTotal   uint64
	swapUsed    uint64
	diskTotal   uint64
	diskUsed    uint64
	diskFree    uint64
	diskPercent float64
	diskLetter  string
	netSent     uint64
	netRecv     uint64
	prevSent    uint64
	prevRecv    uint64
	netUpRate   float64
	netDownRate float64
	hostname    string
	osVersion   string
	uptime      uint64
	bootTime    uint64
	topProcs    []procInfo
	health      int
	gpus        []gpuInfo
	battery     batteryInfo
	disks       []diskPartInfo
	netIfaces   []netIfaceInfo
	procCount   int
	sysModel    string
	sysMfg      string
	biosVersion string
	winBuild    string
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
}

// ━━━ Bubbletea ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func initialModel() model {
	return model{width: 80, height: 24, stats: systemStats{diskLetter: "C:"}}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return tickMsg(time.Now()) }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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

func tick() tea.Cmd {
	return tea.Tick(refreshRate, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ━━━ Stats Gathering ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func gatherStats(prev systemStats) systemStats {
	s := systemStats{
		prevSent: prev.netSent,
		prevRecv: prev.netRecv,
	}

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

	diskUsage, err := disk.Usage("C:\\")
	if err == nil {
		s.diskTotal = diskUsage.Total
		s.diskUsed = diskUsage.Used
		s.diskFree = diskUsage.Free
		s.diskPercent = diskUsage.UsedPercent
		s.diskLetter = "C:"
	}

	netIO, err := net.IOCounters(false)
	if err == nil && len(netIO) > 0 {
		s.netSent = netIO[0].BytesSent
		s.netRecv = netIO[0].BytesRecv
		if s.prevSent > 0 {
			s.netUpRate = float64(s.netSent-s.prevSent) / refreshRate.Seconds()
			s.netDownRate = float64(s.netRecv-s.prevRecv) / refreshRate.Seconds()
		}
	}

	hostInfo, err := host.Info()
	if err == nil {
		s.hostname = hostInfo.Hostname
		s.osVersion = fmt.Sprintf("%s %s", hostInfo.Platform, hostInfo.PlatformVersion)
		s.uptime = hostInfo.Uptime
		s.bootTime = hostInfo.BootTime
	}

	procs, err := process.Processes()
	if err == nil {
		s.procCount = len(procs)
		var procList []procInfo
		for _, p := range procs {
			name, _ := p.Name()
			cpuPct, _ := p.CPUPercent()
			memPct, _ := p.MemoryPercent()
			if cpuPct > 0 || memPct > 0 {
				procList = append(procList, procInfo{name: name, cpuPercent: cpuPct, memPercent: memPct})
			}
		}
		sort.Slice(procList, func(i, j int) bool {
			return procList[i].cpuPercent > procList[j].cpuPercent
		})
		if len(procList) > maxProcesses {
			procList = procList[:maxProcesses]
		}
		s.topProcs = procList
	}

	// All disk partitions
	parts, err := disk.Partitions(false)
	if err == nil {
		var diskList []diskPartInfo
		for _, p := range parts {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil || usage.Total == 0 {
				continue
			}
			diskList = append(diskList, diskPartInfo{
				mount:   p.Mountpoint,
				total:   usage.Total,
				used:    usage.Used,
				free:    usage.Free,
				percent: usage.UsedPercent,
				fstype:  p.Fstype,
			})
		}
		s.disks = diskList
	}

	// GPU info via WMI
	s.gpus = getGPUs()

	// Battery info via WMI
	s.battery = getBattery()

	// System model/manufacturer
	s.sysModel, s.sysMfg = getSystemModel()

	// Windows build
	s.winBuild = getWinBuild()

	// Network interfaces
	netPerIface, err := net.IOCounters(true)
	if err == nil {
		for _, iface := range netPerIface {
			if iface.BytesSent > 0 || iface.BytesRecv > 0 {
				s.netIfaces = append(s.netIfaces, netIfaceInfo{
					name:      iface.Name,
					bytesSent: iface.BytesSent,
					bytesRecv: iface.BytesRecv,
				})
			}
		}
		// Limit to top 5 by total traffic
		sort.Slice(s.netIfaces, func(i, j int) bool {
			return (s.netIfaces[i].bytesSent + s.netIfaces[i].bytesRecv) > (s.netIfaces[j].bytesSent + s.netIfaces[j].bytesRecv)
		})
		if len(s.netIfaces) > 5 {
			s.netIfaces = s.netIfaces[:5]
		}
	}

	s.health = calculateHealth(s)
	return s
}

// ━━━ WMI Helpers ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func getGPUs() []gpuInfo {
	out, err := exec.Command("wmic", "path", "win32_VideoController", "get", "Name,AdapterRAM,DriverVersion", "/format:csv").CombinedOutput()
	if err != nil {
		return nil
	}
	var gpus []gpuInfo
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		vramBytes, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		vram := formatBytes(uint64(vramBytes))
		gpus = append(gpus, gpuInfo{
			name:   strings.TrimSpace(parts[2]),
			driver: strings.TrimSpace(parts[3]),
			vram:   vram,
		})
	}
	return gpus
}

func getBattery() batteryInfo {
	out, err := exec.Command("wmic", "path", "Win32_Battery", "get", "EstimatedChargeRemaining,BatteryStatus", "/format:csv").CombinedOutput()
	if err != nil {
		return batteryInfo{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		status, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		pct, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		return batteryInfo{
			percent:  pct,
			charging: status == 2,
			present:  true,
		}
	}
	return batteryInfo{}
}

func getSystemModel() (string, string) {
	modelOut, err := exec.Command("wmic", "computersystem", "get", "Model", "/format:value").CombinedOutput()
	model := ""
	if err == nil {
		for _, line := range strings.Split(string(modelOut), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Model=") {
				model = strings.TrimPrefix(strings.TrimSpace(line), "Model=")
			}
		}
	}
	mfgOut, err := exec.Command("wmic", "computersystem", "get", "Manufacturer", "/format:value").CombinedOutput()
	mfg := ""
	if err == nil {
		for _, line := range strings.Split(string(mfgOut), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Manufacturer=") {
				mfg = strings.TrimPrefix(strings.TrimSpace(line), "Manufacturer=")
			}
		}
	}
	return model, mfg
}

func getWinBuild() string {
	out, err := exec.Command("wmic", "os", "get", "BuildNumber", "/format:value").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "BuildNumber=") {
			return strings.TrimPrefix(strings.TrimSpace(line), "BuildNumber=")
		}
	}
	return ""
}

func calculateHealth(s systemStats) int {
	score := 100
	if s.cpuPercent > 90 {
		score -= 30
	} else if s.cpuPercent > 80 {
		score -= 20
	} else if s.cpuPercent > 60 {
		score -= 10
	}
	if s.memPercent > 90 {
		score -= 30
	} else if s.memPercent > 85 {
		score -= 20
	} else if s.memPercent > 70 {
		score -= 10
	}
	if s.diskPercent > 95 {
		score -= 30
	} else if s.diskPercent > 90 {
		score -= 20
	} else if s.diskPercent > 80 {
		score -= 10
	}
	return max(0, score)
}

// ━━━ View ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func (m model) View() string {
	w := m.width
	if w < 30 {
		w = 80
	}

	narrow := w < 70

	var cardW int
	if narrow {
		cardW = w
	} else {
		cardW = (w - 1) / 2
	}

	gaugeW := max(4, cardW-21)

	header := m.viewHeader(w)
	cpuCard := m.viewCPU(cardW, gaugeW)
	memCard := m.viewMem(cardW, gaugeW)
	disksCard := m.viewDisks(cardW, gaugeW)
	netCard := m.viewNet(cardW)
	procCard := m.viewProcs(cardW)
	sysCard := m.viewSys(cardW)
	gpuCard := m.viewGPU(cardW)
	batCard := m.viewBattery(cardW, gaugeW)

	var body string
	if narrow {
		body = lipgloss.JoinVertical(lipgloss.Left,
			header, cpuCard, memCard, disksCard, netCard, gpuCard, procCard, sysCard, batCard,
		)
	} else {
		r1 := lipgloss.JoinHorizontal(lipgloss.Top, cpuCard, " ", memCard)
		r2 := lipgloss.JoinHorizontal(lipgloss.Top, disksCard, " ", netCard)
		r3 := lipgloss.JoinHorizontal(lipgloss.Top, gpuCard, " ", batCard)
		r4 := lipgloss.JoinHorizontal(lipgloss.Top, procCard, " ", sysCard)
		body = lipgloss.JoinVertical(lipgloss.Left, header, r1, r2, r3, r4)
	}

	help := m.viewHelp()
	return body + "\n" + help + "\n"
}

func (m model) viewHeader(w int) string {
	s := m.stats
	left := brandStyle.Render("🐹 WiMo Status") + "  " + healthBadge(s.health)
	info := dimStyle.Render(s.hostname)
	if s.osVersion != "" {
		info += dimStyle.Render(" · " + s.osVersion)
	}
	if s.winBuild != "" {
		info += dimStyle.Render(" (Build " + s.winBuild + ")")
	}
	info += dimStyle.Render(fmt.Sprintf(" · %d procs", s.procCount))
	gap := max(2, w-4-lipgloss.Width(left)-lipgloss.Width(info))
	content := left + strings.Repeat(" ", gap) + info

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrSurface).
		Padding(0, 1).
		Width(w).
		Render(content)
}

func healthBadge(h int) string {
	label := fmt.Sprintf("● %d", h)
	var bg lipgloss.Color
	if h >= 80 {
		bg = clrGreen
	} else if h >= 60 {
		bg = clrYellow
	} else {
		bg = clrRed
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(clrDark).
		Background(bg).
		Padding(0, 1).
		Render(label)
}

func (m model) viewCPU(w, gw int) string {
	s := m.stats
	modelName := s.cpuModel
	maxLen := max(10, w-13)
	if len(modelName) > maxLen {
		modelName = modelName[:maxLen-3] + "..."
	}
	lines := []string{
		accentStyle.Render("⚙  CPU"),
		"",
		labelStyle.Render("Usage ") + bar(s.cpuPercent, gw) + valueStyle.Render(fmt.Sprintf(" %5.1f%%", s.cpuPercent)),
		"",
		labelStyle.Render("Model  ") + dimStyle.Render(modelName),
		labelStyle.Render("Cores  ") + valueStyle.Render(fmt.Sprintf("%d", s.cpuCores)) +
			dimStyle.Render(fmt.Sprintf(" · %d threads", s.cpuThreads)),
		labelStyle.Render("Freq   ") + valueStyle.Render(fmt.Sprintf("%.1f GHz", s.cpuFreq/1000)),
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewMem(w, gw int) string {
	s := m.stats
	swapPct := 0.0
	if s.swapTotal > 0 {
		swapPct = float64(s.swapUsed) / float64(s.swapTotal) * 100
	}
	lines := []string{
		accentStyle.Render("▦  Memory"),
		"",
		labelStyle.Render("Used  ") + bar(s.memPercent, gw) + valueStyle.Render(fmt.Sprintf(" %5.1f%%", s.memPercent)),
		labelStyle.Render("      ") + dimStyle.Render(formatBytes(s.memUsed)+" / "+formatBytes(s.memTotal)),
		"",
		labelStyle.Render("Free  ") + valueStyle.Render(formatBytes(s.memFree)),
		labelStyle.Render("Swap  ") + bar(swapPct, gw) + dimStyle.Render(fmt.Sprintf(" %s", formatBytes(s.swapUsed))),
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewDisks(w, gw int) string {
	s := m.stats
	lines := []string{
		accentStyle.Render("▤  Disks"),
		"",
	}

	if len(s.disks) > 0 {
		for _, d := range s.disks {
			mount := d.mount
			if len(mount) > 3 {
				mount = mount[:3]
			}
			bw := max(4, gw-4)
			lines = append(lines,
				labelStyle.Render(fmt.Sprintf("%-3s ", mount))+bar(d.percent, bw)+valueStyle.Render(fmt.Sprintf(" %5.1f%%", d.percent)),
				labelStyle.Render("    ")+dimStyle.Render(fmt.Sprintf("%s / %s  free: %s  %s", formatBytes(d.used), formatBytes(d.total), formatBytes(d.free), d.fstype)),
			)
		}
	} else {
		lines = append(lines,
			labelStyle.Render("Used  ")+bar(s.diskPercent, gw)+valueStyle.Render(fmt.Sprintf(" %5.1f%%", s.diskPercent)),
			labelStyle.Render("      ")+dimStyle.Render(formatBytes(s.diskUsed)+" / "+formatBytes(s.diskTotal)),
			"",
			labelStyle.Render("Free  ")+valueStyle.Render(formatBytes(s.diskFree)),
		)
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewNet(w int) string {
	s := m.stats
	lines := []string{
		accentStyle.Render("⇅  Network"),
		"",
		tealStyle.Render("↓") + labelStyle.Render(" Down  ") + valueStyle.Render(formatRate(s.netDownRate)),
		peachStyle.Render("↑") + labelStyle.Render(" Up    ") + valueStyle.Render(formatRate(s.netUpRate)),
		"",
		labelStyle.Render("  Sent  ") + dimStyle.Render(formatBytes(s.netSent)),
		labelStyle.Render("  Recv  ") + dimStyle.Render(formatBytes(s.netRecv)),
	}

	if len(s.netIfaces) > 0 {
		lines = append(lines, "")
		maxIfaces := min(3, len(s.netIfaces))
		for i := 0; i < maxIfaces; i++ {
			iface := s.netIfaces[i]
			name := iface.name
			if len(name) > 18 {
				name = name[:15] + "..."
			}
			lines = append(lines, dimStyle.Render(fmt.Sprintf("  %-18s ↑%s ↓%s", name, formatBytes(iface.bytesSent), formatBytes(iface.bytesRecv))))
		}
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewProcs(w int) string {
	s := m.stats
	nameW := 16
	procBarW := max(3, min(12, w-4-nameW-16))

	lines := []string{
		accentStyle.Render("▶  Top Processes") + dimStyle.Render(fmt.Sprintf("  (%d total)", s.procCount)),
		"",
		dimStyle.Render(fmt.Sprintf("%-*s %-*s %6s %5s", nameW, "Process", procBarW, "CPU", "CPU%", "Mem%")),
	}
	for i := 0; i < maxProcesses; i++ {
		if i < len(s.topProcs) {
			p := s.topProcs[i]
			name := p.name
			if len(name) > nameW {
				name = name[:nameW-3] + "..."
			}
			b := procBar(p.cpuPercent, procBarW)
			lines = append(lines, fmt.Sprintf("%-*s %s %5.1f%% %4.1f%%", nameW, name, b, p.cpuPercent, p.memPercent))
		} else {
			lines = append(lines, dimStyle.Render("·"))
		}
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewSys(w int) string {
	s := m.stats

	bootTimeStr := ""
	if s.bootTime > 0 {
		bt := time.Unix(int64(s.bootTime), 0)
		bootTimeStr = bt.Format("2006-01-02 15:04")
	}

	lines := []string{
		accentStyle.Render("ℹ  System"),
		"",
		labelStyle.Render("Uptime   ") + valueStyle.Render(formatUptime(s.uptime)),
		labelStyle.Render("Boot     ") + dimStyle.Render(bootTimeStr),
		labelStyle.Render("OS       ") + valueStyle.Render(s.osVersion),
	}
	if s.winBuild != "" {
		lines = append(lines, labelStyle.Render("Build    ")+dimStyle.Render(s.winBuild))
	}
	lines = append(lines,
		labelStyle.Render("Host     ")+valueStyle.Render(s.hostname),
		labelStyle.Render("Arch     ")+valueStyle.Render(runtime.GOARCH),
	)
	if s.sysMfg != "" {
		lines = append(lines, labelStyle.Render("Vendor   ")+dimStyle.Render(s.sysMfg))
	}
	if s.sysModel != "" {
		lines = append(lines, labelStyle.Render("Model    ")+dimStyle.Render(s.sysModel))
	}
	lines = append(lines, labelStyle.Render("Runtime  ")+dimStyle.Render(runtime.Version()))
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewGPU(w int) string {
	s := m.stats
	lines := []string{
		accentStyle.Render("◈  GPU"),
		"",
	}

	if len(s.gpus) == 0 {
		lines = append(lines, dimStyle.Render("No GPU detected"))
	} else {
		for i, gpu := range s.gpus {
			if i > 0 {
				lines = append(lines, "")
			}
			name := gpu.name
			maxLen := max(10, w-8)
			if len(name) > maxLen {
				name = name[:maxLen-3] + "..."
			}
			lines = append(lines, valueStyle.Render(name))
			if gpu.vram != "" && gpu.vram != "0 B" {
				lines = append(lines, labelStyle.Render("VRAM    ")+dimStyle.Render(gpu.vram))
			}
			if gpu.driver != "" {
				lines = append(lines, labelStyle.Render("Driver  ")+dimStyle.Render(gpu.driver))
			}
		}
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewBattery(w, gw int) string {
	s := m.stats
	lines := []string{
		accentStyle.Render("🔋 Battery"),
		"",
	}

	if !s.battery.present {
		lines = append(lines,
			dimStyle.Render("No battery detected"),
			"",
			dimStyle.Render("Desktop/AC powered"),
		)
	} else {
		pct := float64(s.battery.percent)
		status := "Discharging"
		if s.battery.charging {
			status = "Charging"
		}
		lines = append(lines,
			labelStyle.Render("Level  ")+bar(pct, gw)+valueStyle.Render(fmt.Sprintf(" %d%%", s.battery.percent)),
			labelStyle.Render("Status ")+valueStyle.Render(status),
		)
	}
	return newCardStyle(w).Render(strings.Join(lines, "\n"))
}

func (m model) viewHelp() string {
	return "  " +
		lipgloss.NewStyle().Bold(true).Foreground(clrText).Render("q") + " " +
		dimStyle.Render("quit") +
		dimStyle.Render(" · ") +
		lipgloss.NewStyle().Bold(true).Foreground(clrText).Render("r") + " " +
		dimStyle.Render("refresh") +
		dimStyle.Render(" · ") +
		dimStyle.Render("refreshing every 2s")
}

// ━━━ Bar Rendering ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func bar(percent float64, width int) string {
	if width < 1 {
		return ""
	}
	filled := int(math.Round(percent / 100 * float64(width)))
	filled = max(0, min(filled, width))
	empty := width - filled

	var s lipgloss.Style
	switch {
	case percent < 50:
		s = greenStyle
	case percent < 75:
		s = yellowStyle
	default:
		s = redStyle
	}
	return s.Render(strings.Repeat("█", filled)) + dimStyle.Render(strings.Repeat("░", empty))
}

func procBar(percent float64, width int) string {
	if width < 1 {
		return ""
	}
	filled := int(math.Round(percent / 100 * float64(width)))
	filled = max(0, min(filled, width))
	empty := width - filled

	var s lipgloss.Style
	switch {
	case percent < 30:
		s = tealStyle
	case percent < 60:
		s = yellowStyle
	default:
		s = redStyle
	}
	return s.Render(strings.Repeat("█", filled)) + dimStyle.Render(strings.Repeat("░", empty))
}

// ━━━ Utilities ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

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
	switch {
	case bytesPerSec < 1024:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	case bytesPerSec < 1024*1024:
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	default:
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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
