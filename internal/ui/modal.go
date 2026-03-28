package ui

import "github.com/charmbracelet/lipgloss"

func RenderModal(theme Theme, width, height int, content string) string {
	boxWidth := width / 2
	if boxWidth < 48 {
		boxWidth = width - 6
	}
	if boxWidth > 72 {
		boxWidth = 72
	}
	box := theme.Modal.Width(boxWidth).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
