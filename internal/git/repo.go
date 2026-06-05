package git

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
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

// LocalRepoConfig holds identity settings read from .git/config in the current directory.
type LocalRepoConfig struct {
	UserName   string
	UserEmail  string
	OriginURL  string
	ConfigPath string
}

// HasGitInCwd reports whether .git exists in the current working directory only.
func HasGitInCwd() bool {
	dir, err := os.Getwd()
	if err != nil {
		return false
	}
	_, err = gitConfigPath(dir)
	return err == nil
}

// GetRemoteOriginURL reads .git/config from the current directory only.
func GetRemoteOriginURL() (*RepoInfo, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath, err := gitConfigPath(dir)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(configPath)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrGitConfigMissing, configPath)
	}

	url, err := readRemoteOriginURL(configPath)
	if err != nil {
		return nil, err
	}

	return &RepoInfo{
		URL:      url,
		RepoName: filepath.Base(dir),
		RootDir:  dir,
	}, nil
}

// gitConfigPath resolves .git/config in dir without searching parent directories.
func gitConfigPath(dir string) (string, error) {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: no .git directory in current path", ErrNotGitRepository)
		}
		return "", fmt.Errorf("failed to check .git: %w", err)
	}

	if info.IsDir() {
		return filepath.Join(gitPath, "config"), nil
	}

	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("%w: cannot read .git file: %w", ErrNotGitRepository, err)
	}

	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("%w: invalid .git file in current path", ErrNotGitRepository)
	}

	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir: "))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}

	return filepath.Join(gitDir, "config"), nil
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

// ReadLocalRepoConfig reads user.name, user.email, and origin URL from .git/config in cwd.
func ReadLocalRepoConfig() (*LocalRepoConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath, err := gitConfigPath(dir)
	if err != nil {
		return nil, err
	}

	cfg := &LocalRepoConfig{ConfigPath: configPath}

	userName, userEmail, originURL, err := parseLocalGitConfig(configPath)
	if err != nil {
		return nil, err
	}

	cfg.UserName = userName
	cfg.UserEmail = userEmail
	cfg.OriginURL = SanitizeRemoteURL(originURL)
	return cfg, nil
}

func parseLocalGitConfig(configPath string) (userName, userEmail, originURL string, err error) {
	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", "", fmt.Errorf("%w: %s", ErrGitConfigMissing, configPath)
		}
		return "", "", "", fmt.Errorf("failed to open git config: %w", err)
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line)
			continue
		}

		key, value, ok := splitGitConfigLine(line)
		if !ok {
			continue
		}

		switch section {
		case "[user]":
			switch key {
			case "name":
				userName = value
			case "email":
				userEmail = value
			}
		case "[remote \"origin\"]":
			if key == "url" {
				originURL = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", "", fmt.Errorf("failed to read git config: %w", err)
	}

	return userName, userEmail, originURL, nil
}

func splitGitConfigLine(line string) (key, value string, ok bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

// SanitizeRemoteURL redacts credentials from a git remote URL for display.
func SanitizeRemoteURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return redactEmbeddedCredentials(raw)
	}

	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.UserPassword(username, "***")
		} else {
			parsed.User = url.User("***")
		}
	}

	return parsed.String()
}

func redactEmbeddedCredentials(raw string) string {
	re := regexp.MustCompile(`://[^@/]+@`)
	return re.ReplaceAllString(raw, "://***@")
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
