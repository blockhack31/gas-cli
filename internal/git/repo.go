package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
			return nil, fmt.Errorf("not a git repository (or any of the parent directories)")
		}
		dir = parent
	}
}

func readRemoteOriginURL(configPath string) (string, error) {
	f, err := os.Open(configPath)
	if err != nil {
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
	return "", fmt.Errorf("remote origin URL not found in .git/config")
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
	return "", fmt.Errorf("could not parse GitHub username from URL: %s", repoURL)
}
