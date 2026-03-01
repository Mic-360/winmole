package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	version   = "1.0.0"
	maxItems  = 50
	barWidth  = 24
	brandName = "🐹 WiMo Analyze"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("208"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("255"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	sizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	largeSizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)

	barFilledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	barEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("51"))

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

type entry struct {
	name    string
	path    string
	size    int64
	isDir   bool
	percent float64
}

type model struct {
	entries    []entry
	cursor     int
	currentDir string
	history    []string
	totalSize  int64
	width      int
	height     int
	scanning   bool
	err        error
}

type scanDoneMsg struct {
	entries   []entry
	totalSize int64
	err       error
}

func initialModel() model {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "C:\\"
	}

	return model{
		currentDir: home,
		cursor:     0,
		history:    []string{},
		scanning:   true,
		width:      80,
		height:     24,
	}
}

func scanDir(dir string) tea.Cmd {
	return func() tea.Msg {
		entries, totalSize, err := scanDirectory(dir)
		return scanDoneMsg{entries: entries, totalSize: totalSize, err: err}
	}
}

func scanDirectory(dir string) ([]entry, int64, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []entry
	)

	// Use a semaphore to limit concurrent goroutines
	sem := make(chan struct{}, runtime.NumCPU()*2)

	for _, de := range dirEntries {
		wg.Add(1)
		go func(d fs.DirEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullPath := filepath.Join(dir, d.Name())
			var size int64

			if d.IsDir() {
				size = getDirSize(fullPath)
			} else {
				info, err := d.Info()
				if err == nil {
					size = info.Size()
				}
			}

			if size > 0 {
				mu.Lock()
				results = append(results, entry{
					name:  d.Name(),
					path:  fullPath,
					size:  size,
					isDir: d.IsDir(),
				})
				mu.Unlock()
			}
		}(de)
	}

	wg.Wait()

	// Sort by size descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].size > results[j].size
	})

	// Limit to maxItems
	if len(results) > maxItems {
		results = results[:maxItems]
	}

	// Calculate total and percentages
	var totalSize int64
	for _, e := range results {
		totalSize += e.size
	}
	for i := range results {
		if totalSize > 0 {
			results[i].percent = float64(results[i].size) / float64(totalSize) * 100
		}
	}

	return results, totalSize, nil
}

func getDirSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func renderBar(percent float64) string {
	filled := int(percent / 100 * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	bar := barFilledStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", empty))

	return bar
}

func (m model) Init() tea.Cmd {
	return scanDir(m.currentDir)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case scanDoneMsg:
		m.scanning = false
		m.err = msg.err
		m.entries = msg.entries
		m.totalSize = msg.totalSize
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}

		case "enter", "l", "right":
			if len(m.entries) > 0 && m.entries[m.cursor].isDir {
				m.history = append(m.history, m.currentDir)
				m.currentDir = m.entries[m.cursor].path
				m.scanning = true
				return m, scanDir(m.currentDir)
			}

		case "backspace", "h", "left":
			if len(m.history) > 0 {
				m.currentDir = m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
				m.scanning = true
				return m, scanDir(m.currentDir)
			} else {
				parent := filepath.Dir(m.currentDir)
				if parent != m.currentDir {
					m.history = append(m.history, m.currentDir)
					m.currentDir = parent
					m.scanning = true
					return m, scanDir(m.currentDir)
				}
			}

		case "o":
			// Open in Explorer
			if len(m.entries) > 0 {
				path := m.entries[m.cursor].path
				exec.Command("explorer.exe", path).Start()
			}

		case "f":
			// Reveal in Explorer
			if len(m.entries) > 0 {
				path := m.entries[m.cursor].path
				exec.Command("explorer.exe", "/select,", path).Start()
			}

		case "delete":
			// Delete with confirmation (handled via placeholder)
			// In a real impl, we'd show a confirmation dialog
		}
	}

	return m, nil
}

func padRight(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func (m model) View() string {
	var b strings.Builder
	w := m.width
	if w < 40 {
		w = 80
	}
	innerW := w - 4 // account for "│  " and " │"

	// Header
	top := borderStyle.Render("╭" + strings.Repeat("─", w-2) + "╮")
	b.WriteString(top + "\n")

	headerContent := "  " + titleStyle.Render(brandName) + "  " +
		dimStyle.Render("·") + "  " + dimStyle.Render(truncatePath(m.currentDir, 30)) +
		"  " + dimStyle.Render("·") + "  Total: " + sizeStyle.Render(formatSize(m.totalSize))
	headerLine := padRight(headerContent, w-2)
	b.WriteString(borderStyle.Render("│") + headerLine + borderStyle.Render("│") + "\n")

	sep := borderStyle.Render("├" + strings.Repeat("─", w-2) + "┤")
	b.WriteString(sep + "\n")

	if m.scanning {
		scanLine := padRight("  Scanning...", w-2)
		b.WriteString(borderStyle.Render("│") + scanLine + borderStyle.Render("│") + "\n")
	} else if m.err != nil {
		errLine := padRight("  Error: "+m.err.Error(), w-2)
		b.WriteString(borderStyle.Render("│") + errLine + borderStyle.Render("│") + "\n")
	} else {
		// Calculate visible items
		visibleHeight := m.height - 8
		if visibleHeight < 5 {
			visibleHeight = 5
		}

		scrollStart := 0
		if m.cursor >= visibleHeight {
			scrollStart = m.cursor - visibleHeight + 1
		}

		for i := scrollStart; i < len(m.entries) && i < scrollStart+visibleHeight; i++ {
			e := m.entries[i]

			// Cursor indicator
			cursor := "   "
			if i == m.cursor {
				cursor = " ▶ "
			}

			// Number
			num := fmt.Sprintf("%2d.", i+1)

			// Progress bar
			bar := renderBar(e.percent)

			// Percent
			pct := fmt.Sprintf("%5.1f%%", e.percent)

			// Icon
			icon := "📄"
			if e.isDir {
				icon = "📁"
			}

			// Name — scale to terminal width
			maxNameLen := innerW - 55 // space for cursor+num+bar+pct+icon+size
			if maxNameLen < 10 {
				maxNameLen = 10
			}
			name := e.name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			// Size
			sizeStr := formatSize(e.size)
			sizeRendered := sizeStyle.Render(sizeStr)
			if e.size > 1024*1024*1024 { // > 1GB
				sizeRendered = largeSizeStyle.Render(sizeStr)
			}

			line := fmt.Sprintf("%s %s  %s  %s  %s %-*s  %s",
				cursor, num, bar, pct, icon, maxNameLen, name, sizeRendered)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render(padRight(line, w-2)))
			} else {
				b.WriteString(normalStyle.Render(padRight(line, w-2)))
			}
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString(sep + "\n")
	help1 := "  ↑↓/jk navigate  · Enter drill-in · Backspace up · O open"
	help2 := "  F reveal in Explorer · Q quit"
	b.WriteString(borderStyle.Render("│") + padRight(helpStyle.Render(help1), w-2) + borderStyle.Render("│") + "\n")
	b.WriteString(borderStyle.Render("│") + padRight(helpStyle.Render(help2), w-2) + borderStyle.Render("│") + "\n")
	bottom := borderStyle.Render("╰" + strings.Repeat("─", w-2) + "╯")
	b.WriteString(bottom + "\n")

	return b.String()
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
