package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/pkg/util"
)

func Card(theme Theme, title string, body string, width int) string {
	style := theme.Card.Copy()
	if width > 0 {
		style = style.Width(width)
	}
	return style.Render(theme.Subheader.Render(title) + "\n\n" + body)
}

func Badge(theme Theme, label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(theme.Background).Background(color).Padding(0, 1).Bold(true).Render(label)
}

func HealthBadge(theme Theme, health int) string {
	color := theme.Accent
	if health < 80 {
		color = theme.Warning
	}
	if health < 60 {
		color = theme.Error
	}
	return Badge(theme, fmt.Sprintf("HEALTH %d", health), color)
}

func Meter(theme Theme, label string, value float64, width int, spark string) string {
	if width < 12 {
		width = 12
	}
	barWidth := width - 10
	if barWidth < 6 {
		barWidth = 6
	}
	filled := int((value / 100) * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled
	color := theme.Accent
	if value > 70 {
		color = theme.Warning
	}
	if value > 85 {
		color = theme.Error
	}
	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) + theme.MutedText.Render(strings.Repeat("░", empty))
	line := fmt.Sprintf("%-10s %s %5.1f%%", label, bar, value)
	if spark != "" {
		line += "  " + theme.MutedText.Render(spark)
	}
	return line
}

func TableLines(theme Theme, title string, rows []string, width int) string {
	if len(rows) == 0 {
		rows = []string{theme.MutedText.Render("No data available")}
	}
	return Card(theme, title, strings.Join(rows, "\n"), width)
}

func KeyValue(theme Theme, key, value string) string {
	return theme.MutedText.Render(key) + "  " + theme.BodyText.Render(value)
}

func Tabs(theme Theme, labels []string, active int) string {
	parts := make([]string, 0, len(labels))
	for index, label := range labels {
		style := theme.Tab
		if index == active {
			style = theme.TabActive
		}
		parts = append(parts, style.Render(label))
	}
	return strings.Join(parts, " ")
}

func Footer(theme Theme, left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return theme.StatusBar.Width(width).Render(left + strings.Repeat(" ", gap) + right)
}

func Spark(theme Theme, values []float64, width int) string {
	return lipgloss.NewStyle().Foreground(theme.Primary).Render(util.Sparkline(values, width))
}
