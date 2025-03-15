package config

import (
	"time"
)

// Config holds application configuration settings
type Config struct {
	RefreshInterval time.Duration
	Theme           Theme
	LogFilePath     string
}

// Theme represents UI theme settings
type Theme struct {
	ContainerRunning string
	ContainerStopped string
	ContainerPaused  string
	HeaderColor      string
	BorderColor      string
	TitleColor       string
	TextColor        string
	StatusBarColor   string
}

// NewConfig creates and returns a new Config instance with default values
func NewConfig() *Config {
	return &Config{
		RefreshInterval: 5 * time.Second,
		Theme: Theme{
			ContainerRunning: "#4caf50", // Bright green
			ContainerStopped: "#f44336", // Bright red
			ContainerPaused:  "#ff9800", // Bright orange
			HeaderColor:      "#2e3440", // Dark slate blue
			BorderColor:      "#4c566a", // Medium slate blue
			TitleColor:       "#88c0d0", // Light blue
			TextColor:        "#d8dee9", // Off-white
			StatusBarColor:   "#2e3440", // Dark slate blue
		},
		LogFilePath: "docker-tui.log",
	}
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	// Currently just using default config, but can be extended to load from file
	return NewConfig(), nil
}
