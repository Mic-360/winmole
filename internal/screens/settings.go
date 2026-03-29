package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
)

func Settings(theme ui.Theme, store state.Store, listView string, width int) string {
	body := strings.Join([]string{
		ui.KeyValue(theme, "Theme", store.Config.Theme),
		ui.KeyValue(theme, "Refresh", formatRefresh(store.Config.RefreshIntervalSeconds)),
		ui.KeyValue(theme, "Purge depth", strconv.Itoa(store.Config.PurgeDepth)),
		ui.KeyValue(theme, "Winget", boolText(store.Config.WingetEnabled)),
		ui.KeyValue(theme, "Update checks", boolText(store.Config.CheckUpdates)),
	}, "\n")
	if width < 110 {
		return lipgloss.JoinVertical(lipgloss.Left, ui.Card(theme, "Config", body, width), listView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, ui.Card(theme, "Config", body, width/3), "  ", listView)
}

func boolText(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func formatRefresh(value int) string {
	if value <= 0 {
		value = 3
	}
	return fmt.Sprintf("%ds", value)
}
