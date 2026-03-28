package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
	"github.com/mic-360/wimo/pkg/util"
)

func Actions(theme ui.Theme, store state.Store, listView string, width int) string {
	tabs := ui.Tabs(theme, []string{"Clean", "Uninstall", "Optimize"}, activePaneIndex(store.ActiveActionPane))
	summary := actionSummary(theme, store)
	if width < 110 {
		return lipgloss.JoinVertical(lipgloss.Left, tabs, ui.Card(theme, "Queue", summary, width), listView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		ui.Card(theme, "Queue", summary, width/3),
		"  ",
		lipgloss.JoinVertical(lipgloss.Left, tabs, listView),
	)
}

func actionSummary(theme ui.Theme, store state.Store) string {
	switch store.ActiveActionPane {
	case state.ActionPaneClean:
		selected, size := 0, int64(0)
		for _, target := range store.CleanTargets {
			if target.Selected {
				selected++
				size += target.Size
			}
		}
		return strings.Join([]string{
			ui.KeyValue(theme, "Workflow", "Deep clean"),
			ui.KeyValue(theme, "Selected", fmt.Sprintf("%d targets", selected)),
			ui.KeyValue(theme, "Potential reclaim", util.FormatBytes(size)),
			ui.KeyValue(theme, "Behavior", "Explicit selection, no auto-delete"),
		}, "\n")
	case state.ActionPaneUninstall:
		selected := 0
		size := int64(0)
		for _, app := range store.InstalledApps {
			if app.Selected {
				selected++
				size += app.Size
			}
		}
		return strings.Join([]string{
			ui.KeyValue(theme, "Workflow", "Application removal"),
			ui.KeyValue(theme, "Selected", fmt.Sprintf("%d apps", selected)),
			ui.KeyValue(theme, "Approx size", util.FormatBytes(size)),
			ui.KeyValue(theme, "Source", "Registry inventory enriched with winget"),
		}, "\n")
	default:
		selected := 0
		for _, task := range store.OptimizeTasks {
			if task.Selected {
				selected++
			}
		}
		return strings.Join([]string{
			ui.KeyValue(theme, "Workflow", "System optimization"),
			ui.KeyValue(theme, "Selected", fmt.Sprintf("%d tasks", selected)),
			ui.KeyValue(theme, "Approach", "User-chosen, categorized actions"),
			ui.KeyValue(theme, "Execution", "Sequential with runtime logging"),
		}, "\n")
	}
}

func activePaneIndex(pane state.ActionPane) int {
	switch pane {
	case state.ActionPaneUninstall:
		return 1
	case state.ActionPaneOptimize:
		return 2
	default:
		return 0
	}
}
