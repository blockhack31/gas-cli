package ghsetup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/git"
)

// Result holds the outcome of configuring a repository.
type Result struct {
	ProfileName string
	UserName    string
	Email       string
	RepoRoot    string
	RepoName    string
}

// FindProfile returns a profile matching the GitHub username from a remote URL.
func FindProfile(cfg *config.Config, ghUsername string) (string, *config.Profile) {
	return findProfile(cfg, ghUsername)
}

// ProfileMatchesOwner reports whether a profile corresponds to a GitHub remote owner.
func ProfileMatchesOwner(prof *config.Profile, profileName, ghUsername string) bool {
	if prof == nil || ghUsername == "" {
		return false
	}
	if prof.GitName == ghUsername {
		return true
	}
	return profileName == ghUsername
}

// ResolveProfile picks the best profile for a GitHub remote owner.
// preferredName is the highlighted profile in the UI, if any.
// Returns empty name when the caller should prompt for selection.
func ResolveProfile(cfg *config.Config, ghUsername, preferredName string) (string, *config.Profile) {
	if preferredName != "" {
		if prof, err := cfg.GetProfile(preferredName); err == nil {
			if ProfileMatchesOwner(prof, preferredName, ghUsername) {
				return preferredName, prof
			}
		}
	}

	if name, prof := findProfile(cfg, ghUsername); prof != nil {
		return name, prof
	}

	return "", nil
}

// UserFacingError returns a title and message suitable for displaying repo setup failures.
func UserFacingError(err error) (title, message string) {
	switch {
	case errors.Is(err, git.ErrNotGitRepository), errors.Is(err, git.ErrGitConfigMissing):
		return "Not a GitHub repository",
			".git/config was not found.\n\nRun gascli from inside a cloned GitHub repository, or cd to a directory that contains .git/config."
	case errors.Is(err, git.ErrOriginMissing):
		return "Not a GitHub repository",
			"No origin remote found in .git/config.\n\nAdd a GitHub remote first:\n  git remote add origin https://github.com/owner/repo.git"
	case errors.Is(err, git.ErrNotGitHubRepo):
		return "Not a GitHub repository",
			err.Error() + "\n\nRepo setup only works with GitHub remotes (github.com)."
	default:
		return "Repo setup failed", err.Error()
	}
}

// ConfigureRepo matches a profile to the current repository and applies HTTPS PAT setup.
func ConfigureRepo(cfg *config.Config) (*Result, error) {
	repoInfo, err := git.GetGitHubRepoInfo()
	if err != nil {
		return nil, err
	}

	ghUsername, err := git.ParseGitHubUsername(repoInfo.URL)
	if err != nil {
		return nil, err
	}

	profileName, prof := findProfile(cfg, ghUsername)
	if prof == nil {
		return nil, fmt.Errorf("no matching profile for GitHub user '%s'", ghUsername)
	}

	return applyProfile(cfg, profileName, prof, repoInfo, ghUsername)
}

// ConfigureRepoWithProfile configures the current repository using the given profile.
func ConfigureRepoWithProfile(cfg *config.Config, profileName string) (*Result, error) {
	prof, err := cfg.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	repoInfo, err := git.GetGitHubRepoInfo()
	if err != nil {
		return nil, err
	}

	ghUsername, err := git.ParseGitHubUsername(repoInfo.URL)
	if err != nil {
		return nil, err
	}

	return applyProfile(cfg, profileName, prof, repoInfo, ghUsername)
}

func findProfile(cfg *config.Config, ghUsername string) (string, *config.Profile) {
	if cfg.CurrentProfile != "" {
		if p, err := cfg.GetProfile(cfg.CurrentProfile); err == nil && p.GitName == ghUsername {
			return cfg.CurrentProfile, p
		}
	}

	if name, p := cfg.GetProfileByGitNameOrName(ghUsername); p != nil {
		return name, p
	}

	return "", nil
}

func applyProfile(cfg *config.Config, profileName string, profile *config.Profile, repoInfo *git.RepoInfo, ghUsername string) (*Result, error) {
	if profile.PAT == "" {
		return nil, fmt.Errorf("profile '%s' has no PAT; set with: gascli pat set %s <token>", profileName, profileName)
	}

	email := profile.PrimaryEmail
	if email == "" {
		return nil, fmt.Errorf("profile '%s' has no email; add with: gascli add-email %s <email>", profileName, profileName)
	}

	userName := profile.GitName
	if userName == "" {
		userName = profileName
	}
	if userName == "" {
		userName = ghUsername
	}

	noreplyEmail, err := getNoreplyEmailViaGh(profile.PAT)
	if err != nil {
		return nil, fmt.Errorf("failed to check email via gh: %w\nEnsure gh is installed and the PAT has user:email scope", err)
	}
	if noreplyEmail != "" {
		email = noreplyEmail
	}

	remoteURL := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", ghUsername, profile.PAT, ghUsername, repoInfo.RepoName)

	if err := runGitConfig(repoInfo.RootDir, "user.name", userName); err != nil {
		return nil, fmt.Errorf("failed to set user.name: %w", err)
	}
	if err := runGitConfig(repoInfo.RootDir, "user.email", email); err != nil {
		return nil, fmt.Errorf("failed to set user.email: %w", err)
	}
	if err := runGitRemoteSetURL(repoInfo.RootDir, "origin", remoteURL); err != nil {
		return nil, fmt.Errorf("failed to set remote URL: %w", err)
	}
	if err := runGitLsRemote(repoInfo.RootDir); err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	cfg.CurrentProfile = profileName

	return &Result{
		ProfileName: profileName,
		UserName:    userName,
		Email:       email,
		RepoRoot:    repoInfo.RootDir,
		RepoName:    repoInfo.RepoName,
	}, nil
}

type ghEmailEntry struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
	Type       string `json:"type"`
}

func getNoreplyEmailViaGh(pat string) (string, error) {
	c := exec.Command("gh", "api", "user/emails")
	c.Env = append(os.Environ(), "GH_TOKEN="+pat)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api user/emails failed: %w\n%s", err, string(out))
	}

	var entries []ghEmailEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return "", fmt.Errorf("failed to parse gh api response: %w", err)
	}

	for _, e := range entries {
		if (e.Type == "noreply" || strings.HasSuffix(e.Email, "@users.noreply.github.com")) && e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
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
