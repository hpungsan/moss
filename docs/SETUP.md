# Moss Setup

Installation and storage paths. Primitive-specific usage lives in the primitive runbooks.

## Install

### From Source

```bash
git clone https://github.com/hpungsan/moss.git
cd moss
```

Then choose one:

| Goal | Command |
|------|---------|
| Build locally | `go build -o bin/moss ./cmd/moss` |
| Install to PATH | `go install ./cmd/moss` |

**Alternative: Using Makefile**

| Goal | Command |
|------|---------|
| Build locally | `make build` → `bin/moss` |
| Install to PATH | `make install` → `$GOPATH/bin/moss` |
| Build with version | `make build-release VERSION=1.0.0` |
| Cross-compile all platforms | `make build-all VERSION=1.0.0` |
| Cross-compile + checksums | `make build-checksums VERSION=1.0.0` |

### Pre-built Binaries

Download from GitHub Releases:

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `moss-darwin-arm64` |
| macOS (Intel) | `moss-darwin-amd64` |
| Linux (x64) | `moss-linux-amd64` |
| Linux (ARM64) | `moss-linux-arm64` |
| Windows (x64) | `moss-windows-amd64.exe` |

```bash
# Example: macOS Apple Silicon
curl -LO https://github.com/hpungsan/moss/releases/latest/download/moss-darwin-arm64
chmod +x moss-darwin-arm64
sudo mv moss-darwin-arm64 /usr/local/bin/moss
```

### Verify Build

```bash
# Check version
moss --version

# Show help
moss --help

# Test MCP protocol (piped input = MCP server mode)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | moss
```

## Storage

- DB: `~/.moss/moss.db`
- Config: `~/.moss/config.json` (global), `.moss/config.json` (repo override)

## MCP Usage

Moss runs as an MCP server when invoked by an MCP client (e.g., Claude Code). See [integrations/claude-code.md](integrations/claude-code.md).

## CLI Usage

The CLI mirrors capsule operations for debugging and scripting:

```bash
echo "## Objective\n..." | moss store --name=auth
moss fetch --name=auth
moss inventory
```

## Primitive Runbooks

- Capsules: `docs/capsule/RUNBOOK.md`
