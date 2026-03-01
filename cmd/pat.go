package cmd

import (
	"fmt"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/git"
	"github.com/spf13/cobra"
)

var patCmd = &cobra.Command{
	Use:   "pat",
	Short: "Manage Personal Access Token for a profile",
	Long:  `Set or clear the GitHub Personal Access Token for HTTPS authentication.`,
}

var patSetCmd = &cobra.Command{
	Use:   "set <profile> <token>",
	Short: "Set PAT for a profile",
	Long:  `Set the GitHub Personal Access Token for HTTPS authentication.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runPatSet,
}

var patClearCmd = &cobra.Command{
	Use:   "clear <profile>",
	Short: "Clear PAT for a profile",
	Long:  `Remove the stored PAT from a profile.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPatClear,
}

func init() {
	rootCmd.AddCommand(patCmd)
	patCmd.AddCommand(patSetCmd)
	patCmd.AddCommand(patClearCmd)
}

func runPatSet(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	token := args[1]

	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return err
	}

	profile.PAT = token
	if err := cfg.AddProfile(profile); err != nil {
		return err
	}

	if err := configMgr.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	gitMgr, err := git.NewConfigManager()
	if err == nil {
		if err := gitMgr.SetupAllProfiles(cfg); err != nil {
			fmt.Printf("Warning: Failed to update git config: %v\n", err)
		}
	}

	fmt.Printf("✓ PAT set for profile '%s'\n", profileName)
	return nil
}

func runPatClear(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return err
	}

	profile.PAT = ""
	if err := cfg.AddProfile(profile); err != nil {
		return err
	}

	if err := configMgr.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	gitMgr, err := git.NewConfigManager()
	if err == nil {
		if err := gitMgr.SetupAllProfiles(cfg); err != nil {
			fmt.Printf("Warning: Failed to update git config: %v\n", err)
		}
	}

	fmt.Printf("✓ PAT cleared for profile '%s'\n", profileName)
	return nil
}
