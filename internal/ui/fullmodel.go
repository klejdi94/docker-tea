package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/klejdi94/docker-tea/internal/config"
	"github.com/klejdi94/docker-tea/internal/docker"
)

// Icons for UI elements
const (
	// Resource type icons
	IconContainer = "🐳 "
	IconImage     = "📦 "
	IconVolume    = "💾 "
	IconNetwork   = "🌐 "

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
	config          *config.Config
	docker          *docker.Service
	ctx             context.Context
	width           int
	height          int
	loading         bool
	err             error
	dockerConnected bool
	containerTable  table.Model
	imageTable      table.Model
	volumeTable     table.Model
	networkTable    table.Model
	viewport        viewport.Model
	currentTab      Tab
	currentMode     Mode
	statusMsg       string
	containers      []docker.ContainerInfo
	images          []docker.ImageInfo
	volumes         []docker.VolumeInfo
	networks        []docker.NetworkInfo
	logContent      string
	inspectContent  string
	statsContent    string
	selectedID      string
	selectedName    string
	showHelp        bool
	ticker          *time.Ticker
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
		key.WithKeys("d", "delete"),
		key.WithHelp("d", "remove"),
	),
}

// NewFullModel creates a new model for Docker Tea
func NewFullModel(dockerService *docker.Service, config *config.Config, ctx context.Context) FullModel {
	return FullModel{
		config:          config,
		docker:          dockerService,
		ctx:             ctx,
		loading:         true,
		dockerConnected: true, // Assume connected, we'll check immediately
		statusMsg:       "Initializing...",
		currentTab:      ContainersTab,
		currentMode:     ListMode,
		viewport:        viewport.New(0, 0),
	}
}

// Init initializes the model
func (m FullModel) Init() tea.Cmd {
	return tea.Batch(
		m.checkDockerConnection,
		m.fetchContainers,
		m.fetchImages,
		m.fetchVolumes,
		m.fetchNetworks,
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
				return m, m.inspectResource
			}

			m.statusMsg = "Refreshing..."
			return m, tea.Batch(
				m.fetchContainers,
				m.fetchImages,
				m.fetchVolumes,
				m.fetchNetworks,
			)

		case key.Matches(msg, DefaultFullKeyMap.NextTab):
			if m.currentMode == ListMode {
				m.currentTab = (m.currentTab + 1) % 4 // Cycle through the tabs
				return m, nil
			}

		case key.Matches(msg, DefaultFullKeyMap.PrevTab):
			if m.currentMode == ListMode {
				m.currentTab = (m.currentTab - 1 + 4) % 4 // Cycle through the tabs
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

		// Handle action keys
		if m.currentMode == ListMode {
			// Update selection before performing actions
			m.updateSelection()

			switch {
			case key.Matches(msg, DefaultFullKeyMap.Inspect):
				if m.selectedID != "" {
					m.currentMode = InspectMode
					return m, m.inspectResource
				}

			case key.Matches(msg, DefaultFullKeyMap.Logs):
				// Only containers have logs
				if m.currentTab == ContainersTab && m.selectedID != "" {
					m.currentMode = LogsMode
					return m, m.fetchLogs
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

			// Container actions - only apply in containers tab
			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Start):
				return m, m.containerAction("start")

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Stop):
				return m, m.containerAction("stop")

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Restart):
				return m, m.containerAction("restart")

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Pause):
				return m, m.containerAction("pause")

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Resume):
				return m, m.containerAction("unpause")

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Kill):
				return m, m.containerAction("kill")

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				return m, m.containerAction("remove")

			// Image actions - only apply in images tab
			case m.currentTab == ImagesTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				return m, m.imageAction("remove")

			// Volume actions - only apply in volumes tab
			case m.currentTab == VolumesTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				return m, m.volumeAction("remove")

			// Network actions - only apply in networks tab
			case m.currentTab == NetworksTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				return m, m.networkAction("remove")
			}

			// Handle navigation keys for tables
			table := m.getCurrentTable()
			if table.Width() > 0 {
				*table, cmd = table.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else if m.currentMode == InspectMode {
			// Allow actions directly from inspect mode
			switch {
			case key.Matches(msg, DefaultFullKeyMap.Logs):
				// Only containers have logs
				if m.currentTab == ContainersTab && m.selectedID != "" {
					m.currentMode = LogsMode
					return m, m.fetchLogs
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

			// Container actions - only apply in containers tab
			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Start):
				m.statusMsg = "Starting container..."
				return m, tea.Batch(
					m.containerAction("start"),
					// Return to inspect mode after action completes
					func() tea.Msg {
						return afterActionMsg{action: "inspect"}
					},
				)

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Stop):
				m.statusMsg = "Stopping container..."
				return m, tea.Batch(
					m.containerAction("stop"),
					func() tea.Msg {
						return afterActionMsg{action: "inspect"}
					},
				)

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Restart):
				m.statusMsg = "Restarting container..."
				return m, tea.Batch(
					m.containerAction("restart"),
					func() tea.Msg {
						return afterActionMsg{action: "inspect"}
					},
				)

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Pause):
				m.statusMsg = "Pausing container..."
				return m, tea.Batch(
					m.containerAction("pause"),
					func() tea.Msg {
						return afterActionMsg{action: "inspect"}
					},
				)

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Resume):
				m.statusMsg = "Unpausing container..."
				return m, tea.Batch(
					m.containerAction("unpause"),
					func() tea.Msg {
						return afterActionMsg{action: "inspect"}
					},
				)

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Kill):
				m.statusMsg = "Killing container..."
				return m, tea.Batch(
					m.containerAction("kill"),
					func() tea.Msg {
						return afterActionMsg{action: "list"}
					},
				)

			case m.currentTab == ContainersTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				m.statusMsg = "Removing container..."
				return m, tea.Batch(
					m.containerAction("remove"),
					func() tea.Msg {
						return afterActionMsg{action: "list"}
					},
				)

			// Image actions in inspect mode
			case m.currentTab == ImagesTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				m.statusMsg = "Removing image..."
				return m, tea.Batch(
					m.imageAction("remove"),
					func() tea.Msg {
						return afterActionMsg{action: "list"}
					},
				)

			// Volume actions in inspect mode
			case m.currentTab == VolumesTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				m.statusMsg = "Removing volume..."
				return m, tea.Batch(
					m.volumeAction("remove"),
					func() tea.Msg {
						return afterActionMsg{action: "list"}
					},
				)

			// Network actions in inspect mode
			case m.currentTab == NetworksTab && key.Matches(msg, DefaultFullKeyMap.Remove):
				m.statusMsg = "Removing network..."
				return m, tea.Batch(
					m.networkAction("remove"),
					func() tea.Msg {
						return afterActionMsg{action: "list"}
					},
				)
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
		m.viewport.SetContent(m.inspectContent)
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
			return m, m.inspectResource
		} else if msg.action == "list" {
			// Return to list mode
			m.currentMode = ListMode
			// Refresh the containers
			return m, m.fetchContainers
		}
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

	// Container actions
	if m.currentTab == ContainersTab {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")).
			Render("Container Actions:"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %sStart, %sStop, %sRestart, %sPause, %sUnpause, %sKill, %sRemove",
			IconStart, IconStop, IconRestart, IconPause, IconUnpause, IconKill, IconRemove))
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
