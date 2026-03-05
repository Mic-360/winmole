package services

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mic-360/wimo/internal/state"
)

type CleanerService struct {
	logger *Logger
}

func NewCleanerService(logger *Logger) *CleanerService {
	return &CleanerService{logger: logger}
}

func (c *CleanerService) Targets() []state.CleanTarget {
	home, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	roaming := os.Getenv("APPDATA")
	temp := os.TempDir()
	return []state.CleanTarget{
		{ID: "temp", Category: "System", Title: "Temporary files", Description: "Temp folders used by installers and tools", Paths: []string{temp, filepath.Join(local, "Temp")}, Selected: true, Status: "ready"},
		{ID: "browser-cache", Category: "Browsers", Title: "Browser caches", Description: "Chrome, Edge, Brave, Firefox cache folders", Paths: []string{filepath.Join(local, "Google", "Chrome", "User Data", "Default", "Cache"), filepath.Join(local, "Microsoft", "Edge", "User Data", "Default", "Cache"), filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data", "Default", "Cache"), filepath.Join(local, "Mozilla", "Firefox", "Profiles", "*", "cache2")}, Selected: true, Status: "ready"},
		{ID: "gpu-cache", Category: "Browsers", Title: "GPU caches", Description: "Browser and Electron GPU caches", Paths: []string{filepath.Join(local, "Google", "Chrome", "User Data", "Default", "GPUCache"), filepath.Join(local, "Microsoft", "Edge", "User Data", "Default", "GPUCache"), filepath.Join(roaming, "discord", "GPUCache")}, Selected: true, Status: "ready"},
		{ID: "thumbnail-cache", Category: "Explorer", Title: "Explorer caches", Description: "Thumbnail and icon databases", Paths: []string{filepath.Join(local, "Microsoft", "Windows", "Explorer", "thumbcache_*.db"), filepath.Join(local, "Microsoft", "Windows", "Explorer", "iconcache_*.db")}, Selected: true, Status: "ready"},
		{ID: "dev-caches", Category: "Developer", Title: "Developer caches", Description: "npm, pnpm, pip, go build and NuGet caches", Paths: []string{filepath.Join(roaming, "npm-cache"), filepath.Join(local, "pnpm", "store"), filepath.Join(local, "pip", "Cache"), filepath.Join(local, "go-build"), filepath.Join(local, "NuGet", "v3-cache"), filepath.Join(home, ".gradle", "caches")}, Selected: false, Status: "ready"},
		{ID: "workspace-storage", Category: "Developer", Title: "Editor workspace storage", Description: "VS Code and JetBrains logs and workspace data", Paths: []string{filepath.Join(roaming, "Code", "logs"), filepath.Join(roaming, "Code", "User", "workspaceStorage"), filepath.Join(local, "JetBrains", "*", "caches")}, Selected: false, Status: "ready"},
		{ID: "chat-clients", Category: "Apps", Title: "Chat and media caches", Description: "Discord, Slack, Teams, Zoom, Spotify cache folders", Paths: []string{filepath.Join(roaming, "discord", "Cache"), filepath.Join(roaming, "Slack", "Cache"), filepath.Join(roaming, "Microsoft", "Teams", "Cache"), filepath.Join(roaming, "Zoom", "data"), filepath.Join(local, "Spotify", "Data")}, Selected: false, Status: "ready"},
		{ID: "recycle-bin", Category: "System", Title: "Recycle Bin", Description: "Deleted files still occupying storage", Paths: []string{filepath.Join(`C:\`, `$Recycle.Bin`)}, Selected: false, RequiresAdmin: true, Status: "ready"},
		{ID: "windows-update", Category: "System", Title: "Windows update cache", Description: "SoftwareDistribution downloads and delivery optimization", Paths: []string{filepath.Join(`C:\Windows`, `SoftwareDistribution`, `Download`), filepath.Join(`C:\Windows`, `SoftwareDistribution`, `DeliveryOptimization`)}, Selected: false, RequiresAdmin: true, Status: "ready"},
		{ID: "windows-logs", Category: "System", Title: "Windows logs", Description: "DISM, CBS and temporary log files", Paths: []string{filepath.Join(`C:\Windows`, `Temp`), filepath.Join(`C:\Windows`, `Logs`, `DISM`), filepath.Join(`C:\Windows`, `Logs`, `CBS`)}, Selected: false, RequiresAdmin: true, Status: "ready"},
	}
}

func (c *CleanerService) Scan(ctx context.Context, targets []state.CleanTarget) ([]state.CleanTarget, error) {
	updated := make([]state.CleanTarget, len(targets))
	copy(updated, targets)
	for index := range updated {
		select {
		case <-ctx.Done():
			return updated, ctx.Err()
		default:
		}
		totalSize := int64(0)
		items := 0
		for _, pattern := range updated[index].Paths {
			matches := resolvePattern(pattern)
			for _, match := range matches {
				size, count := pathUsage(match)
				totalSize += size
				items += count
			}
		}
		updated[index].Size = totalSize
		updated[index].Items = items
		if updated[index].RequiresAdmin && !probeAdmin() {
			updated[index].Status = "admin"
		} else if totalSize == 0 {
			updated[index].Status = "empty"
		} else {
			updated[index].Status = "ready"
		}
	}
	sort.Slice(updated, func(i, j int) bool { return updated[i].Size > updated[j].Size })
	return updated, nil
}

func (c *CleanerService) Execute(ctx context.Context, targets []state.CleanTarget, dryRun bool) (OperationReport, []state.CleanTarget, error) {
	updated := make([]state.CleanTarget, len(targets))
	copy(updated, targets)
	report := OperationReport{Title: "Cleanup", Message: "Cleanup complete"}
	for index := range updated {
		if !updated[index].Selected {
			continue
		}
		if updated[index].RequiresAdmin && !probeAdmin() {
			updated[index].Status = "skipped"
			report.Errors = append(report.Errors, updated[index].Title+": administrator rights required")
			continue
		}
		select {
		case <-ctx.Done():
			return report, updated, ctx.Err()
		default:
		}
		matches := []string{}
		for _, pattern := range updated[index].Paths {
			matches = append(matches, resolvePattern(pattern)...)
		}
		for _, match := range matches {
			if protectedPath(match) {
				report.Errors = append(report.Errors, updated[index].Title+": blocked protected path "+match)
				continue
			}
			bytes, _ := pathUsage(match)
			if dryRun {
				report.Bytes += bytes
				continue
			}
			if err := removePath(match); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", match, err))
				continue
			}
			report.Bytes += bytes
		}
		report.Count++
		updated[index].Status = "done"
	}
	if dryRun {
		report.Message = "Dry run complete"
	}
	return report, updated, nil
}

func resolvePattern(pattern string) []string {
	pattern = os.ExpandEnv(pattern)
	matches, err := filepath.Glob(pattern)
	if err == nil && len(matches) > 0 {
		return matches
	}
	if _, err := os.Stat(pattern); err == nil {
		return []string{pattern}
	}
	return nil
}

func pathUsage(path string) (int64, int) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0
	}
	if !info.IsDir() {
		return info.Size(), 1
	}
	total := int64(0)
	count := 0
	_ = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		count++
		if fileInfo, err := entry.Info(); err == nil {
			total += fileInfo.Size()
		}
		return nil
	})
	return total, count
}

func protectedPath(path string) bool {
	path = strings.ToLower(filepath.Clean(path))
	protected := []string{strings.ToLower(`C:\Windows`), strings.ToLower(`C:\Program Files`), strings.ToLower(`C:\Program Files (x86)`), strings.ToLower(`C:\Users`)}
	for _, prefix := range protected {
		if path == prefix {
			return true
		}
	}
	if strings.Contains(path, strings.ToLower(`\Documents\`)) || strings.Contains(path, strings.ToLower(`\Desktop\`)) || strings.Contains(path, strings.ToLower(`\Downloads\`)) {
		return true
	}
	return false
}

func removePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}
