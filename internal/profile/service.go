package profile

import (
	"errors"
	"fmt"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/git"
	"github.com/calghar/gas-cli/internal/ssh"
)

// Upsert merges incoming fields with an existing profile when present, then validates and stores it.
func Upsert(cfg *config.Config, profile *config.Profile) error {
	if existing, err := cfg.GetProfile(profile.Name); err == nil {
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

	if profile.PrimaryEmail != "" {
		profile.Emails = []string{profile.PrimaryEmail}
	}

	return cfg.AddProfile(profile)
}

// Save persists configuration to disk.
func Save(mgr *config.ConfigManager, cfg *config.Config) error {
	if err := mgr.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
}

// AfterSave applies SSH and Git side-effects after a profile is saved.
func AfterSave(cfg *config.Config, prof *config.Profile) error {
	var errs []error

	sshMgr, err := ssh.NewConfigManager()
	if err != nil {
		errs = append(errs, fmt.Errorf("ssh config manager: %w", err))
	} else if err := sshMgr.EnsureProfileEntry(prof); err != nil {
		errs = append(errs, fmt.Errorf("ssh config: %w", err))
	}

	gitMgr, err := git.NewConfigManager()
	if err != nil {
		errs = append(errs, fmt.Errorf("git config manager: %w", err))
	} else {
		if err := gitMgr.SetupProfile(prof, ""); err != nil {
			errs = append(errs, fmt.Errorf("git profile config: %w", err))
		}
		if err := gitMgr.SetupAllProfiles(cfg); err != nil {
			errs = append(errs, fmt.Errorf("git directory rules: %w", err))
		}
	}

	return errors.Join(errs...)
}

// AfterRemove cleans up SSH and Git artifacts for a removed profile.
func AfterRemove(profileName string) error {
	var errs []error

	gitMgr, err := git.NewConfigManager()
	if err != nil {
		errs = append(errs, fmt.Errorf("git config manager: %w", err))
	} else if err := gitMgr.RemoveProfileConfig(profileName); err != nil {
		errs = append(errs, fmt.Errorf("git config: %w", err))
	}

	sshMgr, err := ssh.NewConfigManager()
	if err != nil {
		errs = append(errs, fmt.Errorf("ssh config manager: %w", err))
	} else if err := sshMgr.RemoveProfileEntry(profileName); err != nil {
		errs = append(errs, fmt.Errorf("ssh config: %w", err))
	}

	return errors.Join(errs...)
}
