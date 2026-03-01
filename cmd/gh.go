package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/git"
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
	// Get remote URL and repo info from .git/config
	repoInfo, err := git.GetRemoteOriginURL()
	if err != nil {
		return err
	}

	// Parse GitHub username from URL
	ghUsername, err := git.ParseGitHubUsername(repoInfo.URL)
	if err != nil {
		return err
	}

	// Load config
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}
	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Find matching profile
	var profileName string
	var profile *config.Profile

	// 1. Check default/current profile first
	if cfg.CurrentProfile != "" {
		p, err := cfg.GetProfile(cfg.CurrentProfile)
		if err == nil && p.GitName == ghUsername {
			profileName = cfg.CurrentProfile
			profile = p
		}
	}

	// 2. If no match, find by git-name or profile name
	if profile == nil {
		name, p := cfg.GetProfileByGitNameOrName(ghUsername)
		if p != nil {
			profileName = name
			profile = p
		}
	}

	// 3. If no match (owner/org from URL doesn't match any profile git-name), list and ask
	if profile == nil {
		profileName, profile = promptSelectProfile(cfg, ghUsername)
		if profile == nil {
			return fmt.Errorf("no profile selected")
		}
	}

	// Profile must have PAT
	if profile.PAT == "" {
		return fmt.Errorf("profile '%s' has no PAT; set with: gascli pat set %s <token>", profileName, profileName)
	}

	// Profile must have email for commits
	email := profile.PrimaryEmail
	if email == "" {
		return fmt.Errorf("profile '%s' has no email; add with: gascli add-email %s <email>", profileName, profileName)
	}

	// Use profile's GitName for user.name (GitHub username for commits); fallback to profile name or URL owner
	userName := profile.GitName
	if userName == "" {
		userName = profileName
	}
	if userName == "" {
		userName = ghUsername
	}

	// Run git commands from repo root
	remoteURL := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", ghUsername, profile.PAT, ghUsername, repoInfo.RepoName)

	if err := runGitConfig(repoInfo.RootDir, "user.name", userName); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}
	if err := runGitConfig(repoInfo.RootDir, "user.email", email); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}
	if err := runGitRemoteSetURL(repoInfo.RootDir, "origin", remoteURL); err != nil {
		return fmt.Errorf("failed to set remote URL: %w", err)
	}

	fmt.Println("Verifying remote access...")
	if err := runGitLsRemote(repoInfo.RootDir); err != nil {
		return fmt.Errorf("verification failed: %w\nIf the selected profile cannot access this repo, try a different profile.", err)
	}
	runGitRemoteV(repoInfo.RootDir)
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

func runGitConfig(dir, key, value string) error {
	c := exec.Command("git", "config", key, value)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

func runGitRemoteSetURL(dir, remote, url string) error {
	c := exec.Command("git", "remote", "set-url", remote, url)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

func runGitLsRemote(dir string) error {
	c := exec.Command("git", "ls-remote", "origin")
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

func runGitRemoteV(dir string) {
	c := exec.Command("git", "remote", "-v")
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
}
