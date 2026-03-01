# Developer Guide

## Agent Skill

gas-cli includes a Cursor agent skill at `.cursor/skills/gas-cli/` that provides:

- Architecture overview and key paths
- Command reference
- Conventions for adding features
- Links to detailed docs

The skill is automatically available when working in this repository. Use it when contributing, adding features, or answering questions about gas-cli.

## Project Structure

```
gas-cli/
├── cmd/              # Cobra commands
│   ├── add.go        # Add profile (--pat flag)
│   ├── auto.go       # Directory rules (includeIf)
│   ├── list.go       # list, ls, current
│   ├── pat.go        # pat set, pat clear
│   ├── switch.go     # Manual switch
│   └── ...
├── internal/
│   ├── config/       # Profile, Config, ConfigManager
│   ├── git/          # Git config, credential helper
│   ├── ssh/          # SSH config entries
│   └── platform/     # Darwin, Linux, Windows keychain
├── docs/             # User and developer documentation
└── .cursor/skills/   # Agent skills
```

## Key Implementation Details

- **Config**: `~/.gascli/config.json` (JSON, 0600)
- **PAT**: Writes `~/.gascli/credentials-{profile}` in Git credential store format
- **includeIf**: Profile gitconfig at `~/.gitconfig-{profile}`, loaded by path
- **SSH**: Host alias `github.com-{profile}`, key at `~/.ssh/id_{profile}`

## Documentation Index

| Doc | Purpose |
|-----|---------|
| [commands.md](commands.md) | Full command reference |
| [user-guide.md](user-guide.md) | Workflows and usage |
| [security.md](security.md) | SSH, GPG, storage |
| [features.md](features.md) | Feature overview |
| [use-cases.md](use-cases.md) | Example scenarios |
| [installation.md](installation.md) | Install instructions |
