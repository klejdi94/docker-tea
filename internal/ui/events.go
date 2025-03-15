package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klejdi94/docker-tea/internal/docker"
)

// DockerEventMsg represents a message sent when a Docker event occurs
type DockerEventMsg struct {
	Event docker.DockerEvent
}

// SetupEventListener creates a goroutine that listens for Docker events and
// forwards them to the Bubble Tea program
func SetupEventListener(ctx context.Context, dockerSvc *docker.Service, program *tea.Program) {
	// Create a cancellable context for the event listener
	eventCtx, cancel := context.WithCancel(ctx)

	// Start the event listener in a separate goroutine
	go func() {
		defer cancel() // Ensure context is cancelled when goroutine exits

		// Subscribe to Docker events
		err := dockerSvc.SubscribeToEvents(eventCtx, func(event docker.DockerEvent) {
			// Only forward events if program is set
			if program != nil {
				program.Send(DockerEventMsg{Event: event})
			}
		})

		// Log any errors that aren't just from context cancellation
		if err != nil && eventCtx.Err() == nil {
			if program != nil {
				program.Send(ErrorMsg{err: fmt.Errorf("Docker event subscription error: %w", err)})
			}
		}
	}()
}

// ErrorMsg represents an error message from the event listener
type ErrorMsg struct {
	err error
}

// StartEventSubscription creates a command to start event subscription
func StartEventSubscription(dockerSvc *docker.Service, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		// This will be run in a goroutine managed by Bubble Tea
		SetupEventListener(context.Background(), dockerSvc, program)
		return nil
	}
}

// HandleDockerEvent processes a Docker event and returns appropriate commands
func HandleDockerEvent(model *FullModel, event docker.DockerEvent) []tea.Cmd {
	var cmds []tea.Cmd

	// Update status message with event info
	model.statusMsg = fmt.Sprintf("Docker event: %s %s for %s", event.Action, event.Type, event.ID)

	// Refresh resources based on event type
	switch event.Type {
	case "container":
		cmds = append(cmds, model.fetchContainers)
	case "image":
		cmds = append(cmds, model.fetchImages)
	case "volume":
		cmds = append(cmds, model.fetchVolumes)
	case "network":
		cmds = append(cmds, model.fetchNetworks)
	}

	// If in monitor mode and the event is about the currently monitored container
	if model.currentMode == MonitorMode && model.selectedID == event.ID {
		cmds = append(cmds, model.fetchStats)
	}

	return cmds
}
