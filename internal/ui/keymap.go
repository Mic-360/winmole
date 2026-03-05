package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Tab        key.Binding
	BackTab    key.Binding
	Enter      key.Binding
	Back       key.Binding
	Toggle     key.Binding
	Run        key.Binding
	Refresh    key.Binding
	Search     key.Binding
	Palette    key.Binding
	Pause      key.Binding
	Help       key.Binding
	Quit       key.Binding
	SelectAll  key.Binding
	SelectNone key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "navigate up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "navigate down")),
		Left:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "previous")),
		Right:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next")),
		Tab:        key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
		BackTab:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back focus")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back:       key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Toggle:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
		Run:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "run action")),
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Palette:    key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "command palette")),
		Pause:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		SelectAll:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		SelectNone: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "clear")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Toggle, k.Run, k.Search, k.Palette, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Left, k.Right, k.Tab, k.BackTab}, {k.Enter, k.Toggle, k.Run, k.Refresh, k.Search, k.Pause}, {k.SelectAll, k.SelectNone, k.Palette, k.Help, k.Quit}}
}
