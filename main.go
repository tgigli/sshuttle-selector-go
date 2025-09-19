package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

var (
	// Clean color palette
	primaryColor   = lipgloss.Color("39")  // Blue
	successColor   = lipgloss.Color("42")  // Green
	warningColor   = lipgloss.Color("214") // Orange
	dangerColor    = lipgloss.Color("196") // Red
	subtleColor    = lipgloss.Color("245") // Gray
	selectedColor  = lipgloss.Color("51")  // Cyan

	// Simple styles
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		MarginLeft(2).
		MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(subtleColor).
		MarginTop(1).
		MarginLeft(2)

	activeItemStyle = lipgloss.NewStyle().
		Foreground(successColor).
		MarginLeft(4)

	availableItemStyle = lipgloss.NewStyle().
		MarginLeft(4)

	actionItemStyle = lipgloss.NewStyle().
		Foreground(warningColor).
		MarginLeft(4)

	dangerItemStyle = lipgloss.NewStyle().
		Foreground(dangerColor).
		MarginLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(selectedColor).
		MarginLeft(2).
		PaddingLeft(1).
		PaddingRight(1)

	statusStyle = lipgloss.NewStyle().
		Foreground(subtleColor).
		Italic(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(subtleColor).
		MarginTop(1).
		MarginLeft(2)

	quitTextStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Margin(1, 0, 2, 2)

	debugMode = false
)

type itemType int

const (
	ItemActiveTunnel itemType = iota
	ItemAvailableTunnel
	ItemAction
)

type item struct {
	name        string
	destination string
	command     string
	itemType    itemType
	pid         int // for active tunnels
}

type activeTunnel struct {
	PID         int
	Command     string
	Destination string
}

type TunnelConfig struct {
	Name        string `yaml:"name"`
	Host        string `yaml:"host"`
	User        string `yaml:"user"`
	Subnets     string `yaml:"subnets"`
	ExtraArgs   string `yaml:"extra_args,omitempty"`
}

type Config struct {
	Tunnels []TunnelConfig `yaml:"tunnels"`
}

func (i item) FilterValue() string { return i.name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	if i.name == "" {
		return
	}

	var content string
	var style lipgloss.Style

	switch i.itemType {
	case ItemAction:
		if strings.Contains(i.name, "CURRENT TUNNEL") {
			content = "CURRENT TUNNEL"
			style = sectionStyle
		} else if strings.Contains(i.name, "AVAILABLE TUNNELS") {
			content = "AVAILABLE TUNNELS"
			style = sectionStyle
		} else if strings.Contains(i.name, "Add New") {
			content = "+ Add New Tunnel"
			style = actionItemStyle
		} else {
			content = i.name
			style = sectionStyle
		}

	case ItemActiveTunnel:
		// Show current active tunnel with stop hint
		content = strings.Replace(i.name, "●", "●", 1) // Keep the bullet
		style = activeItemStyle

	case ItemAvailableTunnel:
		content = fmt.Sprintf("  %s", i.name)
		style = availableItemStyle

	default:
		content = i.name
		style = availableItemStyle
	}

	// Apply selection highlighting
	if index == m.Index() && i.name != "" {
		if !isSelectableItem(i) {
			// Don't highlight non-selectable items
			fmt.Fprint(w, style.Render(content))
		} else {
			fmt.Fprint(w, selectedItemStyle.Render("> "+content))
		}
	} else {
		fmt.Fprint(w, style.Render(content))
	}
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
	filter   textinput.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func isSelectableItem(i item) bool {
	// Section headers and empty separators are not selectable
	if i.itemType == ItemAction && (strings.Contains(i.name, "TUNNEL") || i.name == "") {
		return false
	}
	return true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			// Navigate up, skipping non-selectable items
			currentIndex := m.list.Index()
			for i := currentIndex - 1; i >= 0; i-- {
				if item, ok := m.list.Items()[i].(item); ok && isSelectableItem(item) {
					m.list.Select(i)
					break
				}
			}
			return m, nil

		case "down", "j":
			// Navigate down, skipping non-selectable items
			currentIndex := m.list.Index()
			items := m.list.Items()
			for i := currentIndex + 1; i < len(items); i++ {
				if item, ok := items[i].(item); ok && isSelectableItem(item) {
					m.list.Select(i)
					break
				}
			}
			return m, nil

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok && isSelectableItem(i) {
				// Handle different item types
				switch i.itemType {
				case ItemActiveTunnel:
					// Kill current tunnel
					if err := killTunnel(i.pid); err != nil {
						m.choice = fmt.Sprintf("Failed to stop tunnel: %v", err)
					} else {
						m.choice = fmt.Sprintf("Tunnel stopped: %s", i.destination)
					}
				case ItemAvailableTunnel:
					// Kill any existing tunnel first, then start new one
					if err := killAllTunnels(); err != nil {
						log.Printf("Warning: Failed to kill existing tunnels: %v", err)
					}
					// Start the selected tunnel
					m.choice = i.command
				case ItemAction:
					if i.command == "add_new" {
						m.choice = "add_new_tunnel"
					}
				}
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(m.choice)
	}
	if m.quitting {
		return quitTextStyle.Render("Goodbye!")
	}

	helpText := helpStyle.Render("↑/↓ navigate • enter select • q quit • / search")

	return m.list.View() + "\n" + helpText
}

func getActiveTunnels() ([]activeTunnel, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tunnels []activeTunnel
	scanner := bufio.NewScanner(bytes.NewReader(output))
	re := regexp.MustCompile(`sshuttle.*-r\s+(\S+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "sshuttle") && strings.Contains(line, "-r") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				pid, err := strconv.Atoi(fields[1])
				if err != nil {
					continue
				}

				matches := re.FindStringSubmatch(line)
				destination := "unknown"
				if len(matches) > 1 {
					destination = matches[1]
				}

				tunnels = append(tunnels, activeTunnel{
					PID:         pid,
					Command:     line,
					Destination: destination,
				})
			}
		}
	}

	return tunnels, nil
}

func killTunnel(pid int) error {
	cmd := exec.Command("kill", strconv.Itoa(pid))
	return cmd.Run()
}

func killAllTunnels() error {
	tunnels, err := getActiveTunnels()
	if err != nil {
		return err
	}

	for _, tunnel := range tunnels {
		if err := killTunnel(tunnel.PID); err != nil {
			log.Printf("Failed to kill tunnel %d: %v", tunnel.PID, err)
		}
	}

	return nil
}

func loadAllItems() ([]list.Item, error) {
	var items []list.Item

	// Get active tunnels (should be only one now)
	activeTunnels, err := getActiveTunnels()
	if err != nil {
		log.Printf("Error getting active tunnels: %v", err)
	}

	// Add current active tunnel (if any)
	if len(activeTunnels) > 0 {
		// Take only the first active tunnel (single tunnel mode)
		tunnel := activeTunnels[0]
		items = append(items, item{
			name:     "CURRENT TUNNEL",
			itemType: ItemAction,
			command:  "",
		})

		items = append(items, item{
			name:        fmt.Sprintf("● %s (PID: %d) - Click to stop", tunnel.Destination, tunnel.PID),
			destination: tunnel.Destination,
			command:     fmt.Sprintf("kill %d", tunnel.PID),
			itemType:    ItemActiveTunnel,
			pid:         tunnel.PID,
		})

		// Add separator
		items = append(items, item{
			name:     "",
			itemType: ItemAction,
			command:  "",
		})
	}

	// Add available tunnels section
	items = append(items, item{
		name:     "AVAILABLE TUNNELS",
		itemType: ItemAction,
		command:  "",
	})

	// Load config tunnels
	configItems, err := loadConfigTunnels()
	if err != nil {
		return nil, err
	}

	items = append(items, configItems...)

	// Add separator and new tunnel option
	items = append(items, item{
		name:     "",
		itemType: ItemAction,
		command:  "",
	})
	items = append(items, item{
		name:     "+ Add New Tunnel",
		itemType: ItemAction,
		command:  "add_new",
	})

	return items, nil
}

func loadConfigTunnels() ([]list.Item, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "sshuttle-selector", "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return []list.Item{
			item{
				name:        "Example Server",
				destination: "user@example.com",
				command:     "sshuttle -r user@example.com 10.0.0.0/8",
				itemType:    ItemAvailableTunnel,
			},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	items := make([]list.Item, len(config.Tunnels))
	for i, tunnel := range config.Tunnels {
		// Build SSH command with key if specified
		sshCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no")
		if strings.Contains(tunnel.ExtraArgs, "-i ") {
			// Extract key path from extra_args
			keyPath := strings.TrimSpace(strings.Split(tunnel.ExtraArgs, "-i ")[1])
			sshCmd += fmt.Sprintf(" -i %s", keyPath)
		}

		// Add debug flags if in debug mode
		if debugMode {
			sshCmd += " -vvv"
		}

		// Build sshuttle command
		var command string
		if debugMode {
			// In debug mode, don't use --daemon and add -v flag
			command = fmt.Sprintf("sshuttle -v -r %s@%s %s --ssh-cmd=\"%s\"", tunnel.User, tunnel.Host, tunnel.Subnets, sshCmd)
		} else {
			// Normal mode uses --daemon
			command = fmt.Sprintf("sshuttle -r %s@%s %s --daemon --ssh-cmd=\"%s\"", tunnel.User, tunnel.Host, tunnel.Subnets, sshCmd)
		}

		// Add other extra args (excluding -i)
		if tunnel.ExtraArgs != "" && !strings.Contains(tunnel.ExtraArgs, "-i ") {
			command += " " + tunnel.ExtraArgs
		}

		items[i] = item{
			name:        tunnel.Name,
			destination: fmt.Sprintf("%s@%s", tunnel.User, tunnel.Host),
			command:     command,
			itemType:    ItemAvailableTunnel,
		}
	}

	return items, nil
}

func handleAddCommand(name, host, user, subnets, extraArgs string) error {
	// Validate required parameters
	if name == "" {
		return fmt.Errorf("tunnel name is required (use -name)")
	}
	if host == "" {
		return fmt.Errorf("SSH hostname is required (use -host)")
	}
	if user == "" {
		return fmt.Errorf("SSH username is required (use -user)")
	}
	if subnets == "" {
		return fmt.Errorf("subnets are required (use -subnets)")
	}

	// Validate subnet format
	if err := validateSubnets(subnets); err != nil {
		return fmt.Errorf("invalid subnet format: %v", err)
	}

	// Validate SSH connectivity (optional test)
	if err := validateSSHConnection(user, host, extraArgs); err != nil {
		fmt.Printf("Warning: SSH connectivity test failed: %v\n", err)
		fmt.Print("Continue anyway? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("operation cancelled")
		}
	}

	// Load existing config or create new one
	config, err := loadOrCreateConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Check for duplicate names
	for _, tunnel := range config.Tunnels {
		if tunnel.Name == name {
			return fmt.Errorf("tunnel with name '%s' already exists", name)
		}
	}

	// Add new tunnel
	newTunnel := TunnelConfig{
		Name:      name,
		Host:      host,
		User:      user,
		Subnets:   subnets,
		ExtraArgs: extraArgs,
	}

	config.Tunnels = append(config.Tunnels, newTunnel)

	// Save config
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	return nil
}

func validateSubnets(subnets string) error {
	// Split by comma and validate each CIDR
	subnetsSlice := strings.Split(subnets, ",")
	for _, subnet := range subnetsSlice {
		subnet = strings.TrimSpace(subnet)
		if _, _, err := net.ParseCIDR(subnet); err != nil {
			return fmt.Errorf("invalid CIDR '%s': %v", subnet, err)
		}
	}
	return nil
}

func validateSSHConnection(user, host, extraArgs string) error {
	// Build SSH test command
	sshArgs := []string{"-o", "ConnectTimeout=10", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no"}

	// Parse extra args for SSH key
	if strings.Contains(extraArgs, "-i ") {
		keyPath := strings.TrimSpace(strings.Split(extraArgs, "-i ")[1])
		keyPath = strings.Split(keyPath, " ")[0] // Take only the key path
		sshArgs = append(sshArgs, "-i", keyPath)
	}

	// Add user@host
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, host), "exit")

	// Test SSH connection
	cmd := exec.Command("ssh", sshArgs...)
	return cmd.Run()
}

func loadOrCreateConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "sshuttle-selector", "config.yaml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config
		return &Config{Tunnels: []TunnelConfig{}}, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "sshuttle-selector", "config.yaml")

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

func main() {
	// Parse command line flags
	debugFlag := flag.Bool("debug", false, "Enable debug mode (adds -v to sshuttle and -vvv to ssh)")
	addFlag := flag.Bool("add", false, "Add new tunnel configuration")
	nameFlag := flag.String("name", "", "Tunnel name (required with -add)")
	hostFlag := flag.String("host", "", "SSH hostname (required with -add)")
	userFlag := flag.String("user", "", "SSH username (required with -add)")
	subnetsFlag := flag.String("subnets", "", "CIDR subnets to tunnel (required with -add)")
	extraArgsFlag := flag.String("extra-args", "", "Additional sshuttle arguments (optional)")

	flag.Parse()

	debugMode = *debugFlag

	// Handle CLI mode for adding configurations
	if *addFlag {
		if err := handleAddCommand(*nameFlag, *hostFlag, *userFlag, *subnetsFlag, *extraArgsFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Tunnel configuration added successfully!")
		os.Exit(0)
	}

	items, err := loadAllItems()
	if err != nil {
		log.Printf("Error loading items: %v", err)
		log.Fatal("Failed to load configuration")
	}

	const defaultList = 20
	l := list.New(items, itemDelegate{}, defaultWidth, defaultList)
	l.Title = "SSH Tunnel Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle

	// Find first selectable item and set it as selected
	for i, listItem := range items {
		if item, ok := listItem.(item); ok && isSelectableItem(item) {
			l.Select(i)
			break
		}
	}

	m := model{list: l}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Handle the selected action
	if finalModel := result.(model); finalModel.choice != "" {
		if finalModel.choice == "add_new_tunnel" {
			fmt.Println("Coming soon: Interactive tunnel creation")
		} else if strings.HasPrefix(finalModel.choice, "Tunnel stopped:") ||
				  strings.HasPrefix(finalModel.choice, "Failed to stop") ||
				  strings.HasPrefix(finalModel.choice, "All tunnels killed") ||
				  strings.HasPrefix(finalModel.choice, "Failed to kill") {
			// Just print the status message
			fmt.Println(finalModel.choice)
		} else {
			// Execute sshuttle command
			fmt.Printf("Starting tunnel...\n")

			// Use shell to execute the command properly
			cmd := exec.Command("sh", "-c", finalModel.choice)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			if err := cmd.Run(); err != nil {
				fmt.Printf("Error executing command: %v\n", err)
				os.Exit(1)
			}
		}
	}
}