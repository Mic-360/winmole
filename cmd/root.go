package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mic-360/wimo/internal/services"
	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/tui"
)

var (
	debugFlag bool
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "winmole",
		Short:   "Modern Windows maintenance TUI",
		Long:    "Winmole is a Windows-first terminal UI for cleanup, uninstall, optimization, project analysis and artifact purge workflows.",
		Version: "2.0.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithScreen(state.ScreenDashboard)
		},
	}
	root.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Enable verbose runtime logging")
	root.AddCommand(newTUICmd())
	root.AddCommand(screenCmd("dashboard", state.ScreenDashboard, "Open the live status dashboard"))
	root.AddCommand(screenCmd("projects", state.ScreenProjects, "Open project analyzer and purge"))
	root.AddCommand(screenCmd("actions", state.ScreenActions, "Open cleanup, uninstall and optimize workflows"))
	root.AddCommand(screenCmd("logs", state.ScreenLogs, "Open runtime log viewer"))
	root.AddCommand(screenCmd("settings", state.ScreenSettings, "Open settings editor"))
	root.AddCommand(screenCmd("help", state.ScreenHelp, "Open built-in documentation"))
	root.AddCommand(screenCmd("status", state.ScreenDashboard, "Alias for dashboard"))
	root.AddCommand(screenCmd("analyze", state.ScreenProjects, "Alias for project analyzer"))
	root.AddCommand(screenCmd("purge", state.ScreenProjects, "Alias for project purge"))
	root.AddCommand(screenCmd("clean", state.ScreenActions, "Alias for actions"))
	root.AddCommand(screenCmd("uninstall", state.ScreenActions, "Alias for actions"))
	root.AddCommand(screenCmd("optimize", state.ScreenActions, "Alias for actions"))
	return root
}

func screenCmd(name string, screen state.Screen, description string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: description,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithScreen(screen)
		},
	}
}

func runWithScreen(screen state.Screen) error {
	if screen == "" {
		screen = state.ScreenDashboard
	}
	container, err := services.NewContainer()
	if err != nil {
		return err
	}
	defer container.Close()
	if debugFlag {
		container.Logger.Info("cli", "debug mode enabled")
	}
	return tui.Run(container, screen)
}
