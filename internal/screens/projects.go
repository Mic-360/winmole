package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
	"github.com/mic-360/wimo/pkg/util"
)

func Projects(theme ui.Theme, store state.Store, listView string, width int) string {
	project := store.SelectedProjectData()
	if project == nil {
		return ui.Card(theme, "Projects", theme.MutedText.Render("No projects discovered yet. Configure scan paths in Settings and press r."), width)
	}
	badgeParts := make([]string, 0, len(project.Ecosystems))
	for _, eco := range project.Ecosystems {
		badgeParts = append(badgeParts, ui.Badge(theme, strings.ToUpper(eco), theme.Primary))
	}
	summary := strings.Join([]string{
		theme.Header.Render(project.Name),
		theme.MutedText.Render(util.ShortenPath(project.Root, 52)),
		strings.Join(badgeParts, " "),
		"",
		ui.KeyValue(theme, "Artifacts", fmt.Sprintf("%d items", project.ArtifactCount)),
		ui.KeyValue(theme, "Reclaimable", util.FormatBytes(project.TotalArtifactBytes)),
		ui.KeyValue(theme, "Last scan", project.LastScan.Format("15:04:05")),
	}, "\n")
	analyzer := []string{}
	for _, usage := range project.Analyzer {
		analyzer = append(analyzer, ui.Meter(theme, usage.Name, usage.Percent, 34, ""))
		analyzer = append(analyzer, theme.MutedText.Render("  "+util.FormatBytes(usage.Size)+"  "+util.ShortenPath(usage.Path, 42)))
	}
	if len(analyzer) == 0 {
		analyzer = append(analyzer, theme.MutedText.Render("No analyzer data available for this project"))
	}
	right := lipgloss.JoinVertical(lipgloss.Left,
		ui.Card(theme, "Project summary", summary, width/2),
		ui.Card(theme, map[bool]string{true: "Artifact selection", false: "Analyzer"}[store.ProjectMode == state.ProjectsModeArtifacts], strings.Join(analyzer, "\n"), width/2),
	)
	if width < 110 {
		return lipgloss.JoinVertical(lipgloss.Left, listView, right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", right)
}
