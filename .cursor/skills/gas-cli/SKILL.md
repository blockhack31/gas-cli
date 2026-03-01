---
name: gas-cli
description: Develop and extend gas-cli (GitHub Account Switcher CLI). Use when contributing to gas-cli, adding features, fixing bugs, or answering questions about multi-profile Git/SSH/GPG management, includeIf automation, or PAT authentication.
---

# gas-cli Agent Skill

## What gas-cli Does

gas-cli manages multiple GitHub identities with automatic directory-based switching via Git's `includeIf`, SSH key isolation, GPG signing, and PAT for HTTPS.

## Key Paths

| Path | Purpose |
|------|---------|
| `~/.gascli/config.json` | Profiles, directory rules, PAT (0600) |
| `~/.gascli/credentials-{profile}` | Git credential store for PAT (0600) |
| `~/.gitconfig-{profile}` | Per-profile Git config (includeIf target) |
| `~/.ssh/config` | Host aliases `github.com-{profile}` |
| `~/.ssh/id_{profile}` | SSH keys (user-created) |

## Architecture

```
cmd/          # Cobra commands (add, list, switch, auto, pat, etc.)
internal/
  config/     # Profile struct, ConfigManager, Load/Save
  git/        # SetupProfile, credential helper, includeIf
  ssh/        # SSH config entries, GetSSHKeyPath
  platform/   # Keychain: Darwin, Linux, Windows
```

**Profile struct** (`internal/config/profile.go`): `Name`, `Emails`, `PrimaryEmail`, `GitName`, `GPGKey`, `SSHKeyPath`, `PAT`

## Command Reference

```bash
# Profiles (name required; git-name, email|pat optional)
gascli add <name> [git-name] [email|pat] [gpg-key] [--pat TOKEN]
# name + git-name + PAT: gascli add work "John Doe" ghp_xxx
gascli list | gascli ls
gascli remove <name>

# Directory-based (includeIf)
gascli auto <directory> <profile>
gascli auto-list
gascli auto-remove <directory>

# Manual switch
gascli switch <profile> [email]
gascli --auto-ssh switch <profile>

# PAT
gascli pat set <profile> <token>
gascli pat clear <profile>

# Emails
gascli add-email <profile> <email>
gascli remove-email <profile> <email>
gascli list-emails <profile>

# Backup
gascli export [file]
gascli import <file>
```

## Adding Features

1. **New profile field**: Add to `Profile` in `internal/config/profile.go`, update `Validate()` if needed.
2. **New command**: Create `cmd/foo.go`, register in `init()` with `rootCmd.AddCommand()`.
3. **Git integration**: Extend `internal/git/config.go` `SetupProfile()` for profile-specific gitconfig.
4. **PAT/credential**: PAT writes `~/.gascli/credentials-{profile}` and `[credential "https://github.com"]` in profile gitconfig.

## Conventions

- SSH key path: `~/.ssh/id_{profileName}`
- Host alias: `github.com-{profileName}`
- Profile gitconfig: `~/.gitconfig-{profileName}`
- Build output: `./bin/gascli`; install: `$GOPATH/bin` or `$GOBIN`

## Additional Resources

- [docs/commands.md](../../../docs/commands.md) - Full command reference
- [docs/security.md](../../../docs/security.md) - SSH, GPG, storage
- [docs/user-guide.md](../../../docs/user-guide.md) - Workflows
