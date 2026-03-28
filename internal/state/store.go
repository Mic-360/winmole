package state

import "time"

type Screen string

type FocusArea string

type ActionPane string

type ProjectsMode string

const (
	ScreenDashboard Screen = "dashboard"
	ScreenProjects  Screen = "projects"
	ScreenActions   Screen = "actions"
	ScreenLogs      Screen = "logs"
	ScreenSettings  Screen = "settings"
	ScreenHelp      Screen = "help"
)

const (
	FocusSidebar FocusArea = "sidebar"
	FocusContent FocusArea = "content"
	FocusModal   FocusArea = "modal"
	FocusPalette FocusArea = "palette"
)

const (
	ActionPaneClean     ActionPane = "clean"
	ActionPaneUninstall ActionPane = "uninstall"
	ActionPaneOptimize  ActionPane = "optimize"
)

const (
	ProjectsModeInventory ProjectsMode = "inventory"
	ProjectsModeArtifacts ProjectsMode = "artifacts"
)

type NavItem struct {
	Screen      Screen
	Title       string
	Subtitle    string
	Icon        string
	Description string
}

type CleanTarget struct {
	ID            string
	Category      string
	Title         string
	Description   string
	Paths         []string
	RequiresAdmin bool
	Selected      bool
	Size          int64
	Items         int
	Status        string
	LastError     string
}

type InstalledApp struct {
	ID                   string
	Name                 string
	Version              string
	Publisher            string
	Source               string
	InstallDate          string
	Size                 int64
	Selected             bool
	WingetID             string
	UninstallString      string
	QuietUninstallString string
	LocalPath            string
	Method               string
}

type OptimizeTask struct {
	ID          string
	Category    string
	Title       string
	Description string
	AdminOnly   bool
	Selected    bool
	Status      string
	LastResult  string
	Duration    time.Duration
	Command     string
}

type DirectoryUsage struct {
	Name    string
	Path    string
	Size    int64
	Percent float64
}

type Artifact struct {
	ID        string
	ProjectID string
	Label     string
	Path      string
	Type      string
	Size      int64
	AgeDays   int
	Selected  bool
}

type Project struct {
	ID                 string
	Name               string
	Root               string
	Ecosystems         []string
	LastScan           time.Time
	TotalArtifactBytes int64
	ArtifactCount      int
	Analyzer           []DirectoryUsage
	Artifacts          []Artifact
}

type SettingItem struct {
	ID          string
	Title       string
	Description string
	Value       string
}

type HelpDoc struct {
	ID      string
	Title   string
	Path    string
	Content string
}

type CommandAction struct {
	ID          string
	Title       string
	Description string
	Screen      Screen
	Keywords    []string
}

type PendingAction struct {
	Kind   string
	IDs    []string
	Target string
}

type ModalState struct {
	Visible      bool
	Kind         string
	Title        string
	Body         string
	Placeholder  string
	Value        string
	ConfirmLabel string
	CancelLabel  string
	Pending      PendingAction
}

type PaletteState struct {
	Visible bool
	Query   string
}

type ConfigState struct {
	Theme                  string
	ScanPaths              []string
	PurgeDepth             int
	RefreshIntervalSeconds int
	WingetEnabled          bool
	CheckUpdates           bool
}

type Store struct {
	Screen           Screen
	Focus            FocusArea
	StatusText       string
	Busy             bool
	BusyText         string
	LastError        string
	Navigation       []NavItem
	Config           ConfigState
	Runtime          RuntimeSnapshot
	ActiveActionPane ActionPane
	ProjectMode      ProjectsMode
	SelectedProject  string
	CleanTargets     []CleanTarget
	InstalledApps    []InstalledApp
	OptimizeTasks    []OptimizeTask
	Projects         []Project
	Logs             []LogEntry
	LogQuery         string
	LogPaused        bool
	HelpDocs         []HelpDoc
	ActiveHelpDoc    string
	Palette          PaletteState
	Modal            ModalState
}

func DefaultNavigation() []NavItem {
	return []NavItem{
		{Screen: ScreenDashboard, Title: "Dashboard", Subtitle: "Live status", Icon: "◆", Description: "System overview, health, runtime stats"},
		{Screen: ScreenProjects, Title: "Projects", Subtitle: "Analyze + purge", Icon: "◫", Description: "Project inventory, analyzer, artifact purge"},
		{Screen: ScreenActions, Title: "Actions", Subtitle: "Clean, uninstall, optimize", Icon: "▶", Description: "Interactive maintenance workflows"},
		{Screen: ScreenLogs, Title: "Logs", Subtitle: "Runtime events", Icon: "⋯", Description: "Searchable task logs"},
		{Screen: ScreenSettings, Title: "Settings", Subtitle: "Config", Icon: "⚙", Description: "Paths, refresh, theme, integration"},
		{Screen: ScreenHelp, Title: "Help", Subtitle: "Docs", Icon: "?", Description: "Keybindings, architecture, developer guide"},
	}
}

func NewStore(config ConfigState) Store {
	store := Store{
		Screen:           ScreenDashboard,
		Focus:            FocusSidebar,
		StatusText:       "Ready",
		Navigation:       DefaultNavigation(),
		Config:           config,
		ActiveActionPane: ActionPaneClean,
		ProjectMode:      ProjectsModeInventory,
	}
	return store
}

func (s Store) SelectedProjectData() *Project {
	for index := range s.Projects {
		if s.Projects[index].ID == s.SelectedProject {
			return &s.Projects[index]
		}
	}
	if len(s.Projects) == 0 {
		return nil
	}
	return &s.Projects[0]
}

func (s Store) SelectedHelpDoc() *HelpDoc {
	for index := range s.HelpDocs {
		if s.HelpDocs[index].ID == s.ActiveHelpDoc {
			return &s.HelpDocs[index]
		}
	}
	if len(s.HelpDocs) == 0 {
		return nil
	}
	return &s.HelpDocs[0]
}
