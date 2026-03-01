# Installation

## Requirements

- Git 2.23+ (for includeIf support)
- SSH (for key-based authentication)
- GPG (optional, for commit signing)

## Binary Installation

### macOS

```bash
# Apple Silicon
curl -L https://github.com/calghar/gas-cli/releases/latest/download/gascli-darwin-arm64 -o gascli
chmod +x gascli
sudo mv gascli /usr/local/bin/

# Intel
curl -L https://github.com/calghar/gas-cli/releases/latest/download/gascli-darwin-amd64 -o gascli
chmod +x gascli
sudo mv gascli /usr/local/bin/
```

### Linux

```bash
curl -L https://github.com/calghar/gas-cli/releases/latest/download/gascli-linux-amd64 -o gascli
chmod +x gascli
sudo mv gascli /usr/local/bin/
```

### Windows

Download from [releases page](https://github.com/calghar/gas-cli/releases).

## Build from Source

```bash
git clone https://github.com/calghar/gas-cli.git
cd gas-cli
make build
make install  # Installs to Go bin ($GOPATH/bin or $GOBIN)
```

## Install with Go

```bash
go install github.com/calghar/gas-cli@latest
```

## Verify Installation

```bash
gascli --version
gascli --help
```
