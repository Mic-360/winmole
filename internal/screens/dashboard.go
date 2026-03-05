package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
	"github.com/mic-360/wimo/pkg/util"
)

func Dashboard(theme ui.Theme, store state.Store, width int) string {
	runtime := store.Runtime
	headline := lipgloss.JoinHorizontal(lipgloss.Top,
		theme.Header.Render("winmole"),
		"  ",
		theme.MutedText.Render("Windows-first maintenance cockpit"),
		"  ",
		ui.HealthBadge(theme, runtime.Health),
	)
	statusBody := strings.Join([]string{
		ui.Meter(theme, "CPU", runtime.CPU.Current, 34, ui.Spark(theme, runtime.CPU.History, 12)),
		ui.Meter(theme, "Memory", runtime.Memory.Current, 34, ui.Spark(theme, runtime.Memory.History, 12)),
		ui.Meter(theme, "Disk", runtime.Disk.Current, 34, ui.Spark(theme, runtime.Disk.History, 12)),
		fmt.Sprintf("%s  ↓ %s  ↑ %s", theme.MutedText.Render("Network"), theme.BodyText.Render(util.FormatRate(runtime.Network.DownloadRate)), theme.BodyText.Render(util.FormatRate(runtime.Network.UploadRate))),
	}, "\n")
	envBody := strings.Join([]string{
		ui.KeyValue(theme, "Host", runtime.Hostname),
		ui.KeyValue(theme, "Platform", runtime.Platform),
		ui.KeyValue(theme, "Build", runtime.Build),
		ui.KeyValue(theme, "PowerShell", runtime.PowerShell),
		ui.KeyValue(theme, "Go", runtime.GoVersion),
		ui.KeyValue(theme, "Uptime", util.FormatUptime(runtime.Uptime)),
	}, "\n")
	quickBody := strings.Join([]string{
		theme.BodyText.Render("Ctrl+P") + " command palette for quick jumps and actions",
		theme.BodyText.Render("Tab") + " switch focus between sidebar and current screen",
		theme.BodyText.Render("r") + " refresh live metrics and inventories",
		theme.BodyText.Render("x") + " execute the selected maintenance workflow",
		theme.BodyText.Render("/") + " filter lists or search logs",
	}, "\n")
	alerts := runtime.Alerts
	if len(alerts) > 5 {
		alerts = alerts[:5]
	}
	alertLines := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		alertLines = append(alertLines, "• "+alert)
	}
	if len(alertLines) == 0 {
		alertLines = append(alertLines, theme.MutedText.Render("No active alerts"))
	}
	leftWidth := width/2 - 2
	if leftWidth < 42 {
		leftWidth = width
	}
	rowOne := lipgloss.JoinHorizontal(lipgloss.Top,
		ui.Card(theme, "System overview", statusBody, leftWidth),
		ui.Card(theme, "Environment", envBody, leftWidth),
	)
	rowTwo := lipgloss.JoinHorizontal(lipgloss.Top,
		ui.Card(theme, "Quick commands", quickBody, leftWidth),
		ui.Card(theme, "Active signals", strings.Join(alertLines, "\n"), leftWidth),
	)
	if width < 110 {
		return lipgloss.JoinVertical(lipgloss.Left, headline, rowOne, rowTwo)
	}
	return lipgloss.JoinVertical(lipgloss.Left, headline, rowOne, rowTwo)
}
