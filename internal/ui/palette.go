package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderPalette(theme Theme, title, prompt, listView string, width, height int) string {
	body := theme.Subheader.Render(title) + "\n\n" + theme.Input.Render(prompt) + "\n\n" + listView
	return RenderModal(theme, width, height, body)
}

func RenderCommandHint(theme Theme, keyLabel, description string) string {
	return fmt.Sprintf("%s %s", theme.Key.Render(keyLabel), theme.MutedText.Render(description))
}

func JoinHints(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "  ")
}

func PlaceMain(theme Theme, width, height int, content string) string {
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}
