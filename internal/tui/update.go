package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
	"github.com/mic-360/wimo/pkg/util"
)

var focusCycle = []state.FocusArea{state.FocusSidebar, state.FocusContent}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var spinCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinCmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeComponents()
		return m, tea.Batch(cmds...)
	case tickMsg:
		if !m.store.LogPaused {
			m.syncLogsViewport()
		}
		cmds = append(cmds, loadRuntimeCmd(m.services.Runtime), tickCmd(m.services.Config.RefreshInterval()))
		return m, tea.Batch(cmds...)
	case runtimeLoadedMsg:
		m.finishPending(msg.err)
		if msg.err == nil {
			m.store.Runtime = msg.snapshot
			m.store.StatusText = "Runtime refreshed"
		}
		m.syncLogsViewport()
		return m, tea.Batch(cmds...)
	case cleanTargetsLoadedMsg:
		m.finishPending(msg.err)
		if msg.err == nil {
			m.store.CleanTargets = msg.targets
			m.store.StatusText = "Cleanup inventory refreshed"
			m.syncCleanList()
		}
		return m, tea.Batch(cmds...)
	case appsLoadedMsg:
		m.finishPending(msg.err)
		if msg.err == nil {
			m.store.InstalledApps = msg.apps
			m.store.StatusText = fmt.Sprintf("Loaded %d uninstallable apps", len(msg.apps))
			m.syncAppsList()
		}
		return m, tea.Batch(cmds...)
	case optimizeLoadedMsg:
		m.finishPending(msg.err)
		if msg.err == nil {
			m.store.OptimizeTasks = msg.tasks
			m.store.StatusText = "Optimization tasks ready"
			m.syncOptimizeList()
		}
		return m, tea.Batch(cmds...)
	case projectsLoadedMsg:
		m.finishPending(msg.err)
		if msg.err == nil {
			m.store.Projects = msg.projects
			if len(msg.projects) > 0 && m.store.SelectedProject == "" {
				m.store.SelectedProject = msg.projects[0].ID
			}
			m.store.StatusText = fmt.Sprintf("Discovered %d projects", len(msg.projects))
			m.syncProjectList()
			m.syncArtifactList()
		}
		return m, tea.Batch(cmds...)
	case docsLoadedMsg:
		m.finishPending(msg.err)
		if msg.err == nil {
			m.store.HelpDocs = msg.docs
			if len(msg.docs) > 0 {
				m.store.ActiveHelpDoc = msg.docs[0].ID
			}
			m.syncHelpList()
			m.syncHelpViewport()
		}
		return m, tea.Batch(cmds...)
	case cleanRunMsg:
		m.store.Busy = false
		if msg.err != nil {
			m.openAlert("Cleanup failed", msg.err.Error())
			return m, tea.Batch(cmds...)
		}
		m.store.CleanTargets = msg.targets
		m.store.StatusText = fmt.Sprintf("Cleanup finished · %d targets · %s", msg.report.Count, formatBytes(msg.report.Bytes))
		m.syncCleanList()
		m.syncLogsViewport()
		return m, tea.Batch(cmds...)
	case uninstallRunMsg:
		m.store.Busy = false
		if msg.err != nil {
			m.openAlert("Uninstall failed", msg.err.Error())
			return m, tea.Batch(cmds...)
		}
		m.store.StatusText = fmt.Sprintf("Removed %d apps", msg.report.Count)
		cmds = append(cmds, loadAppsCmd(m.services.Uninstaller, m.store.Config.WingetEnabled))
		return m, tea.Batch(cmds...)
	case optimizeRunMsg:
		m.store.Busy = false
		if msg.err != nil {
			m.openAlert("Optimization failed", msg.err.Error())
			return m, tea.Batch(cmds...)
		}
		m.store.OptimizeTasks = msg.tasks
		m.store.StatusText = fmt.Sprintf("Optimization complete · %d tasks", msg.report.Count)
		m.syncOptimizeList()
		m.syncLogsViewport()
		return m, tea.Batch(cmds...)
	case purgeRunMsg:
		m.store.Busy = false
		if msg.err != nil {
			m.openAlert("Purge failed", msg.err.Error())
			return m, tea.Batch(cmds...)
		}
		m.store.StatusText = fmt.Sprintf("Purged %d artifacts · %s", msg.report.Count, formatBytes(msg.report.Bytes))
		cmds = append(cmds, loadProjectsCmd(m.services.Purger, m.store.Config))
		return m, tea.Batch(cmds...)
	case configSavedMsg:
		m.store.Busy = false
		if msg.err != nil {
			m.openAlert("Save failed", msg.err.Error())
			return m, tea.Batch(cmds...)
		}
		m.store.Config = msg.config
		m.store.StatusText = "Settings saved"
		m.syncSettingsList()
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		if m.store.Modal.Visible {
			return m.updateModal(msg, cmds)
		}
		if m.store.Palette.Visible {
			return m.updatePalette(msg, cmds)
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+p":
			m.store.Palette.Visible = true
			m.store.Focus = state.FocusPalette
			m.paletteInput.SetValue("")
			m.paletteInput.Focus()
			m.syncPaletteList()
			return m, tea.Batch(cmds...)
		case "?":
			m.setScreen(state.ScreenHelp)
			m.store.Focus = state.FocusContent
			return m, tea.Batch(cmds...)
		case "tab":
			m.toggleFocus(true)
			return m, tea.Batch(cmds...)
		case "shift+tab":
			m.toggleFocus(false)
			return m, tea.Batch(cmds...)
		case "q":
			if m.store.Focus == state.FocusSidebar {
				return m, tea.Quit
			}
		}
		if m.store.Focus == state.FocusSidebar {
			return m.updateSidebar(msg, cmds)
		}
		return m.updateContent(msg, cmds)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) updateSidebar(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.navIndex > 0 {
			m.navIndex--
		}
	case "down", "j":
		if m.navIndex < len(m.store.Navigation)-1 {
			m.navIndex++
		}
	case "enter":
		m.setScreen(m.store.Navigation[m.navIndex].Screen)
		m.store.Focus = state.FocusContent
	}
	return *m, tea.Batch(cmds...)
}

func (m *Model) updateContent(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if current := m.activeList(); current != nil && current.FilterState() == list.Filtering {
		m.updateCurrentList(msg)
		m.syncSelectionFromLists()
		return *m, tea.Batch(cmds...)
	}
	switch m.store.Screen {
	case state.ScreenDashboard:
		if msg.String() == "r" {
			m.store.Busy = true
			cmds = append(cmds, loadRuntimeCmd(m.services.Runtime))
		}
	case state.ScreenProjects:
		return m.updateProjects(msg, cmds)
	case state.ScreenActions:
		return m.updateActions(msg, cmds)
	case state.ScreenLogs:
		return m.updateLogs(msg, cmds)
	case state.ScreenSettings:
		return m.updateSettings(msg, cmds)
	case state.ScreenHelp:
		return m.updateHelp(msg, cmds)
	}
	return *m, tea.Batch(cmds...)
}

func (m *Model) updateProjects(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.store.Busy = true
		cmds = append(cmds, loadProjectsCmd(m.services.Purger, m.store.Config))
		return *m, tea.Batch(cmds...)
	case "enter":
		if m.store.ProjectMode == state.ProjectsModeInventory {
			m.store.ProjectMode = state.ProjectsModeArtifacts
			m.syncArtifactList()
			return *m, tea.Batch(cmds...)
		}
	case "esc", "backspace":
		if m.store.ProjectMode == state.ProjectsModeArtifacts {
			m.store.ProjectMode = state.ProjectsModeInventory
			return *m, tea.Batch(cmds...)
		}
	case " ":
		if m.store.ProjectMode == state.ProjectsModeArtifacts {
			m.toggleArtifactSelection(m.selectedItemID(&m.artifactsList))
			m.syncArtifactList()
			return *m, tea.Batch(cmds...)
		}
	case "a":
		if m.store.ProjectMode == state.ProjectsModeArtifacts {
			m.setAllArtifacts(true)
			m.syncArtifactList()
			return *m, tea.Batch(cmds...)
		}
	case "n":
		if m.store.ProjectMode == state.ProjectsModeArtifacts {
			m.setAllArtifacts(false)
			m.syncArtifactList()
			return *m, tea.Batch(cmds...)
		}
	case "x":
		if m.store.ProjectMode == state.ProjectsModeArtifacts && m.selectedArtifactCount() > 0 {
			m.openConfirm("run-purge", "Purge selected artifacts", fmt.Sprintf("Remove %d selected build artifacts?", m.selectedArtifactCount()))
			return *m, tea.Batch(cmds...)
		}
	}
	if m.store.ProjectMode == state.ProjectsModeArtifacts {
		m.artifactsList, _ = m.artifactsList.Update(msg)
	} else {
		m.projectsList, _ = m.projectsList.Update(msg)
	}
	m.syncSelectionFromLists()
	return *m, tea.Batch(cmds...)
}

func (m *Model) updateActions(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		m.previousActionPane()
		return *m, tea.Batch(cmds...)
	case "right", "l":
		m.nextActionPane()
		return *m, tea.Batch(cmds...)
	case "r":
		m.store.Busy = true
		cmds = append(cmds, m.reloadActiveActionCmd())
		return *m, tea.Batch(cmds...)
	case " ":
		m.toggleCurrentSelection()
		m.syncCurrentActionList()
		return *m, tea.Batch(cmds...)
	case "a":
		m.setAllCurrent(true)
		m.syncCurrentActionList()
		return *m, tea.Batch(cmds...)
	case "n":
		m.setAllCurrent(false)
		m.syncCurrentActionList()
		return *m, tea.Batch(cmds...)
	case "x":
		if m.currentSelectionCount() > 0 {
			m.openConfirm("run-action", "Execute selected workflow", fmt.Sprintf("Run %d selected items from %s?", m.currentSelectionCount(), strings.Title(string(m.store.ActiveActionPane))))
			return *m, tea.Batch(cmds...)
		}
	}
	m.updateCurrentList(msg)
	m.syncSelectionFromLists()
	return *m, tea.Batch(cmds...)
}

func (m *Model) updateLogs(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.openInput("log-filter", "Log search", "Filter logs by source or message", m.store.LogQuery)
		return *m, tea.Batch(cmds...)
	case "p":
		m.store.LogPaused = !m.store.LogPaused
		m.syncLogsViewport()
		return *m, tea.Batch(cmds...)
	case "r":
		m.syncLogsViewport()
		return *m, tea.Batch(cmds...)
	}
	m.logsViewport, _ = m.logsViewport.Update(msg)
	return *m, tea.Batch(cmds...)
}

func (m *Model) updateSettings(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		id := m.selectedItemID(&m.settingsList)
		m.openSettingEditor(id)
		return *m, tea.Batch(cmds...)
	case " ":
		id := m.selectedItemID(&m.settingsList)
		if id == "winget_enabled" {
			m.store.Config.WingetEnabled = !m.store.Config.WingetEnabled
			m.store.Busy = true
			cmds = append(cmds, saveConfigCmd(m.services.Config, m.store.Config))
			return *m, tea.Batch(cmds...)
		}
		if id == "check_updates" {
			m.store.Config.CheckUpdates = !m.store.Config.CheckUpdates
			m.store.Busy = true
			cmds = append(cmds, saveConfigCmd(m.services.Config, m.store.Config))
			return *m, tea.Batch(cmds...)
		}
	}
	m.settingsList, _ = m.settingsList.Update(msg)
	return *m, tea.Batch(cmds...)
}

func (m *Model) updateHelp(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "pgdown", "ctrl+f":
		m.helpViewport.PageDown()
		return *m, tea.Batch(cmds...)
	case "pgup", "ctrl+b":
		m.helpViewport.PageUp()
		return *m, tea.Batch(cmds...)
	}
	m.helpList, _ = m.helpList.Update(msg)
	m.syncSelectionFromLists()
	m.helpViewport, _ = m.helpViewport.Update(msg)
	return *m, tea.Batch(cmds...)
}

func (m *Model) updatePalette(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.store.Palette.Visible = false
		m.store.Focus = state.FocusContent
		return *m, tea.Batch(cmds...)
	case "enter":
		id := m.selectedItemID(&m.paletteList)
		m.store.Palette.Visible = false
		m.store.Focus = state.FocusContent
		cmds = append(cmds, m.executeCommandAction(id))
		return *m, tea.Batch(cmds...)
	case "up", "down", "j", "k":
		m.paletteList, _ = m.paletteList.Update(msg)
		return *m, tea.Batch(cmds...)
	default:
		m.paletteInput, _ = m.paletteInput.Update(msg)
		m.syncPaletteList()
		return *m, tea.Batch(cmds...)
	}
}

func (m *Model) updateModal(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if m.store.Modal.Kind == "input" {
		switch msg.String() {
		case "esc":
			m.closeModal()
			return *m, tea.Batch(cmds...)
		case "enter":
			value := strings.TrimSpace(m.modalInput.Value())
			pending := m.store.Modal.Pending
			m.closeModal()
			cmds = append(cmds, m.applyModalValue(pending, value))
			return *m, tea.Batch(cmds...)
		default:
			m.modalInput, _ = m.modalInput.Update(msg)
			return *m, tea.Batch(cmds...)
		}
	}
	switch msg.String() {
	case "esc", "n":
		m.closeModal()
	case "enter", "y":
		if m.store.Modal.Kind == "alert" {
			m.closeModal()
		} else {
			pending := m.store.Modal.Pending
			m.closeModal()
			cmds = append(cmds, m.executePending(pending))
		}
	}
	return *m, tea.Batch(cmds...)
}

func (m *Model) resizeComponents() {
	layout := ui.ComputeLayout(m.width, m.height)
	listWidth := util.Max(32, layout.ContentWidth/2-2)
	listHeight := util.Max(10, layout.BodyHeight-4)
	m.projectsList.SetSize(listWidth, listHeight)
	m.artifactsList.SetSize(listWidth, listHeight)
	m.cleanList.SetSize(util.Max(44, layout.ContentWidth-4), listHeight)
	m.uninstallList.SetSize(util.Max(44, layout.ContentWidth-4), listHeight)
	m.optimizeList.SetSize(util.Max(44, layout.ContentWidth-4), listHeight)
	m.settingsList.SetSize(util.Max(40, layout.ContentWidth-4), listHeight)
	m.helpList.SetSize(util.Max(32, layout.ContentWidth/3), listHeight)
	m.paletteList.SetSize(util.Max(48, layout.ContentWidth/2), util.Max(10, listHeight-4))
	m.logsViewport.Width = util.Max(40, layout.ContentWidth-6)
	m.logsViewport.Height = util.Max(8, layout.BodyHeight-8)
	m.helpViewport.Width = util.Max(40, layout.ContentWidth/2)
	m.helpViewport.Height = util.Max(8, layout.BodyHeight-8)
	m.help.Width = m.width
}

func (m *Model) finishPending(err error) {
	if m.pending > 0 {
		m.pending--
	}
	m.store.Busy = m.pending > 0
	if err != nil {
		m.openAlert("Background task failed", err.Error())
	}
}

func (m *Model) setScreen(screen state.Screen) {
	m.store.Screen = screen
	m.navIndex = m.screenIndex(screen)
	m.store.StatusText = m.store.Navigation[m.navIndex].Description
	if screen == state.ScreenProjects {
		m.store.ProjectMode = state.ProjectsModeInventory
	}
}

func (m *Model) toggleFocus(forward bool) {
	current := 0
	for i, f := range focusCycle {
		if f == m.store.Focus {
			current = i
			break
		}
	}
	if forward {
		m.store.Focus = focusCycle[(current+1)%len(focusCycle)]
	} else {
		m.store.Focus = focusCycle[(current-1+len(focusCycle))%len(focusCycle)]
	}
}

func (m *Model) activeList() *list.Model {
	switch m.store.Screen {
	case state.ScreenProjects:
		if m.store.ProjectMode == state.ProjectsModeArtifacts {
			return &m.artifactsList
		}
		return &m.projectsList
	case state.ScreenActions:
		switch m.store.ActiveActionPane {
		case state.ActionPaneUninstall:
			return &m.uninstallList
		case state.ActionPaneOptimize:
			return &m.optimizeList
		default:
			return &m.cleanList
		}
	case state.ScreenSettings:
		return &m.settingsList
	case state.ScreenHelp:
		return &m.helpList
	default:
		return nil
	}
}

func (m *Model) updateCurrentList(msg tea.KeyMsg) {
	if current := m.activeList(); current != nil {
		*current, _ = current.Update(msg)
	}
}

func (m *Model) syncSelectionFromLists() {
	if m.store.Screen == state.ScreenProjects && m.store.ProjectMode == state.ProjectsModeInventory {
		if id := m.selectedItemID(&m.projectsList); id != "" {
			m.store.SelectedProject = id
			m.syncArtifactList()
		}
	}
	if m.store.Screen == state.ScreenHelp {
		if id := m.selectedItemID(&m.helpList); id != "" {
			m.store.ActiveHelpDoc = id
			m.syncHelpViewport()
		}
	}
}

func (m *Model) selectedItemID(l *list.Model) string {
	if l == nil {
		return ""
	}
	item, ok := l.SelectedItem().(listItem)
	if !ok {
		return ""
	}
	return item.id
}

func (m *Model) previousActionPane() {
	switch m.store.ActiveActionPane {
	case state.ActionPaneUninstall:
		m.store.ActiveActionPane = state.ActionPaneClean
	case state.ActionPaneOptimize:
		m.store.ActiveActionPane = state.ActionPaneUninstall
	}
}

func (m *Model) nextActionPane() {
	switch m.store.ActiveActionPane {
	case state.ActionPaneClean:
		m.store.ActiveActionPane = state.ActionPaneUninstall
	case state.ActionPaneUninstall:
		m.store.ActiveActionPane = state.ActionPaneOptimize
	}
}

func (m *Model) toggleCurrentSelection() {
	switch m.store.ActiveActionPane {
	case state.ActionPaneClean:
		id := m.selectedItemID(&m.cleanList)
		for index := range m.store.CleanTargets {
			if m.store.CleanTargets[index].ID == id {
				m.store.CleanTargets[index].Selected = !m.store.CleanTargets[index].Selected
			}
		}
	case state.ActionPaneUninstall:
		id := m.selectedItemID(&m.uninstallList)
		for index := range m.store.InstalledApps {
			if m.store.InstalledApps[index].ID == id {
				m.store.InstalledApps[index].Selected = !m.store.InstalledApps[index].Selected
			}
		}
	case state.ActionPaneOptimize:
		id := m.selectedItemID(&m.optimizeList)
		for index := range m.store.OptimizeTasks {
			if m.store.OptimizeTasks[index].ID == id {
				m.store.OptimizeTasks[index].Selected = !m.store.OptimizeTasks[index].Selected
			}
		}
	}
}

func (m *Model) setAllCurrent(selected bool) {
	switch m.store.ActiveActionPane {
	case state.ActionPaneClean:
		for index := range m.store.CleanTargets {
			m.store.CleanTargets[index].Selected = selected
		}
	case state.ActionPaneUninstall:
		for index := range m.store.InstalledApps {
			m.store.InstalledApps[index].Selected = selected
		}
	case state.ActionPaneOptimize:
		for index := range m.store.OptimizeTasks {
			m.store.OptimizeTasks[index].Selected = selected
		}
	}
}

func (m *Model) syncCurrentActionList() {
	switch m.store.ActiveActionPane {
	case state.ActionPaneClean:
		m.syncCleanList()
	case state.ActionPaneUninstall:
		m.syncAppsList()
	case state.ActionPaneOptimize:
		m.syncOptimizeList()
	}
}

func (m *Model) currentSelectionCount() int {
	count := 0
	switch m.store.ActiveActionPane {
	case state.ActionPaneClean:
		for _, item := range m.store.CleanTargets {
			if item.Selected {
				count++
			}
		}
	case state.ActionPaneUninstall:
		for _, item := range m.store.InstalledApps {
			if item.Selected {
				count++
			}
		}
	case state.ActionPaneOptimize:
		for _, item := range m.store.OptimizeTasks {
			if item.Selected {
				count++
			}
		}
	}
	return count
}

func (m *Model) reloadActiveActionCmd() tea.Cmd {
	switch m.store.ActiveActionPane {
	case state.ActionPaneUninstall:
		return loadAppsCmd(m.services.Uninstaller, m.store.Config.WingetEnabled)
	case state.ActionPaneOptimize:
		return loadOptimizeCmd(m.services.Optimizer)
	default:
		return loadCleanTargetsCmd(m.services.Cleaner)
	}
}

func (m *Model) toggleArtifactSelection(id string) {
	for pIndex := range m.store.Projects {
		if m.store.Projects[pIndex].ID != m.store.SelectedProject {
			continue
		}
		for aIndex := range m.store.Projects[pIndex].Artifacts {
			if m.store.Projects[pIndex].Artifacts[aIndex].ID == id {
				m.store.Projects[pIndex].Artifacts[aIndex].Selected = !m.store.Projects[pIndex].Artifacts[aIndex].Selected
			}
		}
	}
}

func (m *Model) setAllArtifacts(selected bool) {
	for pIndex := range m.store.Projects {
		if m.store.Projects[pIndex].ID != m.store.SelectedProject {
			continue
		}
		for aIndex := range m.store.Projects[pIndex].Artifacts {
			m.store.Projects[pIndex].Artifacts[aIndex].Selected = selected
		}
	}
}

func (m *Model) selectedArtifactCount() int {
	count := 0
	project := m.store.SelectedProjectData()
	if project == nil {
		return 0
	}
	for _, artifact := range project.Artifacts {
		if artifact.Selected {
			count++
		}
	}
	return count
}

func (m *Model) openConfirm(kind, title, body string) {
	m.store.Modal = state.ModalState{Visible: true, Kind: "confirm", Title: title, Body: body, ConfirmLabel: "Confirm", CancelLabel: "Cancel", Pending: state.PendingAction{Kind: kind}}
	m.store.Focus = state.FocusModal
}

func (m *Model) openInput(kind, title, body, value string) {
	m.store.Modal = state.ModalState{Visible: true, Kind: "input", Title: title, Body: body, Pending: state.PendingAction{Kind: kind}}
	m.modalInput.SetValue(value)
	m.modalInput.Focus()
	m.store.Focus = state.FocusModal
}

func (m *Model) openAlert(title, body string) {
	m.store.Modal = state.ModalState{Visible: true, Kind: "alert", Title: title, Body: body, ConfirmLabel: "Close"}
	m.store.Focus = state.FocusModal
}

func (m *Model) closeModal() {
	m.store.Modal = state.ModalState{}
	m.store.Focus = state.FocusContent
	m.modalInput.Blur()
}

func (m *Model) openSettingEditor(id string) {
	switch id {
	case "scan_paths":
		m.openInput(id, "Edit scan paths", "Comma-separated project roots for the Projects screen", strings.Join(m.store.Config.ScanPaths, ", "))
	case "purge_depth":
		m.openInput(id, "Edit purge depth", "Maximum recursive depth for project discovery", strconv.Itoa(m.store.Config.PurgeDepth))
	case "refresh_interval":
		m.openInput(id, "Edit refresh interval", "Dashboard refresh interval in seconds", strconv.Itoa(m.store.Config.RefreshIntervalSeconds))
	}
}

func (m *Model) applyModalValue(pending state.PendingAction, value string) tea.Cmd {
	cfg := m.store.Config
	switch pending.Kind {
	case "log-filter":
		m.store.LogQuery = value
		m.syncLogsViewport()
		return nil
	case "scan_paths":
		cfg.ScanPaths = splitAndTrim(value)
	case "purge_depth":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			m.openAlert("Invalid value", "Purge depth must be an integer")
			return nil
		}
		cfg.PurgeDepth = parsed
	case "refresh_interval":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			m.openAlert("Invalid value", "Refresh interval must be an integer")
			return nil
		}
		cfg.RefreshIntervalSeconds = parsed
	default:
		return nil
	}
	m.store.Busy = true
	return saveConfigCmd(m.services.Config, cfg)
}

func (m *Model) executePending(pending state.PendingAction) tea.Cmd {
	m.store.Busy = true
	switch pending.Kind {
	case "run-purge":
		artifacts := []state.Artifact{}
		project := m.store.SelectedProjectData()
		if project != nil {
			artifacts = project.Artifacts
		}
		return runPurgeCmd(m.services.Purger, artifacts)
	case "run-action":
		switch m.store.ActiveActionPane {
		case state.ActionPaneClean:
			return runCleanCmd(m.services.Cleaner, m.store.CleanTargets)
		case state.ActionPaneUninstall:
			return runUninstallCmd(m.services.Uninstaller, m.store.InstalledApps)
		case state.ActionPaneOptimize:
			return runOptimizeCmd(m.services.Optimizer, m.store.OptimizeTasks)
		}
	}
	return nil
}

func (m *Model) executeCommandAction(id string) tea.Cmd {
	switch id {
	case "open-dashboard":
		m.setScreen(state.ScreenDashboard)
	case "open-projects":
		m.setScreen(state.ScreenProjects)
	case "open-actions":
		m.setScreen(state.ScreenActions)
	case "open-logs":
		m.setScreen(state.ScreenLogs)
	case "open-settings":
		m.setScreen(state.ScreenSettings)
	case "open-help":
		m.setScreen(state.ScreenHelp)
	case "refresh-runtime":
		m.store.Busy = true
		return loadRuntimeCmd(m.services.Runtime)
	case "scan-projects":
		m.store.Busy = true
		return loadProjectsCmd(m.services.Purger, m.store.Config)
	case "scan-clean":
		m.store.Busy = true
		return loadCleanTargetsCmd(m.services.Cleaner)
	}
	return nil
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func formatBytes(value int64) string {
	return util.FormatBytes(value)
}
