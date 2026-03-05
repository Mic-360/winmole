package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mic-360/wimo/internal/services"
	"github.com/mic-360/wimo/internal/state"
)

func Run(container *services.Container, start state.Screen) error {
	program := tea.NewProgram(NewModel(container, start), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func loadRuntimeCmd(runtime *services.RuntimeService) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := runtime.Snapshot(context.Background())
		return runtimeLoadedMsg{snapshot: snapshot, err: err}
	}
}

func loadCleanTargetsCmd(cleaner *services.CleanerService) tea.Cmd {
	return func() tea.Msg {
		targets, err := cleaner.Scan(context.Background(), cleaner.Targets())
		return cleanTargetsLoadedMsg{targets: targets, err: err}
	}
}

func loadAppsCmd(service *services.UninstallService, wingetEnabled bool) tea.Cmd {
	return func() tea.Msg {
		apps, err := service.Inventory(context.Background(), wingetEnabled)
		return appsLoadedMsg{apps: apps, err: err}
	}
}

func loadOptimizeCmd(service *services.OptimizerService) tea.Cmd {
	return func() tea.Msg {
		return optimizeLoadedMsg{tasks: service.Tasks()}
	}
}

func loadProjectsCmd(service *services.PurgeService, config state.ConfigState) tea.Cmd {
	return func() tea.Msg {
		projects, err := service.ScanProjects(context.Background(), config.ScanPaths, config.PurgeDepth)
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func loadDocsCmd() tea.Cmd {
	return func() tea.Msg {
		docs, err := services.LoadHelpDocs()
		return docsLoadedMsg{docs: docs, err: err}
	}
}

func runCleanCmd(service *services.CleanerService, targets []state.CleanTarget) tea.Cmd {
	return func() tea.Msg {
		report, updated, err := service.Execute(context.Background(), targets, false)
		return cleanRunMsg{report: report, targets: updated, err: err}
	}
}

func runUninstallCmd(service *services.UninstallService, apps []state.InstalledApp) tea.Cmd {
	return func() tea.Msg {
		report, err := service.Remove(context.Background(), apps)
		return uninstallRunMsg{report: report, err: err}
	}
}

func runOptimizeCmd(service *services.OptimizerService, tasks []state.OptimizeTask) tea.Cmd {
	return func() tea.Msg {
		report, updated, err := service.Run(context.Background(), tasks)
		return optimizeRunMsg{report: report, tasks: updated, err: err}
	}
}

func runPurgeCmd(service *services.PurgeService, artifacts []state.Artifact) tea.Cmd {
	return func() tea.Msg {
		report, err := service.Purge(context.Background(), artifacts)
		return purgeRunMsg{report: report, err: err}
	}
}

func saveConfigCmd(service *services.ConfigService, config state.ConfigState) tea.Cmd {
	return func() tea.Msg {
		err := service.Save(config)
		return configSavedMsg{config: config, err: err}
	}
}
