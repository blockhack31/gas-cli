# User Guide

## Workflow

### 1. Add Profiles

```bash
gascli add work john@company.com "John Doe" ABC123GPG
gascli add personal john@personal.com "John Smith"
```

### 2. Setup Directory Rules (Recommended)

```bash
gascli auto ~/work work
gascli auto ~/personal personal
```

Git now automatically uses the correct profile based on repository location. No manual switching needed.

### 3. Alternative: Manual Switching

```bash
gascli switch work
gascli --auto-ssh switch personal  # Also loads SSH key
```

## Git includeIf Workflow

The tool creates profile-specific gitconfig files and uses Git's `includeIf` directive:

**~/.gitconfig:**
```ini
[includeIf "gitdir:~/work/"]
    path = ~/.gitconfig-work
```

**~/.gitconfig-work:**
```ini
[user]
    email = john@company.com
    name = John Doe
    signingkey = ABC123GPG
[commit]
    gpgsign = true
```

Navigate to `~/work/any-repo` and Git automatically uses work profile.

## SSH Multi-Account

SSH config entries use `IdentitiesOnly yes` to prevent key conflicts:

```bash
# Clone with profile-specific host
git clone git@github.com-work:company/repo.git

# Update existing repo
git remote set-url origin git@github.com-personal:user/repo.git
```

## Multi-Email Profiles

```bash
gascli add-email work john.contractor@company.com
gascli switch work john.contractor@company.com  # Use specific email
gascli list-emails work
```

## Import/Export

```bash
# Backup
gascli export > backup.json

# Restore on new machine
gascli import backup.json
```

## GPG Signing

Automatically configured per profile. Ensure GPG key exists:

```bash
gpg --list-secret-keys --keyid-format LONG
```

## Platform-Specific Notes

**macOS:** SSH keys added to keychain with `--apple-use-keychain`
**Linux:** Standard SSH agent
**Windows:** Windows SSH agent
