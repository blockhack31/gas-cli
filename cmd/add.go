package cmd

import (
	"fmt"
	"strings"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/git"
	"github.com/calghar/gas-cli/internal/ssh"
	"github.com/spf13/cobra"
)

var patFlag string

var addCmd = &cobra.Command{
	Use:   "add <name> [git-name] [email|pat] [gpg-key]",
	Short: "Add a new profile",
	Long: `Add a new GitHub profile. Name is required; git-name, email, and gpg-key are optional.

Examples:
  gascli add work                           # Name only
  gascli add work "John Doe"                # Name + git-name
  gascli add work "John Doe" ghp_xxxx       # Name + git-name + PAT (always supported)
  gascli add work "John Doe" john@company.com ABC123DEF456
  gascli add work john@company.com         # Name + email (2 args, auto-detected)
  gascli add work --pat ghp_xxxx             # Name + PAT via flag`,
	Args: cobra.RangeArgs(1, 4),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&patFlag, "pat", "", "GitHub Personal Access Token for HTTPS authentication")
}

func runAdd(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	gitName := ""
	email := ""
	gpgKey := ""

	if len(args) >= 2 {
		gitName = args[1]
	}
	if len(args) >= 3 {
		email = args[2]
	}
	if len(args) >= 4 {
		gpgKey = args[3]
	}

	// When only 2 args: if second looks like email (@), treat as email not git-name
	if len(args) == 2 && containsAt(args[1]) {
		gitName = ""
		email = args[1]
	}

	// When 3+ args: if third looks like PAT (ghp_/gho_/github_pat_), treat as PAT not email
	if len(args) >= 3 && looksLikePAT(args[2]) {
		email = ""
		if patFlag == "" {
			patFlag = args[2]
		}
	}

	// Initialize configuration manager
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Load existing configuration
	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if Git is installed
	if err := git.CheckGitInstalled(); err != nil {
		return fmt.Errorf("git is required but not found: %w", err)
	}

	// Create or update profile
	profile := &config.Profile{
		Name:         profileName,
		Emails:       []string{},
		PrimaryEmail: email,
		GitName:      gitName,
		GPGKey:       gpgKey,
	}
	if email != "" {
		profile.Emails = []string{email}
	}
	if patFlag != "" {
		profile.PAT = patFlag
	}

	// Merge with existing profile if updating
	if existing, err := cfg.GetProfile(profileName); err == nil {
		if profile.PAT == "" && existing.PAT != "" {
			profile.PAT = existing.PAT
		}
		if profile.GitName == "" && existing.GitName != "" {
			profile.GitName = existing.GitName
		}
		if profile.GPGKey == "" && existing.GPGKey != "" {
			profile.GPGKey = existing.GPGKey
		}
		if profile.PrimaryEmail == "" && existing.PrimaryEmail != "" {
			profile.PrimaryEmail = existing.PrimaryEmail
			profile.Emails = existing.Emails
		} else if len(profile.Emails) == 1 && len(existing.Emails) > 1 {
			profile.Emails = existing.Emails
		}
	}

	// Validate and add profile
	if err := cfg.AddProfile(profile); err != nil {
		return fmt.Errorf("failed to add profile: %w", err)
	}

	// Save configuration
	if err := configMgr.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Setup SSH config entry
	sshMgr, err := ssh.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize SSH manager: %w", err)
	}

	if err := sshMgr.EnsureProfileEntry(profile); err != nil {
		fmt.Printf("Warning: Failed to setup SSH config: %v\n", err)
	} else {
		hostAlias := ssh.GetHostAlias(profileName)
		fmt.Printf("✓ SSH config entry created\n")
		fmt.Printf("  Use this host in git URLs: git@%s:user/repo.git\n", hostAlias)
	}

	// Refresh git config for directory rules (e.g. to apply PAT credential setup)
	gitMgr, err := git.NewConfigManager()
	if err == nil {
		if err := gitMgr.SetupAllProfiles(cfg); err != nil {
			fmt.Printf("Warning: Failed to update git config: %v\n", err)
		}
	}

	// Success message (git-name first, then email)
	fmt.Printf("\n✓ Profile '%s' added successfully!\n", profileName)
	if profile.GitName != "" {
		fmt.Printf("  Git name: %s\n", profile.GitName)
	}
	if profile.PrimaryEmail != "" {
		fmt.Printf("  Email: %s\n", profile.PrimaryEmail)
	}
	if gpgKey != "" {
		fmt.Printf("  GPG key: %s\n", gpgKey)
	}
	if profile.PAT != "" {
		fmt.Printf("  PAT: configured (HTTPS auth)\n")
	}

	// Check if SSH key exists
	sshKeyPath := ssh.GetSSHKeyPath(profileName)
	if !ssh.CheckSSHKeyExists(sshKeyPath) {
		fmt.Printf("\n⚠ SSH key not found: %s\n", sshKeyPath)
		emailHint := profile.PrimaryEmail
		if emailHint == "" {
			emailHint = "your@email.com"
		}
		fmt.Printf("  Generate one with: ssh-keygen -t ed25519 -f %s -C \"%s\"\n", sshKeyPath, emailHint)
	}

	// Suggest next steps
	fmt.Println("\nNext steps:")
	if profile.PrimaryEmail == "" {
		fmt.Printf("  1. Add email: gascli add-email %s <email>\n", profileName)
		fmt.Printf("  2. Set up a directory rule: gascli auto ~/projects/work %s\n", profileName)
		fmt.Printf("  3. Or switch manually: gascli switch %s\n", profileName)
	} else {
		fmt.Printf("  1. Set up a directory rule: gascli auto ~/projects/work %s\n", profileName)
		fmt.Printf("  2. Or switch manually: gascli switch %s\n", profileName)
	}

	return nil
}

func containsAt(s string) bool {
	return strings.Contains(s, "@")
}

func looksLikePAT(s string) bool {
	return strings.HasPrefix(s, "ghp_") || strings.HasPrefix(s, "gho_") || strings.HasPrefix(s, "github_pat_")
}
