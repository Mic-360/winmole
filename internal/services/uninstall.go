package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/pkg/util"
)

type UninstallService struct {
	logger *Logger
}

func NewUninstallService(logger *Logger) *UninstallService {
	return &UninstallService{logger: logger}
}

func (u *UninstallService) Inventory(ctx context.Context, wingetEnabled bool) ([]state.InstalledApp, error) {
	apps := u.registryApps()
	if wingetEnabled {
		u.enrichWithWinget(ctx, apps)
	}
	u.addLocalPrograms(apps)
	items := make([]state.InstalledApp, 0, len(apps))
	for _, app := range apps {
		items = append(items, app)
	}
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
	return items, nil
}

func (u *UninstallService) Remove(ctx context.Context, apps []state.InstalledApp) (OperationReport, error) {
	report := OperationReport{Title: "Uninstall", Message: "Removal complete"}
	for _, app := range apps {
		if !app.Selected {
			continue
		}
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		default:
		}
		if err := u.removeApp(ctx, app); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", app.Name, err))
			continue
		}
		report.Count++
		report.Bytes += app.Size
	}
	return report, nil
}

func (u *UninstallService) registryApps() map[string]state.InstalledApp {
	roots := []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER, registry.LOCAL_MACHINE}
	paths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	apps := map[string]state.InstalledApp{}
	for index, path := range paths {
		key, err := registry.OpenKey(roots[index], path, registry.READ)
		if err != nil {
			continue
		}
		names, _ := key.ReadSubKeyNames(-1)
		for _, subKeyName := range names {
			subKey, err := registry.OpenKey(key, subKeyName, registry.READ)
			if err != nil {
				continue
			}
			app, ok := readAppFromRegistry(subKeyName, subKey)
			subKey.Close()
			if !ok {
				continue
			}
			apps[app.ID] = app
		}
		key.Close()
	}
	return apps
}

func readAppFromRegistry(keyName string, key registry.Key) (state.InstalledApp, bool) {
	name := readStringValue(key, "DisplayName")
	if name == "" {
		return state.InstalledApp{}, false
	}
	systemComponent := readIntegerValue(key, "SystemComponent")
	releaseType := readStringValue(key, "ReleaseType")
	parentKeyName := readStringValue(key, "ParentKeyName")
	if !shouldIncludeApp(name, releaseType, parentKeyName, systemComponent) {
		return state.InstalledApp{}, false
	}
	uninstallString := readStringValue(key, "UninstallString")
	quietString := readStringValue(key, "QuietUninstallString")
	installLocation := readStringValue(key, "InstallLocation")
	if uninstallString == "" && quietString == "" && installLocation == "" {
		return state.InstalledApp{}, false
	}
	publisher := readStringValue(key, "Publisher")
	version := readStringValue(key, "DisplayVersion")
	method := "registry"
	if quietString != "" {
		method = "quiet"
	} else if strings.Contains(strings.ToLower(uninstallString), "msiexec") {
		method = "msi"
	}
	installDate := formatInstallDate(readStringValue(key, "InstallDate"))
	sizeKB := readIntegerValue(key, "EstimatedSize")
	sizeBytes := int64(sizeKB * 1024)
	id := normalizedAppID(name, publisher)
	return state.InstalledApp{ID: id, Name: name, Version: version, Publisher: publisher, Source: "registry", InstallDate: installDate, Size: sizeBytes, WingetID: "", UninstallString: uninstallString, QuietUninstallString: quietString, LocalPath: installLocation, Method: method}, true
}

func shouldIncludeApp(name, releaseType, parentKeyName string, systemComponent uint64) bool {
	if systemComponent == 1 {
		return false
	}
	if parentKeyName != "" {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	blockedPrefixes := []string{"security update", "update for", "hotfix for", "kb", "microsoft visual c++"}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	blockedContains := []string{"windows driver package", "language pack", "servicing stack", "update helper", "edge update", "webview2 runtime", "redistributable"}
	for _, fragment := range blockedContains {
		if strings.Contains(lower, fragment) {
			return false
		}
	}
	releaseLower := strings.ToLower(releaseType)
	if strings.Contains(releaseLower, "update") || strings.Contains(releaseLower, "hotfix") {
		return false
	}
	return true
}

func (u *UninstallService) enrichWithWinget(ctx context.Context, apps map[string]state.InstalledApp) {
	if _, err := exec.LookPath("winget"); err != nil {
		return
	}
	output, err := exec.CommandContext(ctx, "winget", "list", "--accept-source-agreements", "--disable-interactivity").CombinedOutput()
	if err != nil {
		u.logger.Warn("uninstall", "winget inventory unavailable")
		return
	}
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(stripANSI(line))
		if line == "" || strings.HasPrefix(line, "Name ") || strings.HasPrefix(line, "-") || strings.Contains(line, "upgrades available") {
			continue
		}
		columns := splitter.Split(line, -1)
		if len(columns) < 2 {
			continue
		}
		name := strings.TrimSpace(columns[0])
		wingetID := strings.TrimSpace(columns[1])
		if name == "" || wingetID == "" {
			continue
		}
		id := normalizedAppID(name, "")
		for key, app := range apps {
			if strings.EqualFold(app.Name, name) || strings.Contains(strings.ToLower(app.Name), strings.ToLower(name)) || strings.Contains(strings.ToLower(name), strings.ToLower(app.Name)) || key == id {
				app.WingetID = wingetID
				app.Source = "winget+registry"
				if app.Method == "registry" {
					app.Method = "winget"
				}
				apps[key] = app
				break
			}
		}
	}
}

func (u *UninstallService) addLocalPrograms(apps map[string]state.InstalledApp) {
	root := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		id := normalizedAppID(name, "")
		if _, exists := apps[id]; exists {
			continue
		}
		fullPath := filepath.Join(root, name)
		size, _ := limitedDirSize(fullPath, 2)
		apps[id] = state.InstalledApp{ID: id, Name: name, Source: "local", Size: size, LocalPath: fullPath, Method: "folder"}
	}
}

func (u *UninstallService) removeApp(ctx context.Context, app state.InstalledApp) error {
	u.logger.Info("uninstall", "starting removal for "+app.Name)
	var err error
	switch {
	case app.WingetID != "":
		err = exec.CommandContext(ctx, "winget", "uninstall", "--id", app.WingetID, "--silent", "--accept-source-agreements", "--disable-interactivity").Run()
	case app.QuietUninstallString != "":
		err = exec.CommandContext(ctx, "cmd", "/C", app.QuietUninstallString).Run()
	case app.UninstallString != "":
		cmd := app.UninstallString
		if strings.Contains(strings.ToLower(cmd), "msiexec") && !strings.Contains(strings.ToLower(cmd), "/quiet") {
			cmd += " /quiet /norestart"
		}
		err = exec.CommandContext(ctx, "cmd", "/C", cmd).Run()
	case app.LocalPath != "":
		err = os.RemoveAll(app.LocalPath)
	default:
		return fmt.Errorf("no uninstall method available")
	}
	if err != nil {
		return err
	}
	cleanupCommonLeftovers(app.Name)
	u.logger.Info("uninstall", "removed "+app.Name)
	return nil
}

func cleanupCommonLeftovers(name string) {
	for _, base := range []string{os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), `C:\ProgramData`} {
		if base == "" {
			continue
		}
		candidate := filepath.Join(base, name)
		_ = os.RemoveAll(candidate)
	}
}

func normalizedAppID(name, publisher string) string {
	value := strings.ToLower(util.NormalizeWhitespace(name + " " + publisher))
	value = strings.NewReplacer("(", " ", ")", " ", "_", " ", "-", " ").Replace(value)
	value = strings.Join(strings.Fields(value), "-")
	return value
}

func stripANSI(input string) string {
	return ansiRx.ReplaceAllString(input, "")
}

func formatInstallDate(value string) string {
	if len(value) != 8 {
		return ""
	}
	parsed, err := time.Parse("20060102", value)
	if err != nil {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func readStringValue(key registry.Key, name string) string {
	value, _, err := key.GetStringValue(name)
	if err == nil {
		return strings.TrimSpace(value)
	}
	return ""
}

func readIntegerValue(key registry.Key, name string) uint64 {
	value, _, err := key.GetIntegerValue(name)
	if err == nil {
		return value
	}
	return 0
}

func limitedDirSize(path string, depth int) (int64, error) {
	total := int64(0)
	rootDepth := strings.Count(filepath.Clean(path), string(filepath.Separator))
	err := filepath.Walk(path, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			currentDepth := strings.Count(filepath.Clean(current), string(filepath.Separator)) - rootDepth
			if currentDepth > depth {
				return filepath.SkipDir
			}
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}

var (
	splitter = regexp.MustCompile(`\s{2,}`)
	ansiRx   = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)
