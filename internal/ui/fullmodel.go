package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/klejdi94/docker-tea/internal/config"
	"github.com/klejdi94/docker-tea/internal/docker"
	"github.com/klejdi94/docker-tea/internal/ui/views"
)

// Icons for UI elements
const (
	// Resource type icons
	IconContainer = "🐳 "
	IconImage     = "📦 "
	IconVolume    = "💾 "
	IconNetwork   = "🌐 "
	IconCompose   = "🔄 "

	// Status icons
	IconRunning    = "🟢 "
	IconStopped    = "🔴 "
	IconPaused     = "⏸️  "
	IconCreated    = "🆕 "
	IconRestarting = "🔄 "
	IconExited     = "⏹️  "
	IconDead       = "💀 "

	// Action icons
	IconInspect = "🔍 "
	IconLogs    = "📜 "
	IconMonitor = "📊 "
	IconRefresh = "🔄 "
	IconStart   = "▶️  "
	IconStop    = "⏹️  "
	IconRestart = "🔁 "
	IconPause   = "⏸️  "
	IconUnpause = "⏯️  "
	IconKill    = "⚡ "
	IconRemove  = "🗑️  "

	// Navigation icons
	IconBack = "← "
	IconHelp = "❓ "
	IconQuit = "🚪 "

	// Alert icons
	IconWarning = "⚠️ "
	IconError   = "🚨 "
	IconInfo    = "ℹ️ "
)

// Tab is an enum for different tabs
type Tab int

const (
	ContainersTab Tab = iota
	ImagesTab
	VolumesTab
	NetworksTab
	ComposeTab
	LogsTab
)

// ResourceMode tracks current UI mode
type Mode int

const (
	ListMode Mode = iota
	InspectMode
	LogsMode
	MonitorMode
)

// FullModel represents the complete Bubble Tea model for Docker TUI
type FullModel struct {
	config                   *config.Config
	docker                   *docker.Service
	ctx                      context.Context
	width                    int
	height                   int
	loading                  bool
	err                      error
	dockerConnected          bool
	containerTable           table.Model
	imageTable               table.Model
	volumeTable              table.Model
	networkTable             table.Model
	composeTable             table.Model
	viewport                 viewport.Model
	currentTab               Tab
	currentMode              Mode
	statusMsg                string
	containers               []docker.ContainerInfo
	images                   []docker.ImageInfo
	volumes                  []docker.VolumeInfo
	networks                 []docker.NetworkInfo
	composeProjects          []docker.ComposeInfo
	logContent               string
	inspectContent           string
	statsContent             string
	selectedID               string
	selectedName             string
	selectedPath             string
	showHelp                 bool
	ticker                   *time.Ticker
	composeServices          []docker.ComposeServiceInfo
	composeServicesLoading   bool
	selectedProject          string
	selectedProjectPath      string
	loadingCompose           bool
	spinner                  spinner.Model
	composeContainers        []docker.ContainerInfo
	composeContainersLoading bool
}

// FullKeyMap defines the keybindings for the application
type FullKeyMap struct {
	// Global
	Quit key.Binding
	Help key.Binding

	// Navigation
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	GoToTop    key.Binding
	GoToBottom key.Binding

	// Tab navigation
	NextTab key.Binding
	PrevTab key.Binding

	// Resource management
	Refresh key.Binding
	Inspect key.Binding
	Logs    key.Binding
	Monitor key.Binding
	Back    key.Binding

	// Container actions
	Start   key.Binding
	Stop    key.Binding
	Restart key.Binding
	Pause   key.Binding
	Resume  key.Binding
	Kill    key.Binding
	Remove  key.Binding

	// Compose actions
	ComposeUp   key.Binding
	ComposeDown key.Binding
	ComposePull key.Binding
}

var FullKeyMapHelp = [][]key.Binding{
	// Global
	{
		DefaultFullKeyMap.Quit,
		DefaultFullKeyMap.Help,
		DefaultFullKeyMap.Refresh,
	},
	// Navigation
	{
		DefaultFullKeyMap.Up,
		DefaultFullKeyMap.Down,
		DefaultFullKeyMap.NextTab,
		DefaultFullKeyMap.PrevTab,
	},
	// Resource Actions
	{
		DefaultFullKeyMap.Inspect,
		DefaultFullKeyMap.Logs,
		DefaultFullKeyMap.Monitor,
		DefaultFullKeyMap.Back,
	},
	// Container Actions
	{
		DefaultFullKeyMap.Start,
		DefaultFullKeyMap.Stop,
		DefaultFullKeyMap.Restart,
		DefaultFullKeyMap.Pause,
		DefaultFullKeyMap.Resume,
		DefaultFullKeyMap.Kill,
		DefaultFullKeyMap.Remove,
	},
	// Compose Actions
	{
		DefaultFullKeyMap.ComposeUp,
		DefaultFullKeyMap.ComposeDown,
		DefaultFullKeyMap.ComposePull,
	},
}

var DefaultFullKeyMap = FullKeyMap{
	// Global
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),

	// Navigation
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	GoToTop: key.NewBinding(
		key.WithKeys("home"),
		key.WithHelp("home", "go to top"),
	),
	GoToBottom: key.NewBinding(
		key.WithKeys("end"),
		key.WithHelp("end", "go to bottom"),
	),

	// Tab navigation
	NextTab: key.NewBinding(
		key.WithKeys("tab", "right"),
		key.WithHelp("tab/→", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("shift+tab", "left"),
		key.WithHelp("shift+tab/←", "prev tab"),
	),

	// Resource inspection
	Inspect: key.NewBinding(
		key.WithKeys("i", "enter"),
		key.WithHelp("i/enter", "inspect"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Monitor: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "monitor"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),

	// Container actions
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	Stop: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "stop"),
	),
	Restart: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "restart"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pause"),
	),
	Resume: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unpause"),
	),
	Kill: key.NewBinding(
		key.WithKeys("K"),
		key.WithHelp("K", "kill"),
	),
	Remove: key.NewBinding(
		key.WithKeys("delete"),
		key.WithHelp("delete", "remove"),
	),

	// Compose actions
	ComposeUp: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "up"),
	),
	ComposeDown: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "down"),
	),
	ComposePull: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pull"),
	),
}

// NewFullModel creates a new model for Docker Tea
func NewFullModel(dockerService *docker.Service, config *config.Config, ctx context.Context) FullModel {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := FullModel{
		config:            config,
		docker:            dockerService,
		ctx:               ctx,
		loading:           true,
		dockerConnected:   true, // Assume connected, we'll check immediately
		statusMsg:         "Initializing...",
		currentTab:        ContainersTab,
		currentMode:       ListMode,
		viewport:          viewport.New(0, 0),
		spinner:           s,
		composeContainers: []docker.ContainerInfo{},
	}

	return m
}

// Init initializes the model
func (m FullModel) Init() tea.Cmd {
	return tea.Batch(
		m.checkDockerConnection,
		m.fetchContainers,
		m.fetchImages,
		m.fetchVolumes,
		m.fetchNetworks,
		m.fetchComposeProjects,
	)
}

// checkDockerConnection verifies Docker is running
func (m FullModel) checkDockerConnection() tea.Msg {
	_, err := m.docker.Ping(m.ctx)
	if err != nil {
		return dockerConnectionMsg{connected: false, err: err}
	}
	return dockerConnectionMsg{connected: true}
}

// startConnectionCheck periodically checks Docker connection
func (m FullModel) startConnectionCheck() tea.Cmd {
	return tea.Tick(time.Second*10, func(t time.Time) tea.Msg {
		return connectionCheckTickMsg{}
	})
}

// fetchContainers fetches container data from Docker
func (m FullModel) fetchContainers() tea.Msg {
	if !m.dockerConnected {
		return fullContainersMsg{containers: []docker.ContainerInfo{}}
	}

	m.statusMsg = "Fetching containers..."
	containers, err := m.docker.ListContainers(m.ctx, true)
	if err != nil {
		return fullErrMsg{err}
	}
	return fullContainersMsg{containers}
}

// fetchImages fetches image data from Docker
func (m FullModel) fetchImages() tea.Msg {
	m.statusMsg = "Fetching images..."
	images, err := m.docker.ListImages(m.ctx)
	if err != nil {
		return fullErrMsg{err}
	}
	return fullImagesMsg{images}
}

// fetchVolumes fetches volume data from Docker
func (m FullModel) fetchVolumes() tea.Msg {
	m.statusMsg = "Fetching volumes..."
	volumes, err := m.docker.ListVolumes(m.ctx)
	if err != nil {
		return fullErrMsg{err}
	}
	return fullVolumesMsg{volumes}
}

// fetchNetworks fetches network data from Docker
func (m FullModel) fetchNetworks() tea.Msg {
	m.statusMsg = "Fetching networks..."
	networks, err := m.docker.ListNetworks(m.ctx)
	if err != nil {
		return fullErrMsg{err}
	}
	return fullNetworksMsg{networks}
}

// fetchComposeProjects fetches Docker Compose projects
func (m FullModel) fetchComposeProjects() tea.Msg {
	m.statusMsg = "Fetching Docker Compose projects..."
	// Don't make the call if we're not connected to Docker
	if !m.dockerConnected {
		return composeProjectsMsg{projects: []docker.ComposeInfo{}}
	}

	projects, err := m.docker.ListComposeProjects(m.ctx)
	if err != nil {
		return fullErrMsg{err}
	}

	return composeProjectsMsg{projects: projects}
}

// fetchLogs fetches logs for a container
func (m FullModel) fetchLogs() tea.Msg {
	if m.selectedID == "" {
		return fullLogsMsg{"No container selected"}
	}
	m.statusMsg = "Fetching logs..."
	logs, err := m.docker.GetContainerLogs(m.ctx, m.selectedID)
	if err != nil {
		return fullErrMsg{err}
	}
	return fullLogsMsg{logs}
}

// fetchStats fetches monitoring statistics for a container
func (m FullModel) fetchStats() tea.Msg {
	if m.selectedID == "" {
		return fullStatsMsg{"No container selected"}
	}

	m.statusMsg = "Fetching container stats..."
	stats, err := m.docker.GetProcessedStats(m.ctx, m.selectedID)
	if err != nil {
		return fullErrMsg{err}
	}

	var sb strings.Builder

	// Format CPU usage with bar
	cpuBar := createUsageBar(stats.CPUPercentage, 50)

	// Format memory usage with bar
	memBar := createUsageBar(stats.MemoryPercentage, 50)

	// Create header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#5f87ff"))

	// CPU section
	sb.WriteString(headerStyle.Render("CPU Usage:"))
	sb.WriteString(fmt.Sprintf("\n%.2f%%\n", stats.CPUPercentage))
	sb.WriteString(cpuBar)
	sb.WriteString("\n\n")

	// Memory section
	sb.WriteString(headerStyle.Render("Memory Usage:"))
	sb.WriteString(fmt.Sprintf("\n%.2f%% (%s / %s)\n",
		stats.MemoryPercentage,
		formatBytes(stats.MemoryUsage),
		formatBytes(stats.MemoryLimit)))
	sb.WriteString(memBar)
	sb.WriteString("\n\n")

	// Network I/O
	sb.WriteString(headerStyle.Render("Network I/O:"))
	sb.WriteString(fmt.Sprintf("\n📥 RX: %s / 📤 TX: %s\n\n",
		formatBytes(stats.NetworkRx),
		formatBytes(stats.NetworkTx)))

	// Block I/O
	sb.WriteString(headerStyle.Render("Block I/O:"))
	sb.WriteString(fmt.Sprintf("\n📄 Read: %s / 📝 Write: %s\n",
		formatBytes(stats.BlockRead),
		formatBytes(stats.BlockWrite)))

	return fullStatsMsg{sb.String()}
}

// createUsageBar creates a text-based usage bar
func createUsageBar(percentage float64, width int) string {
	filled := int((percentage / 100.0) * float64(width))
	if filled > width {
		filled = width
	}

	// Choose color based on usage
	var barColor lipgloss.Color
	var icon string
	if percentage < 60 {
		barColor = lipgloss.Color("#4CAF50") // Green
		icon = "🟩 "
	} else if percentage < 85 {
		barColor = lipgloss.Color("#FFC107") // Yellow
		icon = "🟨 "
	} else {
		barColor = lipgloss.Color("#F44336") // Red
		icon = "🟥 "
	}

	// Create filled and empty segments with proper styling
	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	filledBar := filledStyle.Render(strings.Repeat("█", filled))
	emptyBar := emptyStyle.Render(strings.Repeat("░", width-filled))

	// Combine segments with percentage
	return fmt.Sprintf("%s%s%s [%.1f%%]",
		icon,
		filledBar,
		emptyBar,
		percentage)
}

// startStatsRefresh starts a ticker to refresh container stats
func (m FullModel) startStatsRefresh() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// stopStatsRefresh stops the stats refresh ticker
func (m FullModel) stopStatsRefresh() tea.Cmd {
	return tea.Batch()
}

// inspectResource fetches details for a resource
func (m FullModel) inspectResource() tea.Msg {
	if m.selectedID == "" {
		return fullInspectMsg{"No resource selected"}
	}

	m.statusMsg = "Inspecting resource..."
	var details string
	var err error

	switch m.currentTab {
	case ContainersTab:
		details, err = m.docker.InspectContainer(m.ctx, m.selectedID)
	case ImagesTab:
		details, err = m.docker.InspectImage(m.ctx, m.selectedID)
	case VolumesTab:
		details, err = m.docker.InspectVolume(m.ctx, m.selectedID)
	case NetworksTab:
		details, err = m.docker.InspectNetwork(m.ctx, m.selectedID)
	}

	if err != nil {
		return fullErrMsg{err}
	}
	return fullInspectMsg{details}
}

// inspectComposeProject fetches details for a Docker Compose project
func (m *FullModel) inspectComposeProject() tea.Msg {
	if m.selectedPath == "" {
		// Try to find the path from the compose projects list
		for _, project := range m.composeProjects {
			if project.Name == m.selectedName {
				m.selectedPath = project.Path
				break
			}
		}

		// If we still don't have a path, return an error
		if m.selectedPath == "" {
			return fullInspectMsg{fmt.Sprintf("No Docker Compose project path found for %s.\nPlease refresh the projects list and try again.",
				m.selectedName)}
		}
	}

	m.statusMsg = fmt.Sprintf("Inspecting Docker Compose project: %s at %s", m.selectedName, m.selectedPath)
	m.composeServicesLoading = true

	return tea.Batch(
		func() tea.Msg {
			return fullInspectMsg{fmt.Sprintf("Loading services for %s at %s...", m.selectedName, m.selectedPath)}
		},
		m.fetchComposeServices,
		m.fetchComposeContainers,
	)
}

// fetchComposeServices fetches Docker Compose services for a project
func (m FullModel) fetchComposeServices() tea.Msg {
	if m.selectedPath == "" {
		return fullComposeServicesMsg{
			services:    []docker.ComposeServiceInfo{},
			projectName: m.selectedName,
			error:       fmt.Errorf("no project path available for %s", m.selectedName),
		}
	}

	// Check if the path exists before trying to use it
	if _, err := os.Stat(m.selectedPath); os.IsNotExist(err) {
		return fullComposeServicesMsg{
			services:    []docker.ComposeServiceInfo{},
			projectName: m.selectedName,
			error:       fmt.Errorf("project path does not exist: %s", m.selectedPath),
		}
	}

	m.statusMsg = fmt.Sprintf("Fetching services for %s at %s...", m.selectedName, m.selectedPath)

	// Add timeout to the context to prevent hanging
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	// Now try to list services
	services, err := m.docker.ListComposeServices(ctx, m.selectedPath)
	if err != nil {
		errMsg := err.Error()
		// Try to provide more user-friendly error messages based on common errors
		if strings.Contains(errMsg, "no compose file found") {
			errMsg = fmt.Sprintf("No docker-compose.yml or compose.yaml file found in %s", m.selectedPath)
		} else if strings.Contains(errMsg, "failed to parse compose file") {
			errMsg = fmt.Sprintf("The compose file in %s has invalid syntax", m.selectedPath)
		} else if strings.Contains(errMsg, "no services found") {
			errMsg = fmt.Sprintf("No services found in the compose file in %s. Check if it has a 'services:' section.", m.selectedPath)
		}

		return fullComposeServicesMsg{
			services:    []docker.ComposeServiceInfo{},
			projectName: m.selectedName,
			error:       fmt.Errorf("%s", errMsg),
		}
	}

	if len(services) == 0 {
		// Return an error message that's more user-friendly
		return fullComposeServicesMsg{
			services:    []docker.ComposeServiceInfo{},
			projectName: m.selectedName,
			error:       fmt.Errorf("no services defined in the compose file for %s", m.selectedName),
		}
	}

	return fullComposeServicesMsg{
		services:    services,
		projectName: m.selectedName,
	}
}

// composeAction performs an action on a Docker Compose project
func (m FullModel) composeAction(action string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedPath == "" {
			return fullActionResultMsg{success: false, message: "No Docker Compose project selected"}
		}

		m.statusMsg = fmt.Sprintf("Performing %s on %s...", action, m.selectedName)
		var err error

		switch action {
		case "up":
			err = m.docker.ComposeUp(m.ctx, m.selectedPath)
		case "down":
			err = m.docker.ComposeDown(m.ctx, m.selectedPath)
		case "pull":
			err = m.docker.ComposePull(m.ctx, m.selectedPath)
		case "logs":
			// For logs, we need to fetch and format them
			logs, logErr := m.docker.ComposeLogs(m.ctx, m.selectedPath)
			if logErr != nil {
				err = logErr
			} else {
				return fullLogsMsg{logs}
			}
		}

		if err != nil {
			return fullActionResultMsg{success: false, message: err.Error()}
		}

		return fullActionResultMsg{
			success: true,
			message: fmt.Sprintf("Successfully performed %s on %s", action, m.selectedName),
			action:  action,
		}
	}
}

// containerAction performs an action on a container
func (m FullModel) containerAction(action string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedID == "" {
			return fullActionResultMsg{success: false, message: "No container selected"}
		}

		m.statusMsg = fmt.Sprintf("Performing %s on %s...", action, m.selectedName)
		var err error

		switch action {
		case "start":
			err = m.docker.StartContainer(m.ctx, m.selectedID)
		case "stop":
			err = m.docker.StopContainer(m.ctx, m.selectedID)
		case "restart":
			err = m.docker.RestartContainer(m.ctx, m.selectedID)
		case "pause":
			err = m.docker.PauseContainer(m.ctx, m.selectedID)
		case "unpause":
			err = m.docker.UnpauseContainer(m.ctx, m.selectedID)
		case "kill":
			err = m.docker.KillContainer(m.ctx, m.selectedID)
		case "remove":
			err = m.docker.RemoveContainer(m.ctx, m.selectedID)
		}

		if err != nil {
			return fullActionResultMsg{success: false, message: err.Error()}
		}

		return fullActionResultMsg{
			success: true,
			message: fmt.Sprintf("Successfully performed %s on %s", action, m.selectedName),
			action:  action,
		}
	}
}

// imageAction performs an action on an image
func (m FullModel) imageAction(action string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedID == "" {
			return fullActionResultMsg{success: false, message: "No image selected"}
		}

		m.statusMsg = fmt.Sprintf("Performing %s on %s...", action, m.selectedName)
		var err error

		switch action {
		case "remove":
			err = m.docker.RemoveImage(m.ctx, m.selectedID, true)
		}

		if err != nil {
			return fullActionResultMsg{success: false, message: err.Error()}
		}

		return fullActionResultMsg{
			success: true,
			message: fmt.Sprintf("Successfully performed %s on %s", action, m.selectedName),
			action:  action,
		}
	}
}

// volumeAction performs an action on a volume
func (m FullModel) volumeAction(action string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedID == "" {
			return fullActionResultMsg{success: false, message: "No volume selected"}
		}

		m.statusMsg = fmt.Sprintf("Performing %s on %s...", action, m.selectedName)
		var err error

		switch action {
		case "remove":
			err = m.docker.RemoveVolume(m.ctx, m.selectedID, true)
		}

		if err != nil {
			return fullActionResultMsg{success: false, message: err.Error()}
		}

		return fullActionResultMsg{
			success: true,
			message: fmt.Sprintf("Successfully performed %s on %s", action, m.selectedName),
			action:  action,
		}
	}
}

// networkAction performs an action on a network
func (m FullModel) networkAction(action string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedID == "" {
			return fullActionResultMsg{success: false, message: "No network selected"}
		}

		m.statusMsg = fmt.Sprintf("Performing %s on %s...", action, m.selectedName)
		var err error

		switch action {
		case "remove":
			err = m.docker.RemoveNetwork(m.ctx, m.selectedID)
		}

		if err != nil {
			return fullActionResultMsg{success: false, message: err.Error()}
		}

		return fullActionResultMsg{
			success: true,
			message: fmt.Sprintf("Successfully performed %s on %s", action, m.selectedName),
			action:  action,
		}
	}
}

// initializeTable creates a table for a specific resource type
func (m *FullModel) initializeTable(resourceType Tab) table.Model {
	var columns []table.Column

	switch resourceType {
	case ContainersTab:
		columns = []table.Column{
			{Title: "NAME", Width: 20},
			{Title: "STATUS", Width: 15},
			{Title: "IMAGE", Width: 30},
			{Title: "ID", Width: 15},
		}
	case ImagesTab:
		columns = []table.Column{
			{Title: "REPOSITORY", Width: 40},
			{Title: "SIZE", Width: 15},
			{Title: "ID", Width: 20},
		}
	case VolumesTab:
		columns = []table.Column{
			{Title: "NAME", Width: 30},
			{Title: "DRIVER", Width: 15},
			{Title: "MOUNTPOINT", Width: 35},
		}
	case NetworksTab:
		columns = []table.Column{
			{Title: "NAME", Width: 30},
			{Title: "DRIVER", Width: 15},
			{Title: "SCOPE", Width: 15},
			{Title: "ID", Width: 20},
		}
	case ComposeTab:
		columns = []table.Column{
			{Title: "NAME", Width: 25},
			{Title: "STATUS", Width: 15},
			{Title: "PATH", Width: 40},
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(m.height-12),
		table.WithWidth(m.width),
		table.WithFocused(true),
	)

	// Set table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	t.SetStyles(s)

	return t
}

// updateTables updates dimensions for all tables
func (m *FullModel) updateTables() {
	height := m.height - 12 // Adjust for header, footer, etc.

	if m.containerTable.Height() != height {
		m.containerTable.SetHeight(height)
		m.containerTable.SetWidth(m.width)
	}

	if m.imageTable.Height() != height {
		m.imageTable.SetHeight(height)
		m.imageTable.SetWidth(m.width)
	}

	if m.volumeTable.Height() != height {
		m.volumeTable.SetHeight(height)
		m.volumeTable.SetWidth(m.width)
	}

	if m.networkTable.Height() != height {
		m.networkTable.SetHeight(height)
		m.networkTable.SetWidth(m.width)
	}

	if m.composeTable.Height() != height {
		m.composeTable.SetHeight(height)
		m.composeTable.SetWidth(m.width)
	}

	// Set viewport height based on current mode
	var viewportHeight int
	if m.currentMode == InspectMode {
		// Less height to accommodate action panel
		viewportHeight = m.height - 16
	} else {
		// Normal height for logs and monitor modes
		viewportHeight = m.height - 8
	}

	if m.viewport.Height != viewportHeight {
		m.viewport.Height = viewportHeight
		m.viewport.Width = m.width
	}
}

// getCurrentTable returns the currently active table based on the active tab
func (m *FullModel) getCurrentTable() *table.Model {
	switch m.currentTab {
	case ContainersTab:
		return &m.containerTable
	case ImagesTab:
		return &m.imageTable
	case VolumesTab:
		return &m.volumeTable
	case NetworksTab:
		return &m.networkTable
	case ComposeTab:
		return &m.composeTable
	default:
		return &m.containerTable
	}
}

// updateSelection updates the selected resource based on the current table cursor
func (m *FullModel) updateSelection() {
	table := m.getCurrentTable()
	selectedRow := table.SelectedRow()

	if len(selectedRow) == 0 {
		m.selectedID = ""
		m.selectedName = ""
		m.selectedPath = ""
		return
	}

	switch m.currentTab {
	case ContainersTab:
		if len(m.containers) > 0 && table.Cursor() < len(m.containers) {
			m.selectedID = m.containers[table.Cursor()].ID
			m.selectedName = m.containers[table.Cursor()].Name
		}

	case ImagesTab:
		if len(m.images) > 0 && table.Cursor() < len(m.images) {
			m.selectedID = m.images[table.Cursor()].ID
			m.selectedName = ""
			if len(m.images[table.Cursor()].RepoTags) > 0 {
				m.selectedName = m.images[table.Cursor()].RepoTags[0]
			}
		}

	case VolumesTab:
		if len(m.volumes) > 0 && table.Cursor() < len(m.volumes) {
			m.selectedID = m.volumes[table.Cursor()].Name
			m.selectedName = m.volumes[table.Cursor()].Name
		}

	case NetworksTab:
		if len(m.networks) > 0 && table.Cursor() < len(m.networks) {
			m.selectedID = m.networks[table.Cursor()].ID
			m.selectedName = m.networks[table.Cursor()].Name
		}

	case ComposeTab:
		if len(m.composeProjects) > 0 && table.Cursor() < len(m.composeProjects) {
			cursorIndex := table.Cursor()
			if cursorIndex >= len(m.composeProjects) {
				// Stay safe
				cursorIndex = 0
			}

			selectedProject := m.composeProjects[cursorIndex]
			m.selectedID = selectedProject.Name
			m.selectedName = selectedProject.Name
			m.selectedPath = selectedProject.Path

			// If path is empty, try to search for it by name
			if m.selectedPath == "" && m.selectedID != "" {
				for _, p := range m.composeProjects {
					if p.Name == m.selectedID {
						m.selectedPath = p.Path
						m.statusMsg = fmt.Sprintf("Found project path: %s", m.selectedPath)
						break
					}
				}

				// If still no path, check if there are any projects with paths at all
				if m.selectedPath == "" {
					for _, p := range m.composeProjects {
						if p.Path != "" {
							m.selectedPath = p.Path
							m.statusMsg = fmt.Sprintf("Using fallback path from project %s: %s", p.Name, p.Path)
							break
						}
					}
				}
			}
		}
	}
}

// Update handles updates to the model
func (m FullModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global key bindings
		switch {
		case key.Matches(msg, DefaultFullKeyMap.Quit):
			m.statusMsg = "Quitting..."
			return m, tea.Quit

		case key.Matches(msg, DefaultFullKeyMap.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, DefaultFullKeyMap.Refresh):
			if m.currentMode == MonitorMode {
				return m, m.fetchStats
			}

			if m.currentMode == InspectMode {
				// Refresh the inspection
				if m.currentTab == ComposeTab {
					return m, m.inspectComposeProject
				}
				return m, m.inspectResource
			}

			m.statusMsg = "Refreshing..."
			return m, tea.Batch(
				m.fetchContainers,
				m.fetchImages,
				m.fetchVolumes,
				m.fetchNetworks,
				m.fetchComposeProjects,
			)

		case key.Matches(msg, DefaultFullKeyMap.NextTab):
			if m.currentMode == ListMode {
				prevTab := m.currentTab
				m.currentTab = (m.currentTab + 1) % 5 // Cycle through the 5 tabs (including Compose)

				// If we're switching to a different tab, ensure data is refreshed
				if prevTab != m.currentTab {
					switch m.currentTab {
					case ContainersTab:
						return m, m.fetchContainers
					case ImagesTab:
						return m, m.fetchImages
					case VolumesTab:
						return m, m.fetchVolumes
					case NetworksTab:
						return m, m.fetchNetworks
					case ComposeTab:
						return m, m.fetchComposeProjects
					}
				}

				// If switching to Compose tab, refresh compose projects
				if m.currentTab == ComposeTab {
					var cmds []tea.Cmd
					// Always refresh projects
					cmds = append(cmds, func() tea.Msg {
						return m.fetchComposeProjects()
					})

					// If in inspect mode with a selected project, fetch services too
					if m.currentMode == InspectMode && m.selectedPath != "" {
						cmds = append(cmds, func() tea.Msg {
							return m.fetchComposeServices()
						})
					}

					if len(cmds) > 0 {
						return m, tea.Batch(cmds...)
					}
				}

				return m, nil
			}

		case key.Matches(msg, DefaultFullKeyMap.PrevTab):
			if m.currentMode == ListMode {
				prevTab := m.currentTab
				m.currentTab = (m.currentTab - 1 + 5) % 5 // Cycle through the 5 tabs (including Compose)

				// If we're switching to a different tab, ensure data is refreshed
				if prevTab != m.currentTab {
					switch m.currentTab {
					case ContainersTab:
						return m, m.fetchContainers
					case ImagesTab:
						return m, m.fetchImages
					case VolumesTab:
						return m, m.fetchVolumes
					case NetworksTab:
						return m, m.fetchNetworks
					case ComposeTab:
						return m, m.fetchComposeProjects
					}
				}

				// If switching to Compose tab, refresh compose projects
				if m.currentTab == ComposeTab {
					cmds = append(cmds, func() tea.Msg {
						return m.fetchComposeProjects()
					})
				}

				return m, nil
			}

		case key.Matches(msg, DefaultFullKeyMap.Back):
			if m.currentMode == MonitorMode {
				// Stop stats refresh when leaving monitor mode
				m.currentMode = ListMode
				return m, m.stopStatsRefresh()
			}
			if m.currentMode != ListMode {
				m.currentMode = ListMode
				return m, nil
			}
		}

		// Handle action keys in ListMode
		if m.currentMode == ListMode {
			// Update selection before performing actions
			m.updateSelection()

			// Process ComposeTab actions first if we're in ComposeTab to avoid conflicts with 'd' key
			if m.currentTab == ComposeTab {
				switch {
				case key.Matches(msg, DefaultFullKeyMap.ComposeUp):
					return m, m.composeAction("up")
				case key.Matches(msg, DefaultFullKeyMap.ComposeDown):
					return m, m.composeAction("down")
				case key.Matches(msg, DefaultFullKeyMap.ComposePull):
					return m, m.composeAction("pull")
				}
			}

			// Process shared actions for all tabs
			switch {
			case key.Matches(msg, DefaultFullKeyMap.Inspect):
				if m.selectedID != "" {
					m.currentMode = InspectMode
					if m.currentTab == ComposeTab {
						// Force update selection to ensure selectedPath is set properly
						m.updateSelection()

						// If path is still empty despite having a selected ID, try to find it in all projects
						if m.selectedPath == "" && m.selectedID != "" && len(m.composeProjects) > 0 {
							// Look for any project with matching name
							for _, p := range m.composeProjects {
								if p.Name == m.selectedID || p.Name == m.selectedName {
									m.selectedPath = p.Path
									m.statusMsg = fmt.Sprintf("Found project path: %s", m.selectedPath)
									break
								}
							}

							// If still no path, check if there are any projects with paths at all
							if m.selectedPath == "" {
								for _, p := range m.composeProjects {
									if p.Path != "" {
										m.selectedPath = p.Path
										m.statusMsg = fmt.Sprintf("Using fallback path from project %s: %s", p.Name, m.selectedPath)
										break
									}
								}
							}
						}

						// Set the viewport content directly for immediate display
						content := m.renderComposeInspect()
						m.viewport.SetContent(content)
						m.viewport.GotoTop()

						// Then fetch services async
						return m, m.inspectComposeProject
					}
					return m, m.inspectResource
				}

			case key.Matches(msg, DefaultFullKeyMap.Logs):
				// Containers and Compose projects have logs
				if m.currentTab == ContainersTab && m.selectedID != "" {
					m.currentMode = LogsMode
					return m, m.fetchLogs
				} else if m.currentTab == ComposeTab && m.selectedPath != "" {
					m.currentMode = LogsMode
					return m, m.composeAction("logs")
				}

			case key.Matches(msg, DefaultFullKeyMap.Monitor):
				// Only containers can be monitored
				if m.currentTab == ContainersTab && m.selectedID != "" {
					m.currentMode = MonitorMode
					return m, tea.Batch(
						m.fetchStats,
						m.startStatsRefresh(),
					)
				}
			}

			// Handle tab-specific actions based on current tab
			switch m.currentTab {
			case ContainersTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Start):
					return m, m.containerAction("start")
				case key.Matches(msg, DefaultFullKeyMap.Stop):
					return m, m.containerAction("stop")
				case key.Matches(msg, DefaultFullKeyMap.Restart):
					return m, m.containerAction("restart")
				case key.Matches(msg, DefaultFullKeyMap.Pause):
					return m, m.containerAction("pause")
				case key.Matches(msg, DefaultFullKeyMap.Resume):
					return m, m.containerAction("unpause")
				case key.Matches(msg, DefaultFullKeyMap.Kill):
					return m, m.containerAction("kill")
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					return m, m.containerAction("remove")
				}
			case ImagesTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					return m, m.imageAction("remove")
				}
			case VolumesTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					return m, m.volumeAction("remove")
				}
			case NetworksTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					return m, m.networkAction("remove")
				}
			}

			// Handle navigation keys for tables
			table := m.getCurrentTable()
			if table.Width() > 0 {
				*table, cmd = table.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else if m.currentMode == InspectMode {
			// Similar approach in inspect mode: handle ComposeTab actions first if applicable
			if m.currentTab == ComposeTab {
				// Add container selection feature
				if msg.String() == "c" {
					m.statusMsg = "Enter container number (1-9):"
					return m, nil
				}

				// Check for number keys 1-9 after pressing 'c'
				if m.statusMsg == "Enter container number (1-9):" {
					numStr := msg.String()
					if numStr >= "1" && numStr <= "9" {
						num, err := strconv.Atoi(numStr)
						if err == nil && num >= 1 && num <= 9 && num <= len(m.composeContainers) {
							// Get the container ID
							selectedID := m.composeContainers[num-1].ID

							// Store the container name for better user feedback
							selectedName := m.composeContainers[num-1].Name

							// Clear the status message and provide feedback
							m.statusMsg = fmt.Sprintf("Switching to container: %s", selectedName)

							// Jump to the container tab with that container selected
							m.jumpToContainer(selectedID)

							return m, nil
						} else if err == nil && num >= 1 && num <= 9 {
							// Invalid container number
							m.statusMsg = fmt.Sprintf("Container %d not found. Valid range: 1-%d",
								num, len(m.composeContainers))
							return m, nil
						}
					}

					// Invalid input - clear status and show message
					m.statusMsg = "Invalid container number. Cancelled selection."
				}

				// Continue with existing compose actions
				switch {
				case key.Matches(msg, DefaultFullKeyMap.ComposeUp):
					m.statusMsg = "Starting Docker Compose project..."
					return m, tea.Batch(
						m.composeAction("up"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.ComposeDown):
					m.statusMsg = "Stopping Docker Compose project..."
					return m, tea.Batch(
						m.composeAction("down"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.ComposePull):
					m.statusMsg = "Pulling Docker Compose images..."
					return m, tea.Batch(
						m.composeAction("pull"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				}
			}

			// Shared actions in inspect mode
			switch {
			case key.Matches(msg, DefaultFullKeyMap.Logs):
				// Containers and Compose projects have logs
				if m.currentTab == ContainersTab && m.selectedID != "" {
					m.currentMode = LogsMode
					return m, m.fetchLogs
				} else if m.currentTab == ComposeTab && m.selectedPath != "" {
					m.currentMode = LogsMode
					return m, m.composeAction("logs")
				}

			case key.Matches(msg, DefaultFullKeyMap.Monitor):
				// Only containers can be monitored
				if m.currentTab == ContainersTab && m.selectedID != "" {
					m.currentMode = MonitorMode
					return m, tea.Batch(
						m.fetchStats,
						m.startStatsRefresh(),
					)
				}
			}

			// Handle tab-specific actions in inspect mode
			switch m.currentTab {
			case ContainersTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Start):
					m.statusMsg = "Starting container..."
					return m, tea.Batch(
						m.containerAction("start"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.Stop):
					m.statusMsg = "Stopping container..."
					return m, tea.Batch(
						m.containerAction("stop"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.Restart):
					m.statusMsg = "Restarting container..."
					return m, tea.Batch(
						m.containerAction("restart"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.Pause):
					m.statusMsg = "Pausing container..."
					return m, tea.Batch(
						m.containerAction("pause"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.Resume):
					m.statusMsg = "Unpausing container..."
					return m, tea.Batch(
						m.containerAction("unpause"),
						func() tea.Msg {
							return afterActionMsg{action: "inspect"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.Kill):
					m.statusMsg = "Killing container..."
					return m, tea.Batch(
						m.containerAction("kill"),
						func() tea.Msg {
							return afterActionMsg{action: "list"}
						},
					)
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					m.statusMsg = "Removing container..."
					return m, tea.Batch(
						m.containerAction("remove"),
						func() tea.Msg {
							return afterActionMsg{action: "list"}
						},
					)
				}
			case ImagesTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					m.statusMsg = "Removing image..."
					return m, tea.Batch(
						m.imageAction("remove"),
						func() tea.Msg {
							return afterActionMsg{action: "list"}
						},
					)
				}
			case VolumesTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					m.statusMsg = "Removing volume..."
					return m, tea.Batch(
						m.volumeAction("remove"),
						func() tea.Msg {
							return afterActionMsg{action: "list"}
						},
					)
				}
			case NetworksTab:
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Remove):
					m.statusMsg = "Removing network..."
					return m, tea.Batch(
						m.networkAction("remove"),
						func() tea.Msg {
							return afterActionMsg{action: "list"}
						},
					)
				}
			}

			// When in inspect mode, let the viewport handle navigation
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else if m.currentMode == LogsMode || m.currentMode == MonitorMode {
			// Additional key handling for monitor mode
			if m.currentMode == MonitorMode {
				switch {
				case key.Matches(msg, DefaultFullKeyMap.Refresh):
					return m, m.fetchStats
				}
			}

			// When in logs or monitor mode, let the viewport handle navigation
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tickMsg:
		// Only refresh stats if we're in monitor mode
		if m.currentMode == MonitorMode {
			cmds = append(cmds, m.fetchStats)
			cmds = append(cmds, m.startStatsRefresh())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Initialize tables if this is the first resize
		if m.containerTable.Width() == 0 {
			m.containerTable = m.initializeTable(ContainersTab)
			m.imageTable = m.initializeTable(ImagesTab)
			m.volumeTable = m.initializeTable(VolumesTab)
			m.networkTable = m.initializeTable(NetworksTab)
			m.composeTable = m.initializeTable(ComposeTab)

			// Set up viewport for details panel
			m.viewport = viewport.New(msg.Width, msg.Height-8)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)

		} else {
			m.updateTables()
		}

	case fullContainersMsg:
		m.loading = false
		m.containers = msg.containers

		// Convert containers to table rows
		rows := []table.Row{}
		for _, c := range msg.containers {
			// Add status icon based on container state
			statusWithIcon := c.State
			switch {
			case strings.Contains(strings.ToLower(c.State), "running"):
				statusWithIcon = IconRunning + c.State
			case strings.Contains(strings.ToLower(c.State), "exited"):
				statusWithIcon = IconExited + c.State
			case strings.Contains(strings.ToLower(c.State), "created"):
				statusWithIcon = IconCreated + c.State
			case strings.Contains(strings.ToLower(c.State), "paused"):
				statusWithIcon = IconPaused + c.State
			case strings.Contains(strings.ToLower(c.State), "restarting"):
				statusWithIcon = IconRestarting + c.State
			case strings.Contains(strings.ToLower(c.State), "dead"):
				statusWithIcon = IconDead + c.State
			}

			row := table.Row{c.Name, statusWithIcon, c.Image, c.ID[:12]}
			rows = append(rows, row)
		}

		m.containerTable.SetRows(rows)
		m.statusMsg = fmt.Sprintf("Loaded %d containers", len(msg.containers))

	case fullImagesMsg:
		m.loading = false
		m.images = msg.images

		// Convert images to table rows
		rows := []table.Row{}
		for _, img := range msg.images {
			repoTag := "<none>:<none>"
			if len(img.RepoTags) > 0 {
				repoTag = img.RepoTags[0]
			}

			// Format size
			size := formatBytes(img.Size)

			row := table.Row{repoTag, size, img.ID[:12]}
			rows = append(rows, row)
		}

		m.imageTable.SetRows(rows)
		m.statusMsg = fmt.Sprintf("Loaded %d images", len(msg.images))

	case fullVolumesMsg:
		m.loading = false
		m.volumes = msg.volumes

		// Convert volumes to table rows
		rows := []table.Row{}
		for _, v := range msg.volumes {
			row := table.Row{v.Name, v.Driver, v.Mountpoint}
			rows = append(rows, row)
		}

		m.volumeTable.SetRows(rows)
		m.statusMsg = fmt.Sprintf("Loaded %d volumes", len(msg.volumes))

	case fullNetworksMsg:
		m.loading = false
		m.networks = msg.networks

		// Convert networks to table rows
		rows := []table.Row{}
		for _, n := range msg.networks {
			row := table.Row{n.Name, n.Driver, n.Scope, n.ID[:12]}
			rows = append(rows, row)
		}

		m.networkTable.SetRows(rows)
		m.statusMsg = fmt.Sprintf("Loaded %d networks", len(msg.networks))

	case fullLogsMsg:
		m.logContent = msg.content
		m.viewport.SetContent(m.logContent)
		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("Showing logs for %s", m.selectedName)

	case fullInspectMsg:
		m.inspectContent = msg.content

		// Special handling for Compose tab
		if m.currentTab == ComposeTab && m.currentMode == InspectMode {
			// Use our custom compose inspection renderer instead of the generic content
			content := m.renderComposeInspect()
			m.viewport.SetContent(content)
		} else {
			// Normal handling for other tabs
			m.viewport.SetContent(m.inspectContent)
		}

		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("Inspecting %s", m.selectedName)

	case fullActionResultMsg:
		m.statusMsg = msg.message
		if msg.success && msg.action != "" {
			// Refresh data after successful action
			switch m.currentTab {
			case ContainersTab:
				return m, m.fetchContainers
			case ImagesTab:
				return m, m.fetchImages
			case VolumesTab:
				return m, m.fetchVolumes
			case NetworksTab:
				return m, m.fetchNetworks
			case ComposeTab:
				return m, m.fetchComposeProjects
			default:
				return m, m.fetchContainers
			}
		}

	case fullErrMsg:
		m.loading = false
		m.err = msg.err
		m.statusMsg = fmt.Sprintf("Error: %v", msg.err)

	case fullStatsMsg:
		m.statsContent = msg.content
		m.viewport.SetContent(m.statsContent)
		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("Monitoring %s", m.selectedName)

	case dockerConnectionMsg:
		m.dockerConnected = msg.connected
		if !m.dockerConnected {
			m.statusMsg = fmt.Sprintf("Docker connection error: %v", msg.err)
			// Start periodic check for reconnection
			return m, m.startConnectionCheck()
		}
		return m, nil

	case connectionCheckTickMsg:
		// Time to check the connection again
		return m, m.checkDockerConnection

	case afterActionMsg:
		// Handle actions after a container action completes
		if msg.action == "inspect" {
			// Stay in inspect mode and refresh the inspection
			if m.currentTab == ComposeTab {
				return m, m.inspectComposeProject
			}
			return m, m.inspectResource
		} else if msg.action == "list" {
			// Return to list mode
			m.currentMode = ListMode
			// Refresh the resources
			if m.currentTab == ComposeTab {
				return m, m.fetchComposeProjects
			}
			return m, m.fetchContainers
		}

	case composeProjectsMsg:
		m.loading = false
		m.composeProjects = msg.projects

		// Convert compose projects to table rows
		rows := []table.Row{}
		for _, p := range msg.projects {
			row := table.Row{p.Name, p.Status, p.Path}
			rows = append(rows, row)
		}

		m.composeTable.SetRows(rows)
		m.statusMsg = fmt.Sprintf("Loaded %d Docker Compose projects", len(msg.projects))

	case fullComposeServicesMsg:
		m.composeServicesLoading = false
		m.composeServices = msg.services
		if msg.error != nil {
			// Show error in the status bar
			m.statusMsg = fmt.Sprintf("Error: %v", msg.error)
		} else {
			m.statusMsg = fmt.Sprintf("Found %d services for %s", len(msg.services), msg.projectName)
		}

		// Update the viewport with the new content
		if m.currentMode == InspectMode && m.currentTab == ComposeTab {
			// Re-render the content with the updated services
			content := m.renderComposeInspect()
			m.viewport.SetContent(content)

			// Preserve scroll position if possible, or go to top if new content
			if len(m.composeServices) > 0 {
				// Keep current position if just updating content
				currentY := m.viewport.YOffset
				m.viewport.SetYOffset(currentY)
			} else {
				// Go to top if first time loading
				m.viewport.GotoTop()
			}
		}

		return m, nil

	case fullComposeContainersMsg:
		m.composeContainersLoading = false
		m.composeContainers = msg.containers
		if msg.error != nil {
			// Show error in the status bar
			m.statusMsg = fmt.Sprintf("Error fetching containers: %v", msg.error)
		} else {
			m.statusMsg = fmt.Sprintf("Found %d containers for %s", len(msg.containers), msg.projectName)
		}

		// Update the viewport with the new content
		if m.currentMode == InspectMode && m.currentTab == ComposeTab {
			// Re-render the content with the updated containers
			content := m.renderComposeInspect()
			m.viewport.SetContent(content)

			// Preserve scroll position if possible
			currentY := m.viewport.YOffset
			m.viewport.SetYOffset(currentY)
		}

		return m, nil

	}

	// Apply any pending commands
	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}

	return m, cmd
}

// View renders the UI
func (m FullModel) View() string {
	var sb strings.Builder

	// Create a header with tabs
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#88c0d0")).
		Render("Docker Tea")

	// Tab bar
	tabBar := m.renderTabBar()

	sb.WriteString(header)
	sb.WriteString("  ")
	sb.WriteString(tabBar)
	sb.WriteString("\n\n")

	// Show Docker connection alert if not connected
	if !m.dockerConnected {
		alertStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#ff0000")).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1).
			Width(m.width - 4)

		sb.WriteString(alertStyle.Render(fmt.Sprintf("%s ALERT: Docker is not running or not responding! %s", IconError, IconError)))
		sb.WriteString("\n\n")
	}

	// Main content area
	switch m.currentMode {
	case ListMode:
		// Render the appropriate table based on the current tab
		switch m.currentTab {
		case ContainersTab:
			if m.loading && m.containerTable.Width() == 0 {
				sb.WriteString("Loading containers...\n")
			} else {
				sb.WriteString(m.containerTable.View())
			}
		case ImagesTab:
			if m.loading && m.imageTable.Width() == 0 {
				sb.WriteString("Loading images...\n")
			} else {
				sb.WriteString(m.imageTable.View())
			}
		case VolumesTab:
			if m.loading && m.volumeTable.Width() == 0 {
				sb.WriteString("Loading volumes...\n")
			} else {
				sb.WriteString(m.volumeTable.View())
			}
		case NetworksTab:
			if m.loading && m.networkTable.Width() == 0 {
				sb.WriteString("Loading networks...\n")
			} else {
				sb.WriteString(m.networkTable.View())
			}
		case ComposeTab:
			sb.WriteString(m.renderComposeTab())
		}
	case InspectMode:
		// Render inspect view
		inspectHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#88c0d0")).
			Render(fmt.Sprintf("Inspecting %s", m.selectedName))

		sb.WriteString(inspectHeader)
		sb.WriteString("\n\n")

		// Calculate available height for the viewport to leave room for action panel
		inspectHeight := m.height - 16 // Leave space for header, footer, and action panel

		// Adjust viewport height if needed
		if m.viewport.Height != inspectHeight {
			m.viewport.Height = inspectHeight
		}

		sb.WriteString(m.viewport.View())

		// Add action panel after the viewport
		sb.WriteString("\n\n")
		sb.WriteString(m.renderActionPanel())

	case LogsMode:
		// Render logs view
		logsHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#88c0d0")).
			Render(fmt.Sprintf("Logs for %s", m.selectedName))

		sb.WriteString(logsHeader)
		sb.WriteString("\n\n")
		sb.WriteString(m.viewport.View())
	case MonitorMode:
		// Render monitoring view
		monitorHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#88c0d0")).
			Render(fmt.Sprintf("Monitoring %s", m.selectedName))

		sb.WriteString(monitorHeader)
		sb.WriteString("\n\n")
		sb.WriteString(m.viewport.View())
	}

	// Footer with status and help
	footerText := fmt.Sprintf("%s | Press ? for help", m.statusMsg)
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4c566a")).
		Render(footerText)

	sb.WriteString("\n")
	sb.WriteString(footer)

	// Help section
	if m.showHelp {
		sb.WriteString("\n\n")
		sb.WriteString(m.renderHelp())
	}

	return sb.String()
}

// renderTabBar renders the tab bar
func (m FullModel) renderTabBar() string {
	tabs := []string{
		IconContainer + "Containers",
		IconImage + "Images",
		IconVolume + "Volumes",
		IconNetwork + "Networks",
		IconCompose + "Compose",
	}

	var renderedTabs []string
	for i, t := range tabs {
		style := lipgloss.NewStyle().
			Padding(0, 2)

		if i == int(m.currentTab) {
			style = style.
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#5f87ff")).
				Bold(true)
		}

		renderedTabs = append(renderedTabs, style.Render(t))
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs...)
}

// renderHelp renders the help text
func (m FullModel) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Keyboard Shortcuts:"))
	sb.WriteString("\n\n")

	// Global commands
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")).
		Render("Global:"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %sQuit, %sToggle help, %sRefresh", IconQuit, IconHelp, IconRefresh))
	sb.WriteString("\n\n")

	// Navigation
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")).
		Render("Navigation:"))
	sb.WriteString("\n")
	sb.WriteString("  ↑/k: Up, ↓/j: Down, Tab/→: Next tab, Shift+Tab/←: Previous tab")
	sb.WriteString("\n\n")

	// Resource actions
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")).
		Render("Resource Actions:"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %sInspect, %sLogs, %sMonitor, %sBack",
		IconInspect, IconLogs, IconMonitor, IconBack))
	sb.WriteString("\n\n")

	// Tab-specific actions
	switch m.currentTab {
	case ContainersTab:
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")).
			Render("Container Actions:"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %sStart, %sStop, %sRestart, %sPause, %sUnpause, %sKill, %sRemove",
			IconStart, IconStop, IconRestart, IconPause, IconUnpause, IconKill, IconRemove))
	case ComposeTab:
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")).
			Render("Compose Actions:"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %sUp, %sDown, %sPull, %sLogs",
			IconStart, IconStop, IconRefresh, IconLogs))
	}

	return sb.String()
}

// renderActionPanel renders a panel of available actions based on current context
func (m FullModel) renderActionPanel() string {
	var sb strings.Builder

	// Style for the panel title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5f87ff")).
		Bold(true)

	// Style for action buttons
	actionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2e3440")).
		Background(lipgloss.Color("#88c0d0")).
		Padding(0, 1).
		Margin(0, 1, 0, 0)

	sb.WriteString(titleStyle.Render("Available Actions:") + "\n")

	// Create a row of action buttons
	var actions []string

	// Common actions for all inspect views
	actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Refresh [r]", IconRefresh)))
	actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Back [Esc]", IconBack)))

	// Tab-specific actions
	switch m.currentTab {
	case ContainersTab:
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Start [s]", IconStart)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Stop [S]", IconStop)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Restart [R]", IconRestart)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Logs [l]", IconLogs)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Monitor [m]", IconMonitor)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Remove [d]", IconRemove)))
	case ImagesTab:
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Remove [d]", IconRemove)))
	case VolumesTab:
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Remove [d]", IconRemove)))
	case NetworksTab:
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Remove [d]", IconRemove)))
	case ComposeTab:
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Up [u]", IconStart)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Down [d]", IconStop)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Pull [p]", IconRefresh)))
		actions = append(actions, actionStyle.Render(fmt.Sprintf("%s Logs [l]", IconLogs)))
	}

	// Render the action buttons in a row
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, actions...))

	// Create a box around the whole thing
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4c566a")).
		Padding(1).
		Width(m.width - 4)

	return boxStyle.Render(sb.String())
}

// Helper functions

// formatBytes converts bytes to a human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Message types for handling async operations
type fullContainersMsg struct {
	containers []docker.ContainerInfo
}

type fullImagesMsg struct {
	images []docker.ImageInfo
}

type fullVolumesMsg struct {
	volumes []docker.VolumeInfo
}

type fullNetworksMsg struct {
	networks []docker.NetworkInfo
}

type fullLogsMsg struct {
	content string
}

type fullInspectMsg struct {
	content string
}

type fullActionResultMsg struct {
	success bool
	message string
	action  string
}

type fullErrMsg struct {
	err error
}

type fullStatsMsg struct {
	content string
}

type tickMsg struct{}

type dockerConnectionMsg struct {
	connected bool
	err       error
}

type connectionCheckTickMsg struct{}

// New message type for handling after-action state changes
type afterActionMsg struct {
	action string // "inspect" or "list"
}

type composeProjectsMsg struct {
	projects []docker.ComposeInfo
}

type composeServicesMsg []docker.ComposeServiceInfo

// Define the message type for compose services
type fullComposeServicesMsg struct {
	services    []docker.ComposeServiceInfo
	projectName string
	error       error
}

// Define a message type for compose containers
type fullComposeContainersMsg struct {
	containers  []docker.ContainerInfo
	projectName string
	error       error
}

// Update the renderComposeTab method to handle cases where no projects are found
func (m *FullModel) renderComposeTab() string {
	if m.loading && m.composeTable.Width() == 0 {
		return "Loading Docker Compose projects..."
	}

	if m.currentMode == InspectMode {
		return m.renderComposeInspect()
	}

	// If no projects are found, show a helpful message
	if len(m.composeProjects) == 0 {
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5f87ff")).
			Bold(true).
			Padding(1)

		infoStyle := lipgloss.NewStyle().
			Padding(1)

		var sb strings.Builder
		sb.WriteString(helpStyle.Render("No Docker Compose projects found"))
		sb.WriteString("\n\n")
		sb.WriteString(infoStyle.Render("Possible reasons:"))
		sb.WriteString("\n")
		sb.WriteString(infoStyle.Render("1. You don't have any Docker Compose projects running"))
		sb.WriteString("\n")
		sb.WriteString(infoStyle.Render("2. Docker Compose is not installed or not in your PATH"))
		sb.WriteString("\n")
		sb.WriteString(infoStyle.Render("3. Your Docker Compose version might not support the 'ls' command"))
		sb.WriteString("\n\n")
		sb.WriteString(infoStyle.Render("Try running 'docker compose ls' in your terminal to verify."))

		return sb.String()
	}

	return m.composeTable.View()
}

// renderComposeInspect renders the compose inspection view in a fancy style
func (m *FullModel) renderComposeInspect() string {
	content, updatedContainers := views.ComposeInspect(
		m.selectedName,
		m.selectedPath,
		m.selectedProject,
		m.selectedProjectPath,
		m.inspectContent,
		m.composeServicesLoading,
		m.composeServices,
		m.composeContainers,
		m.composeContainersLoading,
		m.containers,
		m.viewport.Width,
		m.viewport.Height,
		m.ctx,
		m.docker,
	)

	// Update the composeContainers with the potentially modified containers
	if len(updatedContainers) > 0 && len(m.composeContainers) == 0 {
		m.composeContainers = updatedContainers
		m.statusMsg = fmt.Sprintf("Found %d containers for project %s", len(updatedContainers), m.selectedName)
	}

	return content
}

// fetchComposeContainers fetches containers for a Docker Compose project
func (m FullModel) fetchComposeContainers() tea.Msg {
	if m.selectedName == "" {
		return fullComposeContainersMsg{
			containers:  []docker.ContainerInfo{},
			projectName: "",
			error:       fmt.Errorf("no project selected"),
		}
	}

	composeContainers, err := views.FetchComposeContainers(m.ctx, m.docker, m.selectedName)

	if err != nil {
		return fullComposeContainersMsg{
			containers:  []docker.ContainerInfo{},
			projectName: m.selectedName,
			error:       err,
		}
	}

	// Return the result
	return fullComposeContainersMsg{
		containers:  composeContainers,
		projectName: m.selectedName,
	}
}

// Helper function to jump to a specific container
func (m *FullModel) jumpToContainer(id string) {
	// First, refresh the container list to ensure we have the latest data
	containers, err := m.docker.ListContainers(m.ctx, true)
	if err == nil {
		m.containers = containers
	}

	// Switch to Containers tab
	m.currentTab = ContainersTab
	m.currentMode = ListMode

	// Find the container in the list and select it
	foundIndex := -1

	// First try exact ID match
	for i, container := range m.containers {
		if strings.HasPrefix(container.ID, id) {
			foundIndex = i
			break
		}
	}

	// If not found by ID, try name match (for cases where the ID in compose view might be different)
	if foundIndex == -1 && len(m.composeContainers) > 0 {
		// Find the container name from composeContainers
		var containerName string
		for _, c := range m.composeContainers {
			if strings.HasPrefix(c.ID, id) {
				containerName = c.Name
				// Handle service name in parentheses
				if idx := strings.Index(containerName, " ("); idx > 0 {
					containerName = containerName[:idx]
				}
				break
			}
		}

		// If we found a name, look for it in the main containers list
		if containerName != "" {
			for i, container := range m.containers {
				// Some container names have a leading slash that needs to be trimmed
				name := strings.TrimPrefix(container.Name, "/")
				if name == containerName {
					foundIndex = i
					break
				}
			}
		}
	}

	// If still not found, try a more fuzzy matching approach with container IDs
	if foundIndex == -1 {
		// Try matching just the first few characters of the ID
		for i, container := range m.containers {
			if len(id) >= 6 && len(container.ID) >= 6 &&
				strings.EqualFold(container.ID[:6], id[:6]) {
				foundIndex = i
				break
			}
		}
	}

	// If found, update the cursor position in the container table
	if foundIndex >= 0 {
		m.containerTable.SetCursor(foundIndex)
		m.updateSelection()
		m.statusMsg = fmt.Sprintf("Selected container: %s", m.containers[foundIndex].Name)
	} else {
		m.statusMsg = fmt.Sprintf("Container not found in main list. Try refreshing.")
	}
}
