package screens

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/internal/ui"
)

func Help(theme ui.Theme, listView, docView string, width int) string {
	if width < 110 {
		return lipgloss.JoinVertical(lipgloss.Left, listView, ui.Card(theme, "Document", docView, width))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", ui.Card(theme, "Document", docView, width/2))
}
