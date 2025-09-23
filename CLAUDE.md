# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an SSH Tunnel Selector built with Go and the Bubble Tea TUI framework. It provides an interactive terminal interface for managing SSH tunnels via `sshuttle`, allowing users to start, stop, and configure tunnels through a user-friendly menu system.

## Architecture

**Single-file Architecture**: The entire application is contained in `main.go` (~711 lines), structured as follows:

- **TUI Components**: Uses Bubble Tea model-view-update pattern with a single `model` struct containing a `list.Model`
- **Configuration Management**: YAML-based config at `~/.config/sshuttle-selector/config.yaml` with `TunnelConfig` and `Config` structs
- **Process Management**: Direct system calls to manage sshuttle processes using `ps aux` parsing and `kill` commands
- **Item System**: Three item types (`ItemActiveTunnel`, `ItemAvailableTunnel`, `ItemAction`) represent different UI elements

**Key Data Structures**:
- `TunnelConfig`: YAML configuration for tunnel definitions
- `activeTunnel`: Runtime representation of running sshuttle processes
- `item`: UI list items with type, command, and display information
- `model`: Main TUI state containing list and selection state

## Common Commands

### Build and Run
```bash
# Build the application
go build -o sshuttle-selector main.go

# Run directly
go run main.go

# Run with debug mode (verbose logging, no daemon)
go run main.go --debug
```

### CLI Operations
```bash
# Add new tunnel configuration via CLI
go run main.go -add -name "Test Server" -host "test.com" -user "ubuntu" -subnets "10.0.0.0/8" -extra-args "-i ~/.ssh/key.pem"

# Interactive TUI mode (default)
go run main.go
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

## Configuration System

**Config Location**: `~/.config/sshuttle-selector/config.yaml`

**Structure**:
```yaml
tunnels:
  - name: "Server Name"
    host: "hostname"
    user: "username"
    subnets: "10.0.0.0/8"
    extra_args: "-i ~/.ssh/key.pem --dns"
```

The application automatically handles:
- SSH key extraction from `extra_args`
- CIDR subnet validation
- SSH connectivity testing
- Duplicate name prevention

## Process Management

**sshuttle Command Generation**:
- Normal mode: `sshuttle -r user@host subnets --daemon --ssh-cmd="ssh options"`
- Debug mode: `sshuttle -v -r user@host subnets --ssh-cmd="ssh -vvv options"` (no daemon)

**Process Detection**: Uses `ps aux` with regex parsing to find running sshuttle processes and extract PID/destination information.

## Key Functions

- `loadAllItems()`: Main function that builds the TUI list by combining active tunnels and config tunnels
- `getActiveTunnels()`: Parses system processes to find running sshuttle instances
- `handleAddCommand()`: CLI mode for adding tunnel configurations with validation
- `loadConfigTunnels()`: Loads and parses YAML config into TUI items
- `killTunnel()/killAllTunnels()`: Process termination functions

## Dependencies

Uses these Go modules:
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components (list, textinput)
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `gopkg.in/yaml.v3` - YAML configuration parsing

## External Requirements

- `sshuttle` command must be available in PATH
- SSH access to target servers
- SSH keys configured if using key-based authentication