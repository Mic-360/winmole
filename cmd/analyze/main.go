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
	barWidth  = 20
	brandName = "◉ WiMo Analyze"
)

// Modern palette + styles
var (
	clrBrand   = lipgloss.Color("#9CB98B")
	clrAccent  = lipgloss.Color("#B5CDA3")
	clrGreen   = lipgloss.Color("#A6E3A1")
	clrYellow  = lipgloss.Color("#F9E2AF")
	clrOrange  = lipgloss.Color("#DBC5A0")
	clrText    = lipgloss.Color("#D5DDD0")
	clrSubtext = lipgloss.Color("#A3AE9E")
	clrOverlay = lipgloss.Color("#6B7466")
	clrSurface = lipgloss.Color("#3A4035")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(clrBrand)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(clrText).
			Background(lipgloss.Color("236"))

	normalStyle = lipgloss.NewStyle().
			Foreground(clrText)

	dimStyle = lipgloss.NewStyle().
			Foreground(clrOverlay)

	sizeStyle = lipgloss.NewStyle().
			Foreground(clrGreen).
			Bold(true)

	largeSizeStyle = lipgloss.NewStyle().
			Foreground(clrOrange).
			Bold(true)

	barFilledStyle = lipgloss.NewStyle().
			Foreground(clrGreen)

	barEmptyStyle = lipgloss.NewStyle().
			Foreground(clrOverlay)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(clrAccent)

	borderStyle = lipgloss.NewStyle().
			Foreground(clrSurface)

	helpStyle = lipgloss.NewStyle().
			Foreground(clrSubtext)

	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrSurface).
			Padding(0, 1)
)

type entry struct {
	name    string
	path    string
	size    int64
	isDir   bool
	percent float64
	ext     string
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
	itemCount  int
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
				ext := strings.ToLower(filepath.Ext(d.Name()))
				mu.Lock()
				results = append(results, entry{
					name:  d.Name(),
					path:  fullPath,
					size:  size,
					isDir: d.IsDir(),
					ext:   ext,
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

	var fillStyle lipgloss.Style
	switch {
	case percent >= 50:
		fillStyle = lipgloss.NewStyle().Foreground(clrOrange)
	case percent >= 25:
		fillStyle = lipgloss.NewStyle().Foreground(clrYellow)
	default:
		fillStyle = barFilledStyle
	}

	bar := fillStyle.Render(strings.Repeat("█", filled)) +
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
		m.itemCount = len(msg.entries)
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

func fileIcon(e entry) string {
	if e.isDir {
		return "📁"
	}
	switch e.ext {
	case ".go", ".py", ".js", ".ts", ".rs", ".c", ".cpp", ".java", ".rb", ".cs":
		return "📜"
	case ".zip", ".tar", ".gz", ".rar", ".7z":
		return "📦"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp":
		return "🖼 "
	case ".mp4", ".avi", ".mkv", ".mov", ".wmv":
		return "🎬"
	case ".mp3", ".wav", ".flac", ".ogg", ".aac":
		return "🎵"
	case ".exe", ".msi", ".dll":
		return "⚙ "
	case ".pdf", ".doc", ".docx", ".txt", ".md":
		return "📝"
	case ".log":
		return "📋"
	default:
		return "📄"
	}
}

func (m model) View() string {
	w := m.width
	if w < 40 {
		w = 80
	}
	cardW := w

	var lines []string

	// Breadcrumb path
	breadcrumb := m.currentDir
	if len(breadcrumb) > 50 {
		parts := strings.Split(breadcrumb, string(os.PathSeparator))
		if len(parts) > 3 {
			breadcrumb = parts[0] + string(os.PathSeparator) + "..." + string(os.PathSeparator) +
				strings.Join(parts[len(parts)-2:], string(os.PathSeparator))
		}
	}

	lines = append(lines,
		titleStyle.Render(brandName)+"  "+dimStyle.Render("·")+"  "+dimStyle.Render(breadcrumb),
	)

	// Stats line
	statsContent := fmt.Sprintf("  %s  ·  %s  ·  Depth: %d",
		sizeStyle.Render(formatSize(m.totalSize)),
		dimStyle.Render(fmt.Sprintf("%d items", m.itemCount)),
		len(m.history))
	lines = append(lines, statsContent)
	lines = append(lines, borderStyle.Render(strings.Repeat("─", max(8, cardW-6))))

	if m.scanning {
		lines = append(lines, dimStyle.Render("Scanning directory tree..."))
	} else if m.err != nil {
		lines = append(lines, lipgloss.NewStyle().Foreground(clrOrange).Render("Error: "+m.err.Error()))
	} else {
		colHdr := fmt.Sprintf(" %3s  %-*s  %-*s  %-4s  %s", "#", barWidth, "Usage", 6, "%", "Type", "Name / Size")
		lines = append(lines, headerStyle.Render(colHdr))
		lines = append(lines, borderStyle.Render(strings.Repeat("─", max(8, cardW-6))))

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
			cursor := "  "
			if i == m.cursor {
				cursor = "▶ "
			}

			// Number
			num := fmt.Sprintf("%2d.", i+1)

			// Progress bar
			bar := renderBar(e.percent)

			// Percent
			pct := fmt.Sprintf("%5.1f%%", e.percent)

			// Icon — file type aware
			icon := fileIcon(e)

			// Name — scale to terminal width
			maxNameLen := max(10, cardW-56)
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

			line := fmt.Sprintf("%s%s  %s  %s  %s %-*s  %s",
				cursor, num, bar, pct, icon, maxNameLen, name, sizeRendered)

			if i == m.cursor {
				lines = append(lines, selectedStyle.Render(line))
			} else {
				lines = append(lines, normalStyle.Render(line))
			}
		}
	}

	lines = append(lines, borderStyle.Render(strings.Repeat("─", max(8, cardW-6))))
	help1 := "  ↑↓/jk navigate  · Enter drill-in · Backspace up · O open"
	shown := min(max(5, m.height-8), len(m.entries))
	help2 := "  F reveal in Explorer · Q quit  · " + dimStyle.Render(fmt.Sprintf("%d/%d shown", shown, len(m.entries)))
	lines = append(lines, helpStyle.Render(help1))
	lines = append(lines, helpStyle.Render(help2))

	return frameStyle.Width(cardW).Render(strings.Join(lines, "\n")) + "\n"
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
