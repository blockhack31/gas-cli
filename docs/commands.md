# Command Reference

## Profile Management

```bash
gascli add <name> <email> [git-name] [gpg-key] [--pat TOKEN]
gascli list | gascli ls
gascli current
gascli remove <name>
```

## PAT (Personal Access Token)

```bash
gascli pat set <profile> <token>
gascli pat clear <profile>
```

For HTTPS authentication. Stored in `~/.gascli/config.json` and `~/.gascli/credentials-{profile}`.

## Directory-Based Switching (Git includeIf)

```bash
# Primary workflow - set up once, automatic thereafter
gascli auto <directory> <profile>
gascli auto-list
gascli auto-remove <directory>
```

Creates `.gitconfig-{profile}` files and adds `includeIf` directives. Git automatically loads the correct config based on repository location.

## Manual Switching

```bash
gascli switch <profile> [email]
gascli --auto-ssh switch <profile>  # Also adds SSH key to keychain
```

Modifies global git config. Use `auto` for directory-based switching instead.

## Email Management

```bash
gascli add-email <profile> <email>
gascli remove-email <profile> <email>
gascli list-emails <profile>
```

## Import/Export

```bash
gascli export [file]     # Prints to stdout if no file
gascli import <file>
```

## Global Flags

- `--auto-ssh, -s`: Add SSH key to platform keychain
- `--yes, -y`: Skip confirmations
- `--help, -h`: Command help
- `--version, -v`: Show version

## SSH Configuration

Automatically creates entries in `~/.ssh/config`:

```
Host github.com-{profile}
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_{profile}
    IdentitiesOnly yes
```

Use in git URLs: `git@github.com-work:user/repo.git`
