package cmd

import (
	"github.com/calghar/gas-cli/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open interactive identity manager",
	Long: `Launch a terminal UI for managing GitHub identities.

Use the console to add, edit, and delete profiles, set PAT tokens,
and configure the current git repository for HTTPS access.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
