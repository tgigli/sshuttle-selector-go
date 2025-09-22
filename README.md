# SSH Tunnel Selector

A modern, interactive TUI (Terminal User Interface) for managing SSH tunnels with `sshuttle`. Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![SSH Tunnel Selector Demo](demo.gif)

## Features

- 🔍 **Interactive Selection**: Browse and select tunnels using a beautiful TUI
- 🔄 **Active Tunnel Management**: View and terminate running tunnels
- 📝 **YAML Configuration**: Simple configuration file format
- 🚀 **Quick Launch**: Start tunnels with a single keypress
- 🐛 **Debug Mode**: Verbose logging for troubleshooting
- 💾 **Daemon Mode**: Tunnels run in background by default
- 🎯 **Smart SSH Key Handling**: Automatically uses SSH keys from config

## Installation

### Using Homebrew

```bash
# Install sshuttle (dependency)
brew install sshuttle

# Add tap and install sshuttle-selector
brew tap tgigli/sshuttle-selector
brew install sshuttle-selector
```

### Prerequisites

- `sshuttle` installed via Homebrew
- SSH access to your servers

## Configuration

Create the configuration directory and file:

```bash
mkdir -p ~/.config/sshuttle-selector
```

Create `~/.config/sshuttle-selector/config.yaml`:

```yaml
tunnels:
  - name: "Development Server"
    host: "dev.example.com"
    user: "ubuntu"
    subnets: "10.0.0.0/8"
    extra_args: "-i ~/.ssh/dev-key.pem"

  - name: "Production Server"
    host: "prod.example.com"
    user: "ubuntu"
    subnets: "10.1.0.0/16"
    extra_args: "-i ~/.ssh/prod-key.pem"

  - name: "AWS VPC"
    host: "bastion.example.com"
    user: "ec2-user"
    subnets: "172.16.0.0/12"
    extra_args: "-i ~/.ssh/aws-key.pem --dns"
```

### Configuration Options

| Field | Description | Required |
|-------|-------------|----------|
| `name` | Display name for the tunnel | Yes |
| `host` | SSH server hostname | Yes |
| `user` | SSH username | Yes |
| `subnets` | CIDR ranges to tunnel (comma-separated) | Yes |
| `extra_args` | Additional sshuttle arguments | No |

## Usage

### Interactive Mode

```bash
# Start the selector
sshuttle-selector

# Start with debug mode (verbose logging, no daemon)
sshuttle-selector --debug
```

### CLI Mode - Add Configuration

Add new tunnel configurations directly from command line:

```bash
# Add a new tunnel configuration
sshuttle-selector -add \
  -name "Production Server" \
  -host "prod.example.com" \
  -user "ubuntu" \
  -subnets "10.0.0.0/8" \
  -extra-args "-i ~/.ssh/prod-key.pem"

# Add tunnel with multiple subnets
sshuttle-selector -add \
  -name "Corporate VPN" \
  -host "vpn.company.com" \
  -user "employee" \
  -subnets "10.0.0.0/8,172.16.0.0/12" \
  -extra-args "--dns"

# Simple tunnel without extra arguments
sshuttle-selector -add \
  -name "Dev Server" \
  -host "dev.example.com" \
  -user "developer" \
  -subnets "192.168.1.0/24"
```

#### CLI Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `-add` | Yes | Enable CLI add mode |
| `-name` | Yes | Tunnel display name |
| `-host` | Yes | SSH server hostname |
| `-user` | Yes | SSH username |
| `-subnets` | Yes | CIDR ranges (comma-separated) |
| `-extra-args` | No | Additional sshuttle arguments |

#### CLI Validation

The CLI mode performs the following validations:

1. **Required Parameters**: Ensures all mandatory fields are provided
2. **CIDR Validation**: Validates subnet format (e.g., `10.0.0.0/8`)
3. **SSH Connectivity Test**: Attempts to connect to verify access
4. **Duplicate Check**: Prevents duplicate tunnel names
5. **Configuration Backup**: Creates config directory if needed

#### CLI Examples

```bash
# Valid examples
sshuttle-selector -add -name "Test" -host "test.com" -user "root" -subnets "10.0.0.0/8"

# Invalid examples (will show error)
sshuttle-selector -add -name "Test"  # Missing required parameters
sshuttle-selector -add -name "Test" -host "test.com" -user "root" -subnets "invalid"  # Bad CIDR

# Exit codes
# 0: Success
# 1: Error (missing params, validation failed, etc.)
```

### Interface

The TUI is organized into sections:

#### ACTIVE TUNNEL
- Shows the currently running sshuttle process (only one tunnel can be active)
- Click to terminate the active tunnel
- Starting a new tunnel automatically stops the existing one

#### AVAILABLE TUNNELS
- Shows configured tunnels from your YAML file
- Click to start a new tunnel

### Navigation

- `↑/↓` - Navigate through options
- `Enter` - Select/execute action
- `/` - Search/filter tunnels
- `q` or `Ctrl+C` - Quit

## Examples

### Basic Tunnel
```yaml
- name: "Simple Tunnel"
  host: "server.example.com"
  user: "myuser"
  subnets: "192.168.1.0/24"
```

### Tunnel with SSH Key
```yaml
- name: "Secure Tunnel"
  host: "secure.example.com"
  user: "admin"
  subnets: "10.0.0.0/8"
  extra_args: "-i ~/.ssh/secure-key.pem"
```

### Multiple Subnets with DNS
```yaml
- name: "Corporate VPN"
  host: "vpn.company.com"
  user: "employee"
  subnets: "10.0.0.0/8,172.16.0.0/12"
  extra_args: "--dns"
```

## Debug Mode

```bash
sshuttle-selector --debug
```

This is useful for troubleshooting connection issues.

## How It Works

1. **Configuration Loading**: Reads `~/.config/sshuttle-selector/config.yaml`
2. **Process Detection**: Uses `ps aux` to find running sshuttle processes
3. **Command Building**: Constructs sshuttle commands with proper SSH options
4. **Execution**: Runs commands via shell for proper quote handling

## Troubleshooting

### Common Issues

1. **"sshuttle: command not found"**
   - Install sshuttle: `brew install sshuttle`

2. **SSH key not found**
   - Check the path in `extra_args`
   - Ensure proper permissions: `chmod 600 ~/.ssh/key.pem`

3. **Permission denied**
   - Verify SSH access: `ssh -i ~/.ssh/key.pem user@host`
   - Check SSH agent: `ssh-add ~/.ssh/key.pem`

4. **No tunnels showing**
   - Check config file location: `~/.config/sshuttle-selector/config.yaml`
   - Validate YAML syntax

### Debug Output

Use debug mode to see detailed connection logs:

```bash
sshuttle-selector --debug
# Select a tunnel to see verbose SSH and sshuttle output
```

## Development

### Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [YAML v3](https://gopkg.in/yaml.v3) - Configuration parsing

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Related Projects

- [sshuttle](https://github.com/sshuttle/sshuttle) - The underlying VPN tool
- [fzf](https://github.com/junegunn/fzf) - Command-line fuzzy finder inspiration
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework

---

**Note**: This tool is a wrapper around `sshuttle`. Make sure you understand the security implications of SSH tunneling in your environment.