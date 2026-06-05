package cmd

import (
	"fmt"
	"os"

	"github.com/calghar/gas-cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	autoSSH        bool
	skipPrompts    bool
	version        = "2.0.0"
)

var rootCmd = &cobra.Command{
	Use:   "gascli",
	Short: "GitHub Account Switcher - Manage multiple Git identities",
	Long: `gascli is a modern CLI tool for managing multiple GitHub accounts.

Running gascli with no subcommand opens the interactive identity manager.
Subcommands are available for scripting and automation.`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&autoSSH, "auto-ssh", "s", false, "Automatically add SSH key to agent/keychain")
	rootCmd.PersistentFlags().BoolVarP(&skipPrompts, "yes", "y", false, "Skip confirmation prompts")
}
