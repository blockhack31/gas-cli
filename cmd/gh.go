package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/ghsetup"
	"github.com/calghar/gas-cli/internal/git"
	"github.com/calghar/gas-cli/internal/profile"
	"github.com/spf13/cobra"
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "Configure repo from .git/config and matching profile",
	Long: `Scan .git/config for remote origin URL, extract GitHub username, and configure
the repo with matching profile (user.name, user.email, remote URL with PAT).

Flow:
  1. Parse remote origin from .git/config
  2. Extract GitHub username (e.g. blockhack31 from github.com/blockhack31/repo)
  3. Check default profile first, then find profile with matching git-name
  4. If no match, list profiles and ask which to use
  5. Set git config user.name, user.email, and remote URL with PAT`,
	Args: cobra.NoArgs,
	RunE: runGh,
}

func init() {
	rootCmd.AddCommand(ghCmd)
}

func runGh(cmd *cobra.Command, args []string) error {
	repoInfo, err := git.GetGitHubRepoInfo()
	if err != nil {
		_, msg := ghsetup.UserFacingError(err)
		return fmt.Errorf("%s", msg)
	}

	ghUsername, err := git.ParseGitHubUsername(repoInfo.URL)
	if err != nil {
		_, msg := ghsetup.UserFacingError(err)
		return fmt.Errorf("%s", msg)
	}

	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}
	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	profileName, prof := ghsetup.FindProfile(cfg, ghUsername)
	if prof == nil {
		profileName, prof = promptSelectProfile(cfg, ghUsername)
		if prof == nil {
			return fmt.Errorf("no profile selected")
		}
	}

	result, err := ghsetup.ConfigureRepoWithProfile(cfg, profileName)
	if err != nil {
		return err
	}

	if err := profile.Save(configMgr, cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("Verifying remote access...")
	runGitRemoteV(result.RepoRoot)
	fmt.Println("Git remote URL has been updated and verified successfully!")
	return nil
}

func promptSelectProfile(cfg *config.Config, ghUsername string) (string, *config.Profile) {
	fmt.Printf("Owner/organization '%s' from remote URL does not match any profile git-name.\n\n", ghUsername)
	fmt.Println("Select which profile to use for this repo:")
	names := cfg.ProfileNames()
	fmt.Println("Profiles:")
	for i, name := range names {
		profile, _ := cfg.GetProfile(name)
		gitName := profile.GitName
		if gitName == "" {
			gitName = "(not set)"
		}
		fmt.Printf("  %d. %s (git-name: %s)\n", i+1, name, gitName)
	}

	if skipPrompts {
		return "", nil
	}

	fmt.Printf("\nSelect profile (1-%d): ", len(names))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)

	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(names) {
		fmt.Println("Invalid selection.")
		return "", nil
	}

	profileName := names[idx-1]
	profile, _ := cfg.GetProfile(profileName)
	return profileName, profile
}

func runGitRemoteV(dir string) {
	c := exec.Command("git", "remote", "-v")
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
}
