package state

import "time"

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type Metric struct {
	Label   string
	Current float64
	Max     float64
	Unit    string
	Text    string
	History []float64
}

type DiskInfo struct {
	Mount   string
	FSType  string
	Used    int64
	Free    int64
	Total   int64
	Percent float64
}

type NetworkInfo struct {
	SentBytes          int64
	RecvBytes          int64
	UploadRate         float64
	DownloadRate       float64
	InterfaceSummaries []InterfaceSummary
}

type InterfaceSummary struct {
	Name      string
	SentBytes int64
	RecvBytes int64
}

type ProcessInfo struct {
	Name       string
	CPUPercent float64
	MemPercent float64
}

type GPUInfo struct {
	Name   string
	VRAM   string
	Driver string
}

type BatteryInfo struct {
	Present  bool
	Percent  int
	Charging bool
	Status   string
}

type ActivityTask struct {
	Name      string
	State     string
	StartedAt time.Time
	Detail    string
}

type RuntimeSnapshot struct {
	Timestamp  time.Time
	Hostname   string
	Platform   string
	Build      string
	PowerShell string
	GoVersion  string
	Uptime     time.Duration
	BootTime   time.Time
	Health     int
	CPU        Metric
	Memory     Metric
	Disk       Metric
	Network    NetworkInfo
	Disks      []DiskInfo
	Processes  []ProcessInfo
	GPUs       []GPUInfo
	Battery    BatteryInfo
	Tasks      []ActivityTask
	Alerts     []string
	IsAdmin    bool
}

type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Source  string
	Message string
}
