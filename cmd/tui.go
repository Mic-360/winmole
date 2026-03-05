package cmd

import (
	"github.com/mic-360/wimo/internal/state"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the full-screen terminal UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithScreen(state.ScreenDashboard)
		},
	}
}
