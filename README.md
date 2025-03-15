# Docker Tea 🐳

Docker Tea is a modern terminal-based UI for Docker management. It provides a clean, intuitive interface for monitoring and controlling Docker containers, images, volumes, and networks from your terminal.

## ✨ Features

- **Multi-tab interface** for managing different Docker resources
  - 🐳 Containers
  - 📦 Images
  - 💾 Volumes
  - 🌐 Networks

- **Container management**
  - ▶️ Start, ⏹️ stop, 🔁 restart, ⏸️ pause, ⏯️ unpause, ⚡ kill, and 🗑️ remove containers
  - 🔍 View detailed container information
  - 📜 View container logs
  - 📊 Monitor container resource usage in real-time

- **Resource inspection**
  - 🔍 Detailed inspection of containers, images, volumes, and networks
  - 👁️ User-friendly presentation of resource information
  - 📊 Real-time container resource monitoring (CPU, memory, network, I/O)

- **Keyboard navigation**
  - Vim-style navigation (j/k)
  - Arrow keys support
  - Tab navigation between resource types

## 🚀 Installation

### Prerequisites

- Go 1.18 or higher
- Docker installed and running
- Make (optional, for using Makefile)

### Build from Source

1. Clone the repository:
   ```
   git clone https://github.com/klejdi94/docker-tea.git
   cd docker-tea
   ```

2. Build and run the application:

   **Using Makefile (recommended):**
   ```
   # Build the application
   make build

   # Build and run the application
   make run

   # Show all available commands
   make help
   ```

   **Using Scripts:**

   For Windows:
   ```
   .\scripts\run.bat
   ```

   For Linux/macOS:
   ```
   chmod +x ./scripts/run.sh
   ./scripts/run.sh
   ```

## 🎮 Usage

### Keyboard Controls

#### Global Controls
- 🚪 `q`: Quit
- ❓ `?`: Toggle help
- 🔄 `r`: Refresh data

#### Navigation
- `↑/k`: Move up
- `↓/j`: Move down
- `Tab/→`: Next tab
- `Shift+Tab/←`: Previous tab
- `Home`: Go to top
- `End`: Go to bottom
- `Page Up`: Page up
- `Page Down`: Page down

#### Resource Actions
- 🔍 `i/Enter`: Inspect selected resource
- 📜 `l`: View logs (containers only)
- 📊 `m`: Monitor resource usage (containers only)
- ← `Esc`: Back to list view

#### Container Actions
- ▶️ `s`: Start container
- ⏹️ `S`: Stop container
- 🔁 `R`: Restart container
- ⏸️ `p`: Pause container
- ⏯️ `u`: Unpause container
- ⚡ `K`: Kill container
- 🗑️ `d`: Remove container

## 🔧 Development

### Project Structure

- `cmd/docker-tea/`: Main application entry point
- `internal/`: Internal packages
  - `config/`: Configuration management
  - `docker/`: Docker API interaction
  - `ui/`: User interface components
- `scripts/`: Build and run scripts
  - `run.bat`: Windows script
  - `run.sh`: Linux/macOS script
- `Makefile`: Build automation

### Building for Development

```
# Using Go directly
go build -o docker-tea ./cmd/docker-tea

# Using Make
make build

# Build for multiple platforms
make build-all
```

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The TUI framework
- [Docker Engine API](https://docs.docker.com/engine/api/) - Docker API documentation