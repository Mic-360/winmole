package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type Theme struct {
	Profile termenv.Profile

	Background lipgloss.Color
	Primary    lipgloss.Color
	Accent     lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Surface    lipgloss.Color
	Panel      lipgloss.Color
	Border     lipgloss.Color

	App          lipgloss.Style
	Header       lipgloss.Style
	Subheader    lipgloss.Style
	Sidebar      lipgloss.Style
	SidebarItem  lipgloss.Style
	SidebarFocus lipgloss.Style
	PanelStyle   lipgloss.Style
	Card         lipgloss.Style
	MutedText    lipgloss.Style
	BodyText     lipgloss.Style
	AccentText   lipgloss.Style
	SuccessText  lipgloss.Style
	WarningText  lipgloss.Style
	ErrorText    lipgloss.Style
	StatusBar    lipgloss.Style
	Key          lipgloss.Style
	Tab          lipgloss.Style
	TabActive    lipgloss.Style
	Badge        lipgloss.Style
	Input        lipgloss.Style
	Modal        lipgloss.Style
}

func NewTheme() Theme {
	profile := termenv.ColorProfile()
	background := lipgloss.Color("#0f172a")
	primary := lipgloss.Color("#7c3aed")
	accent := lipgloss.Color("#22c55e")
	warning := lipgloss.Color("#f59e0b")
	errorColor := lipgloss.Color("#ef4444")
	text := lipgloss.Color("#e5e7eb")
	muted := lipgloss.Color("#6b7280")
	surface := lipgloss.Color("#111827")
	panel := lipgloss.Color("#172033")
	border := lipgloss.Color("#334155")

	return Theme{
		Profile:    profile,
		Background: background,
		Primary:    primary,
		Accent:     accent,
		Warning:    warning,
		Error:      errorColor,
		Text:       text,
		Muted:      muted,
		Surface:    surface,
		Panel:      panel,
		Border:     border,
		App: lipgloss.NewStyle().
			Background(background).
			Foreground(text),
		Header: lipgloss.NewStyle().
			Foreground(text).
			Bold(true),
		Subheader: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),
		Sidebar: lipgloss.NewStyle().
			Background(surface).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(border).
			Padding(1, 1),
		SidebarItem: lipgloss.NewStyle().
			Foreground(text).
			Padding(0, 1),
		SidebarFocus: lipgloss.NewStyle().
			Foreground(text).
			Background(primary).
			Bold(true).
			Padding(0, 1),
		PanelStyle: lipgloss.NewStyle().
			Background(background).
			Padding(1, 2),
		Card: lipgloss.NewStyle().
			Background(panel).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(1, 1),
		MutedText:   lipgloss.NewStyle().Foreground(muted),
		BodyText:    lipgloss.NewStyle().Foreground(text),
		AccentText:  lipgloss.NewStyle().Foreground(accent).Bold(true),
		SuccessText: lipgloss.NewStyle().Foreground(accent),
		WarningText: lipgloss.NewStyle().Foreground(warning),
		ErrorText:   lipgloss.NewStyle().Foreground(errorColor).Bold(true),
		StatusBar: lipgloss.NewStyle().
			Background(surface).
			Foreground(text).
			Padding(0, 1),
		Key: lipgloss.NewStyle().
			Foreground(background).
			Background(text).
			Padding(0, 1),
		Tab: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),
		TabActive: lipgloss.NewStyle().
			Foreground(text).
			Background(primary).
			Bold(true).
			Padding(0, 1),
		Badge: lipgloss.NewStyle().
			Foreground(text).
			Background(primary).
			Padding(0, 1),
		Input: lipgloss.NewStyle().
			Foreground(text).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(0, 1),
		Modal: lipgloss.NewStyle().
			Background(surface).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(primary).
			Padding(1, 2),
	}
}
