package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mic-360/wimo/internal/screens"
	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
)

func (m Model) View() string {
	layout := ui.ComputeLayout(m.width, m.height)
	sidebar := m.renderSidebar(layout)
	content := m.renderContent(layout)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	footer := m.renderFooter(layout.Width)
	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)
	if m.store.Palette.Visible {
		palette := ui.RenderPalette(m.theme, "Command palette", m.paletteInput.View(), m.paletteList.View(), m.width, m.height)
		return m.theme.App.Render(view + "\n" + palette)
	}
	if m.store.Modal.Visible {
		return m.theme.App.Render(view + "\n" + m.renderModal())
	}
	return m.theme.App.Render(view)
}

func (m Model) renderSidebar(layout ui.Layout) string {
	lines := []string{m.theme.Header.Render("winmole"), m.theme.MutedText.Render("Windows-first terminal maintenance"), ""}
	for index, item := range m.store.Navigation {
		label := fmt.Sprintf("%s  %s", item.Icon, item.Title)
		body := m.theme.SidebarItem.Render(label)
		if index == m.navIndex {
			body = m.theme.SidebarFocus.Render(label)
		}
		lines = append(lines, body)
		lines = append(lines, m.theme.MutedText.Render("   "+item.Subtitle))
	}
	lines = append(lines, "", m.theme.MutedText.Render("Focus: "+string(m.store.Focus)), m.theme.MutedText.Render("Profile: "+m.theme.Profile.Name()))
	return m.theme.Sidebar.Width(layout.SidebarWidth).Height(layout.BodyHeight).Render(strings.Join(lines, "\n"))
}

func (m Model) renderContent(layout ui.Layout) string {
	width := layout.ContentWidth - 3
	var content string
	switch m.store.Screen {
	case state.ScreenDashboard:
		content = screens.Dashboard(m.theme, m.store, width)
	case state.ScreenProjects:
		listView := m.projectsList.View()
		if m.store.ProjectMode == state.ProjectsModeArtifacts {
			listView = m.artifactsList.View()
		}
		content = screens.Projects(m.theme, m.store, listView, width)
	case state.ScreenActions:
		content = screens.Actions(m.theme, m.store, m.activeActionView(), width)
	case state.ScreenLogs:
		content = screens.Logs(m.theme, m.store, m.logsViewport.View(), width)
	case state.ScreenSettings:
		content = screens.Settings(m.theme, m.store, m.settingsList.View(), width)
	case state.ScreenHelp:
		content = screens.Help(m.theme, m.helpList.View(), m.helpViewport.View(), width)
	default:
		content = m.theme.MutedText.Render("Unknown screen")
	}
	return m.theme.PanelStyle.Width(layout.ContentWidth).Height(layout.BodyHeight).Render(content)
}

func (m Model) activeActionView() string {
	switch m.store.ActiveActionPane {
	case state.ActionPaneUninstall:
		return m.uninstallList.View()
	case state.ActionPaneOptimize:
		return m.optimizeList.View()
	default:
		return m.cleanList.View()
	}
}

func (m Model) renderFooter(width int) string {
	shortHelp := m.help.View(m.keys)
	status := m.store.StatusText
	if m.store.Busy {
		status = m.spinner.View() + "  " + status
	}
	right := m.theme.MutedText.Render(status)
	left := shortHelp
	if left == "" {
		left = ui.JoinHints(
			ui.RenderCommandHint(m.theme, "Esc", "back"),
			ui.RenderCommandHint(m.theme, "Ctrl+P", "palette"),
			ui.RenderCommandHint(m.theme, "?", "help"),
		)
	}
	return ui.Footer(m.theme, left, right, width)
}

func (m Model) renderModal() string {
	body := m.theme.Header.Render(m.store.Modal.Title) + "\n\n" + m.theme.BodyText.Render(m.store.Modal.Body)
	if m.store.Modal.Kind == "input" {
		body += "\n\n" + m.theme.Input.Render(m.modalInput.View())
		body += "\n\n" + m.theme.MutedText.Render("Enter to save · Esc to cancel")
		return ui.RenderModal(m.theme, m.width, m.height, body)
	}
	if m.store.Modal.Kind == "alert" {
		body += "\n\n" + m.theme.MutedText.Render("Enter or Esc to close")
		return ui.RenderModal(m.theme, m.width, m.height, body)
	}
	body += "\n\n" + ui.JoinHints(
		ui.RenderCommandHint(m.theme, "Enter", "confirm"),
		ui.RenderCommandHint(m.theme, "Esc", "cancel"),
	)
	return ui.RenderModal(m.theme, m.width, m.height, body)
}
