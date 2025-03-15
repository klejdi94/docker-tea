package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klejdi94/docker-tea/internal/config"
	"github.com/klejdi94/docker-tea/internal/docker"
	"github.com/klejdi94/docker-tea/internal/ui"
)

// Version information - set via build flags
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Create a cancellable context for the app
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling to gracefully shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("Shutting down...")
		cancel()
		os.Exit(0)
	}()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create the docker service
	dockerService, err := docker.NewDockerService()
	if err != nil {
		fmt.Printf("Failed to connect to Docker: %v\n", err)
		os.Exit(1)
	}

	// Create the model for Bubble Tea
	model := ui.NewFullModel(dockerService, cfg, ctx)

	// Initialize the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set up Docker event listener
	ui.SetupEventListener(ctx, dockerService, p)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
