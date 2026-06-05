package tui

import (
	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/ghsetup"
	"github.com/calghar/gas-cli/internal/git"
)

type repoSetupOutcome int

const (
	repoOutcomeMessage repoSetupOutcome = iota
	repoOutcomePicker
	repoOutcomeConfigured
)

type repoSetupResult struct {
	outcome     repoSetupOutcome
	title       string
	message     string
	ghUsername  string
	profileName string
	result      *ghsetup.Result
}

func executeRepoSetup(cfg *config.Config, profileName, preferred string) repoSetupResult {
	if profileName == "" {
		repoInfo, err := git.GetGitHubRepoInfo()
		if err != nil {
			title, message := ghsetup.UserFacingError(err)
			return repoSetupResult{outcome: repoOutcomeMessage, title: title, message: message}
		}

		ghUsername, err := git.ParseGitHubUsername(repoInfo.URL)
		if err != nil {
			title, message := ghsetup.UserFacingError(err)
			return repoSetupResult{outcome: repoOutcomeMessage, title: title, message: message}
		}

		resolved, _ := ghsetup.ResolveProfile(cfg, ghUsername, preferred)
		if resolved == "" {
			return repoSetupResult{outcome: repoOutcomePicker, ghUsername: ghUsername}
		}
		profileName = resolved
	}

	result, err := ghsetup.ConfigureRepoWithProfile(cfg, profileName)
	if err != nil {
		title, message := ghsetup.UserFacingError(err)
		return repoSetupResult{outcome: repoOutcomeMessage, title: title, message: message}
	}

	return repoSetupResult{
		outcome:     repoOutcomeConfigured,
		profileName: profileName,
		result:      result,
	}
}
