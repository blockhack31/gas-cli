# GitHub Account Switcher

> Modern CLI tool for managing multiple Git identities with automatic switching

[![Platform Support](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue)](https://github.com/calghar/gas-cli#-requirements) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)

## ✨ Features

- **Git includeIf automation** - Set up once, switches automatically by directory
- **SSH config with IdentitiesOnly** - Proper multi-account SSH key isolation
- **PAT (HTTPS) authentication** - Personal Access Token support for HTTPS remotes
- **Repo setup (`gh`)** - Auto-configure repo from remote URL and matching profile
- **Multi-email support** - Multiple emails per profile for different contexts
- **Auto SSH key management** - Platform-specific keychain integration
- **Profile import/export** - Share configurations across machines
- **GPG commit signing** - Automatic GPG key management per profile
- **Cross-platform** - Single binary for macOS, Linux, and Windows

## 🚀 Quick Start

### Installation

#### Option 1: Download Pre-built Binary (Recommended)

```bash
# macOS (Apple Silicon)
curl -L https://github.com/calghar/gas-cli/releases/latest/download/gascli-darwin-arm64 -o gascli
chmod +x gascli
sudo mv gascli /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/calghar/gas-cli/releases/latest/download/gascli-darwin-amd64 -o gascli
chmod +x gascli
sudo mv gascli /usr/local/bin/

# Linux
curl -L https://github.com/calghar/gas-cli/releases/latest/download/gascli-linux-amd64 -o gascli
chmod +x gascli
sudo mv gascli /usr/local/bin/

# Windows (PowerShell)
# Download from https://github.com/calghar/gas-cli/releases
```

#### Option 2: Build from Source

```bash
git clone https://github.com/calghar/gas-cli.git
cd gas-cli
make build
make install  # Installs to Go bin ($GOPATH/bin or $GOBIN)
```

#### Option 3: Install with Go

```bash
go install github.com/calghar/gas-cli@latest
```

### Basic Usage

```bash
# Add profiles (name required; git-name, email, PAT, gpg-key optional)
gascli add work "John Doe" john.doe@company.com ABC123DEF456
gascli add personal john@gmail.com
gascli add work "John Doe" ghp_xxxx  # With PAT for HTTPS

# Setup automatic directory-based switching (recommended!)
gascli auto ~/projects/work work
gascli auto ~/projects/personal personal
# Now Git automatically uses the right profile in each directory!

# Or switch manually (affects global git config)
gascli switch work
gascli --auto-ssh switch personal  # Also adds SSH key

# Configure repo from remote URL (run from repo root)
gascli gh  # Matches profile by GitHub username, sets user.name/email/remote

# List profiles and directory rules
gascli list
gascli auto-list

# View current configuration
gascli current
```

## 📋 Core Commands

| Command | Description |
|---------|-------------|
| `gascli add <name> [git-name] [email/pat] [gpg-key] [--pat TOKEN]` | Add a new profile |
| `gascli auto <dir> <profile>` | Setup automatic switching (uses Git includeIf) |
| `gascli auto-remove <dir>` | Remove directory rule |
| `gascli switch <name> [email]` | Manually switch profile globally |
| `gascli --auto-ssh switch <name>` | Switch and auto-add SSH key to keychain |
| `gascli gh` | Configure repo from remote URL and matching profile |
| `gascli list` | List all profiles with details |
| `gascli current` | Show current Git configuration |
| `gascli auto-list` | List directory rules |
| `gascli remove <name>` | Remove a profile |
| `gascli pat set <profile> <token>` | Set PAT for HTTPS authentication |
| `gascli pat clear <profile>` | Clear PAT for profile |
| `gascli add-email <profile> <email>` | Add email to profile |
| `gascli remove-email <profile> <email>` | Remove email from profile |
| `gascli export [file]` | Export profiles to JSON |
| `gascli import <file>` | Import profiles from JSON |

### 🎯 Git IncludeIf: The Better Way

Instead of manually switching profiles, set up automatic directory-based switching:

```bash
# One-time setup
gascli auto ~/work work-profile
gascli auto ~/personal personal-profile

# That's it! Git now automatically uses the right profile.
# No need to run gascli switch ever again in these directories.
```

**How it works:** Creates `.gitconfig-{profile}` files and adds `includeIf` directives to your global `.gitconfig`. Git automatically loads the correct configuration based on your repository location.

### Key Features Demo

```bash
# Auto SSH key management with SSH config
$ gascli --auto-ssh switch work
Switched to profile 'work' with email 'john@company.com' and name 'John Doe'

Added SSH config entry for profile 'work'
Use this host in git URLs: git@github.com-work:user/repo.git
Adding SSH key to macOS keychain...
Successfully added SSH key for profile 'work' to keychain
```

### How SSH Multi-Account Support Works

**Why host aliases?** SSH can't distinguish between multiple keys for the same host. Without aliases, SSH tries keys in order and GitHub accepts whichever matches first—potentially authenticating you with the wrong account.

When using `--auto-ssh`, the tool automatically:

1. **Creates SSH config entries** in `~/.ssh/config` with unique host aliases
2. **Adds SSH keys to the agent** (without removing other keys)
3. **Uses `IdentitiesOnly yes`** to ensure GitHub uses the correct key per profile

**Example SSH config entries created:**
```ssh
Host github.com-work
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_work
    IdentitiesOnly yes

Host github.com-personal
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_personal
    IdentitiesOnly yes
```

**Using the host aliases in your git repos:**
```bash
# Clone using profile-specific host
git clone git@github.com-work:company/repo.git

# Update existing repo's remote
git remote set-url origin git@github.com-personal:user/repo.git
```

This approach allows multiple GitHub SSH keys to coexist peacefully, with SSH automatically selecting the correct key based on the host alias you use.

### Repo Setup with `gh`

Run `gascli gh` from a repo root to auto-configure it from the remote origin:

```bash
cd ~/projects/my-repo
gascli gh
```

Scans `.git/config` for the remote URL, extracts the GitHub username, finds a matching profile (by git-name), and sets `user.name`, `user.email`, and the remote URL with PAT for HTTPS. If no profile matches, you'll be prompted to select one.

## 🛠️ Requirements

**Runtime Requirements:**

- Git 2.23.0+ (for includeIf support)
- SSH (for key-based authentication)
- GPG (optional, for commit signing)

**Build Requirements (if building from source):**

- Go 1.21+
- Make (optional, but recommended)

## 📖 Documentation

- **[Complete Command Reference](docs/commands.md)** - All commands and examples
- **[Use Cases](docs/use-cases.md)** - Team setups, freelancing, organizations
- **[Security Features](docs/security.md)** - GPG signing, SSH keys, best practices
- **[Installation Guide](docs/installation.md)** - Platform-specific instructions
- **[Features](docs/features.md)** - Technical details and architecture
- **[User Guide](docs/user-guide.md)** - Workflows and best practices

## 🏗️ Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Format code
make fmt

# Show all available commands
make help
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

```bash
git clone https://github.com/calghar/gas-cli.git
cd gas-cli
make deps
make build
./bin/gascli --help
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/calghar/gas-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/calghar/gas-cli/discussions)
- **Documentation**: [Complete Docs](docs/)
