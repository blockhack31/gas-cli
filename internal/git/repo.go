package git

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrNotGitRepository is returned when .git/config cannot be found.
	ErrNotGitRepository = errors.New("not a git repository")
	// ErrGitConfigMissing is returned when .git/config does not exist.
	ErrGitConfigMissing = errors.New(".git/config not found")
	// ErrOriginMissing is returned when origin is not configured in .git/config.
	ErrOriginMissing = errors.New("origin remote not found in .git/config")
	// ErrNotGitHubRepo is returned when the origin URL is not a GitHub repository.
	ErrNotGitHubRepo = errors.New("not a GitHub repository")
)

// RepoInfo holds parsed repo information.
type RepoInfo struct {
	URL      string
	RepoName string
	RootDir  string
}

// GetRemoteOriginURL reads .git/config and returns the origin remote URL.
// Searches from cwd upward for .git directory.
func GetRemoteOriginURL() (*RepoInfo, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up to find .git
	for {
		gitDir := filepath.Join(dir, ".git")
		configPath := filepath.Join(gitDir, "config")
		if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
			url, err := readRemoteOriginURL(configPath)
			if err != nil {
				return nil, err
			}
			repoName := filepath.Base(dir)
			return &RepoInfo{URL: url, RepoName: repoName, RootDir: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("%w: .git/config not found in current directory or parents", ErrNotGitRepository)
		}
		dir = parent
	}
}

// GetGitHubRepoInfo reads .git/config and ensures origin points to a GitHub repository.
func GetGitHubRepoInfo() (*RepoInfo, error) {
	info, err := GetRemoteOriginURL()
	if err != nil {
		return nil, err
	}

	if _, err := ParseGitHubUsername(info.URL); err != nil {
		return nil, fmt.Errorf("%w: origin points to %s", ErrNotGitHubRepo, info.URL)
	}

	return info, nil
}

func readRemoteOriginURL(configPath string) (string, error) {
	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrGitConfigMissing, configPath)
		}
		return "", fmt.Errorf("failed to open git config: %w", err)
	}
	defer f.Close()

	var inOrigin bool
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[remote \"origin\"]") {
			inOrigin = true
			continue
		}
		if inOrigin {
			if strings.HasPrefix(line, "[") {
				break
			}
			if strings.HasPrefix(line, "url = ") {
				return strings.TrimSpace(line[6:]), nil
			}
		}
	}
	return "", fmt.Errorf("%w: no [remote \"origin\"] url in .git/config", ErrOriginMissing)
}

// ParseGitHubUsername extracts the GitHub username (owner) from a repo URL.
// Supports: https://github.com/owner/repo, git@github.com:owner/repo.git
func ParseGitHubUsername(repoURL string) (string, error) {
	// git@github.com:owner/repo.git
	gitMatch := regexp.MustCompile(`github\.com:([^/]+)/`).FindStringSubmatch(repoURL)
	if len(gitMatch) >= 2 {
		return gitMatch[1], nil
	}
	// https://github.com/owner/repo or https://user:pass@github.com/owner/repo
	httpsMatch := regexp.MustCompile(`github\.com[/:]([^/]+)/`).FindStringSubmatch(repoURL)
	if len(httpsMatch) >= 2 {
		return httpsMatch[1], nil
	}
	return "", fmt.Errorf("%w: could not parse GitHub owner from URL: %s", ErrNotGitHubRepo, repoURL)
}
