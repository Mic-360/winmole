package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mic-360/wimo/internal/services"
	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
	"github.com/mic-360/wimo/pkg/util"
)

type Model struct {
	services *services.Container
	store    state.Store
	theme    ui.Theme
	keys     ui.KeyMap
	help     help.Model
	spinner  spinner.Model

	width    int
	height   int
	navIndex int
	pending  int

	projectsList  list.Model
	artifactsList list.Model
	cleanList     list.Model
	uninstallList list.Model
	optimizeList  list.Model
	settingsList  list.Model
	helpList      list.Model
	paletteList   list.Model

	logsViewport viewport.Model
	helpViewport viewport.Model

	paletteInput   textinput.Model
	modalInput     textinput.Model
	commandActions []state.CommandAction
}

type listItem struct {
	id          string
	title       string
	description string
	filterValue string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.description }
func (i listItem) FilterValue() string { return i.filterValue }

type tickMsg time.Time

type runtimeLoadedMsg struct {
	snapshot state.RuntimeSnapshot
	err      error
}

type cleanTargetsLoadedMsg struct {
	targets []state.CleanTarget
	err     error
}

type appsLoadedMsg struct {
	apps []state.InstalledApp
	err  error
}

type optimizeLoadedMsg struct {
	tasks []state.OptimizeTask
	err   error
}

type projectsLoadedMsg struct {
	projects []state.Project
	err      error
}

type docsLoadedMsg struct {
	docs []state.HelpDoc
	err  error
}

type cleanRunMsg struct {
	report  services.OperationReport
	targets []state.CleanTarget
	err     error
}

type uninstallRunMsg struct {
	report services.OperationReport
	err    error
}

type optimizeRunMsg struct {
	report services.OperationReport
	tasks  []state.OptimizeTask
	err    error
}

type purgeRunMsg struct {
	report services.OperationReport
	err    error
}

type configSavedMsg struct {
	config state.ConfigState
	err    error
}

func NewModel(container *services.Container, start state.Screen) Model {
	theme := ui.NewTheme()
	keys := ui.DefaultKeyMap()
	helpModel := help.New()
	helpModel.ShowAll = false

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = theme.AccentText

	paletteInput := textinput.New()
	paletteInput.Prompt = "Search command > "
	paletteInput.Placeholder = "dashboard, purge, uninstall, logs..."
	paletteInput.CharLimit = 64
	paletteInput.Focus()

	modalInput := textinput.New()
	modalInput.Prompt = "Value > "
	modalInput.CharLimit = 256

	store := state.NewStore(container.Config.State())
	store.Screen = start
	store.Focus = state.FocusSidebar
	store.StatusText = "Loading runtime, inventories and docs"

	model := Model{
		services:       container,
		store:          store,
		theme:          theme,
		keys:           keys,
		help:           helpModel,
		spinner:        spin,
		width:          120,
		height:         36,
		paletteInput:   paletteInput,
		modalInput:     modalInput,
		commandActions: defaultCommandActions(),
	}
	model.navIndex = model.screenIndex(start)
	model.projectsList = newList("Projects")
	model.artifactsList = newList("Artifacts")
	model.cleanList = newList("Clean")
	model.uninstallList = newList("Installed apps")
	model.optimizeList = newList("Optimize")
	model.settingsList = newList("Settings")
	model.helpList = newList("Docs")
	model.paletteList = newList("Commands")
	model.logsViewport = viewport.New(80, 20)
	model.helpViewport = viewport.New(80, 20)
	model.syncLists()
	return model
}

func newList(title string) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	l := list.New([]list.Item{}, delegate, 40, 20)
	l.Title = title
	l.SetFilteringEnabled(true)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.DisableQuitKeybindings()
	return l
}

func (m Model) Init() tea.Cmd {
	m.pending = 6
	m.store.Busy = true
	return tea.Batch(
		spinner.Tick,
		loadRuntimeCmd(m.services.Runtime),
		loadCleanTargetsCmd(m.services.Cleaner),
		loadAppsCmd(m.services.Uninstaller, m.store.Config.WingetEnabled),
		loadOptimizeCmd(m.services.Optimizer),
		loadProjectsCmd(m.services.Purger, m.store.Config),
		loadDocsCmd(),
		tickCmd(m.services.Config.RefreshInterval()),
	)
}

func (m *Model) screenIndex(screen state.Screen) int {
	for index, item := range m.store.Navigation {
		if item.Screen == screen {
			return index
		}
	}
	return 0
}

func (m *Model) syncLists() {
	m.syncProjectList()
	m.syncArtifactList()
	m.syncCleanList()
	m.syncAppsList()
	m.syncOptimizeList()
	m.syncSettingsList()
	m.syncHelpList()
	m.syncPaletteList()
	m.syncLogsViewport()
	m.syncHelpViewport()
}

func (m *Model) syncProjectList() {
	items := make([]list.Item, 0, len(m.store.Projects))
	selected := 0
	if len(m.store.Projects) > 0 && m.store.SelectedProject == "" {
		m.store.SelectedProject = m.store.Projects[0].ID
	}
	for index, project := range m.store.Projects {
		title := project.Name + "  ·  " + util.FormatBytes(project.TotalArtifactBytes)
		desc := strings.Join(project.Ecosystems, ", ") + "  ·  " + project.Root
		items = append(items, listItem{id: project.ID, title: title, description: desc, filterValue: title + " " + desc})
		if m.store.SelectedProject == project.ID {
			selected = index
		}
	}
	cmd := m.projectsList.SetItems(items)
	_ = cmd
	m.projectsList.Select(selected)
}

func (m *Model) syncArtifactList() {
	project := m.store.SelectedProjectData()
	items := []list.Item{}
	selected := 0
	if project != nil {
		for index, artifact := range project.Artifacts {
			marker := "[ ]"
			if artifact.Selected {
				marker = "[x]"
			}
			title := marker + " " + artifact.Label + "  ·  " + util.FormatBytes(artifact.Size)
			desc := artifact.Type + "  ·  age " + strconv.Itoa(artifact.AgeDays) + "d"
			items = append(items, listItem{id: artifact.ID, title: title, description: desc, filterValue: title + " " + desc})
			if artifact.Selected {
				selected = index
			}
		}
	}
	cmd := m.artifactsList.SetItems(items)
	_ = cmd
	m.artifactsList.Select(selected)
}

func (m *Model) syncCleanList() {
	items := make([]list.Item, 0, len(m.store.CleanTargets))
	for _, target := range m.store.CleanTargets {
		marker := "[ ]"
		if target.Selected {
			marker = "[x]"
		}
		title := marker + " " + target.Title + "  ·  " + util.FormatBytes(target.Size)
		desc := target.Description + "  ·  " + target.Category + "  ·  " + target.Status
		items = append(items, listItem{id: target.ID, title: title, description: desc, filterValue: title + " " + desc})
	}
	cmd := m.cleanList.SetItems(items)
	_ = cmd
}

func (m *Model) syncAppsList() {
	items := make([]list.Item, 0, len(m.store.InstalledApps))
	for _, app := range m.store.InstalledApps {
		marker := "[ ]"
		if app.Selected {
			marker = "[x]"
		}
		title := marker + " " + app.Name
		if app.Version != "" {
			title += "  v" + app.Version
		}
		desc := strings.Join(filterEmpty([]string{app.Publisher, app.Source, util.FormatBytes(app.Size), app.Method}), "  ·  ")
		items = append(items, listItem{id: app.ID, title: title, description: desc, filterValue: title + " " + desc})
	}
	cmd := m.uninstallList.SetItems(items)
	_ = cmd
}

func (m *Model) syncOptimizeList() {
	items := make([]list.Item, 0, len(m.store.OptimizeTasks))
	for _, task := range m.store.OptimizeTasks {
		marker := "[ ]"
		if task.Selected {
			marker = "[x]"
		}
		title := marker + " " + task.Title
		desc := task.Category + "  ·  " + task.Description
		if task.AdminOnly {
			desc += "  ·  admin"
		}
		if task.Status != "" {
			desc += "  ·  " + task.Status
		}
		items = append(items, listItem{id: task.ID, title: title, description: desc, filterValue: title + " " + desc})
	}
	cmd := m.optimizeList.SetItems(items)
	_ = cmd
}

func (m *Model) syncSettingsList() {
	items := []list.Item{
		listItem{id: "scan_paths", title: "Scan paths", description: strings.Join(m.store.Config.ScanPaths, " ; "), filterValue: "scan paths"},
		listItem{id: "purge_depth", title: "Purge depth", description: fmt.Sprintf("%d", m.store.Config.PurgeDepth), filterValue: "purge depth"},
		listItem{id: "refresh_interval", title: "Refresh interval", description: fmt.Sprintf("%d seconds", m.store.Config.RefreshIntervalSeconds), filterValue: "refresh interval"},
		listItem{id: "winget_enabled", title: "Winget integration", description: fmt.Sprintf("%t", m.store.Config.WingetEnabled), filterValue: "winget integration"},
		listItem{id: "check_updates", title: "Update checks", description: fmt.Sprintf("%t", m.store.Config.CheckUpdates), filterValue: "update checks"},
	}
	cmd := m.settingsList.SetItems(items)
	_ = cmd
}

func (m *Model) syncHelpList() {
	items := make([]list.Item, 0, len(m.store.HelpDocs))
	selected := 0
	if len(m.store.HelpDocs) > 0 && m.store.ActiveHelpDoc == "" {
		m.store.ActiveHelpDoc = m.store.HelpDocs[0].ID
	}
	for index, doc := range m.store.HelpDocs {
		items = append(items, listItem{id: doc.ID, title: doc.Title, description: doc.Path, filterValue: doc.Title})
		if m.store.ActiveHelpDoc == doc.ID {
			selected = index
		}
	}
	cmd := m.helpList.SetItems(items)
	_ = cmd
	m.helpList.Select(selected)
}

func (m *Model) syncPaletteList() {
	query := strings.ToLower(strings.TrimSpace(m.paletteInput.Value()))
	items := make([]list.Item, 0, len(m.commandActions))
	for _, action := range m.commandActions {
		haystack := action.Title + " " + action.Description + " " + strings.Join(action.Keywords, " ")
		if query != "" && !util.FuzzyMatch(query, haystack) {
			continue
		}
		items = append(items, listItem{id: action.ID, title: action.Title, description: action.Description, filterValue: haystack})
	}
	cmd := m.paletteList.SetItems(items)
	_ = cmd
}

func (m *Model) syncLogsViewport() {
	entries := m.services.Logger.Entries()
	m.store.Logs = entries
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if m.store.LogQuery != "" {
			query := strings.ToLower(m.store.LogQuery)
			if !strings.Contains(strings.ToLower(entry.Message), query) && !strings.Contains(strings.ToLower(entry.Source), query) {
				continue
			}
		}
		style := m.theme.MutedText
		switch entry.Level {
		case state.LogInfo:
			style = m.theme.AccentText
		case state.LogWarn:
			style = m.theme.WarningText
		case state.LogError:
			style = m.theme.ErrorText
		}
		lines = append(lines, style.Render("["+strings.ToUpper(string(entry.Level))+"]")+" "+m.theme.MutedText.Render(entry.Time.Format("15:04:05"))+"  "+entry.Source+"  "+entry.Message)
	}
	m.logsViewport.SetContent(strings.Join(lines, "\n"))
	if !m.store.LogPaused {
		m.logsViewport.GotoBottom()
	}
}

func (m *Model) syncHelpViewport() {
	doc := m.store.SelectedHelpDoc()
	if doc == nil {
		m.helpViewport.SetContent("No help documents loaded")
		return
	}
	m.helpViewport.SetContent(doc.Content)
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && trimmed != "0 B" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func defaultCommandActions() []state.CommandAction {
	return []state.CommandAction{
		{ID: "open-dashboard", Title: "Open dashboard", Description: "Jump to live system dashboard", Screen: state.ScreenDashboard, Keywords: []string{"status", "health", "dashboard"}},
		{ID: "open-projects", Title: "Open projects", Description: "Jump to project analyzer and purge screen", Screen: state.ScreenProjects, Keywords: []string{"projects", "analyze", "purge"}},
		{ID: "open-actions", Title: "Open actions", Description: "Jump to clean, uninstall and optimize workflows", Screen: state.ScreenActions, Keywords: []string{"clean", "uninstall", "optimize"}},
		{ID: "open-logs", Title: "Open logs", Description: "Open runtime logs viewer", Screen: state.ScreenLogs, Keywords: []string{"logs", "events"}},
		{ID: "open-settings", Title: "Open settings", Description: "Edit scan paths, depth and refresh", Screen: state.ScreenSettings, Keywords: []string{"settings", "config"}},
		{ID: "open-help", Title: "Open help", Description: "Open markdown documentation", Screen: state.ScreenHelp, Keywords: []string{"help", "docs"}},
		{ID: "refresh-runtime", Title: "Refresh runtime data", Description: "Refresh dashboard metrics", Keywords: []string{"refresh", "status"}},
		{ID: "scan-projects", Title: "Scan projects", Description: "Refresh project inventory and artifact detection", Keywords: []string{"projects", "scan", "purge"}},
		{ID: "scan-clean", Title: "Rescan clean targets", Description: "Refresh clean target sizes", Keywords: []string{"clean", "scan"}},
	}
}
