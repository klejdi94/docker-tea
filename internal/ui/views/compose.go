package views

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/klejdi94/docker-tea/internal/docker"
)

// ComposeInspect renders the compose inspection view
func ComposeInspect(
	selectedName, selectedPath, selectedProject, selectedProjectPath, inspectContent string,
	composeServicesLoading bool,
	composeServices []docker.ComposeServiceInfo,
	composeContainers []docker.ContainerInfo,
	composeContainersLoading bool,
	containers []docker.ContainerInfo,
	viewportWidth, viewportHeight int,
	ctx context.Context,
	dockerService *docker.Service,
) string {
	if composeServicesLoading {
		return "Loading compose services..."
	}

	// StringBuilder for building the full UI
	var sb strings.Builder

	// Project header - use both variables for reliability
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88c0d0"))
	projectName := selectedName
	if selectedProject != "" {
		projectName = selectedProject
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("Compose Project: %s", projectName)))
	sb.WriteString("\n")

	projectPath := selectedPath
	if selectedProjectPath != "" {
		projectPath = selectedProjectPath
	}
	sb.WriteString(fmt.Sprintf("Path: %s", projectPath))
	sb.WriteString("\n\n")

	// Try to directly read the compose file content
	var composeFileContent string
	if _, err := os.Stat(projectPath); err == nil {
		// File exists, read it
		data, err := os.ReadFile(projectPath)
		if err == nil {
			composeFileContent = string(data)
		}
	}

	// Get all available containers for reference
	allContainers := containers
	if len(allContainers) == 0 && dockerService != nil {
		// Try to fetch them if not already available
		fetchedContainers, err := dockerService.ListContainers(ctx, true)
		if err == nil {
			allContainers = fetchedContainers
		}
	}

	// Check if we should extract container info from JSON or add all visible containers
	extractedContainer := false
	tmpComposeContainers := composeContainers
	if len(tmpComposeContainers) == 0 {
		// First try to parse inspectContent as container JSON
		if len(inspectContent) > 0 {
			var containerData map[string]interface{}
			if err := json.Unmarshal([]byte(inspectContent), &containerData); err == nil {
				// Check if this looks like container JSON
				if id, ok := containerData["Id"].(string); ok {
					// Extract container info
					name := "<unknown>"
					if nameInterface, ok := containerData["Name"].(string); ok {
						name = strings.TrimPrefix(nameInterface, "/")
					}

					// Extract image
					image := "<unknown>"
					if config, ok := containerData["Config"].(map[string]interface{}); ok {
						if img, ok := config["Image"].(string); ok {
							image = img
						}
					}

					// Extract state
					state := "unknown"
					if stateData, ok := containerData["State"].(map[string]interface{}); ok {
						if status, ok := stateData["Status"].(string); ok {
							state = status
						}
					}

					// Create a temporary container for display
					tempContainer := docker.ContainerInfo{
						ID:    id[:12],
						Name:  name,
						Image: image,
						State: state,
					}

					// Clear any existing containers and add this one
					tmpComposeContainers = []docker.ContainerInfo{tempContainer}
					extractedContainer = true
				}
			}
		}

		// If we still have no containers, add any that match the services in the file
		if !extractedContainer && composeFileContent != "" {
			// Try to extract service names from the file content
			for _, container := range allContainers {
				// Check if this container's image or name matches anything in the compose file
				if strings.Contains(composeFileContent, container.Image) ||
					strings.Contains(composeFileContent, container.Name) {
					tmpComposeContainers = append(tmpComposeContainers, container)
				}
			}
		}
	}

	// Parse the compose file content (manually since the service might not be doing it correctly)
	tmpComposeServices := composeServices
	if composeFileContent != "" && len(tmpComposeServices) == 0 {
		// Better parsing logic to extract only actual services
		lines := strings.Split(composeFileContent, "\n")

		// First pass: identify the services section and its direct children
		inServices := false
		servicesIndent := 0
		serviceEntries := make(map[string]bool)

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
				continue // Skip empty lines and comments
			}

			// Count leading spaces to determine indentation level
			indent := 0
			for _, char := range line {
				if char == ' ' {
					indent++
				} else if char == '\t' {
					indent += 4 // Count tabs as 4 spaces
				} else {
					break
				}
			}

			// Detect services section
			if strings.HasPrefix(trimmedLine, "services:") {
				inServices = true
				servicesIndent = indent
				continue
			}

			// If outside services or not deeper indentation, skip
			if !inServices || indent <= servicesIndent {
				if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
					// We've exited the services section
					if indent <= servicesIndent {
						inServices = false
					}
				}
				continue
			}

			// This is a direct child of services (should be a service name)
			if indent == servicesIndent+2 && strings.HasSuffix(trimmedLine, ":") {
				serviceName := strings.TrimSuffix(trimmedLine, ":")
				serviceEntries[serviceName] = true
			}
		}

		// Second pass: extract service properties
		for serviceName := range serviceEntries {
			// Default service with minimal info
			service := docker.ComposeServiceInfo{
				Name: serviceName,
			}

			// Extract more details if possible
			inServiceDef := false
			serviceIndent := 0

			for _, line := range lines {
				trimmedLine := strings.TrimSpace(line)
				if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
					continue
				}

				// Count leading spaces
				indent := 0
				for _, char := range line {
					if char == ' ' {
						indent++
					} else if char == '\t' {
						indent += 4
					} else {
						break
					}
				}

				// Find this service's definition
				if !inServiceDef && strings.HasPrefix(trimmedLine, serviceName+":") {
					inServiceDef = true
					serviceIndent = indent
					continue
				}

				// If we're not in this service definition, skip
				if !inServiceDef {
					continue
				}

				// If we've moved to a different section (same indent as service or less), exit
				if indent <= serviceIndent {
					inServiceDef = false
					continue
				}

				// Process service properties
				if indent == serviceIndent+2 {
					// Direct properties of the service
					if strings.HasPrefix(trimmedLine, "image:") {
						service.Image = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "image:"))
					} else if strings.HasPrefix(trimmedLine, "ports:") {
						// Skip the ports header - we'll process individually later
					}
				} else if indent > serviceIndent+2 {
					// Handle lists like ports
					if strings.HasPrefix(trimmedLine, "- ") {
						// Check what kind of list item this is based on context
						if strings.Contains(strings.ToLower(line), "port") {
							port := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "- "))
							service.Ports = append(service.Ports, port)
						}
					}
				}
			}

			// Add the service to our list
			tmpComposeServices = append(tmpComposeServices, service)
		}
	}

	// Debug info about services and containers
	if len(tmpComposeServices) > 0 {
		sb.WriteString(fmt.Sprintf("Found %d services in compose file.", len(tmpComposeServices)))
	} else {
		sb.WriteString("Found 0 services in compose file.")
	}
	sb.WriteString("\n\n")

	// Service list section if services were found
	if len(tmpComposeServices) > 0 {
		serviceHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a3be8c"))
		sb.WriteString(serviceHeaderStyle.Render("Services:"))
		sb.WriteString("\n")

		// Table styles with fixed widths for better alignment
		nameColWidth := 25
		imageColWidth := 25
		portsColWidth := 30

		tableHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d8dee9"))
		nameColStyle := lipgloss.NewStyle().Width(nameColWidth).Foreground(lipgloss.Color("#88c0d0"))
		imageColStyle := lipgloss.NewStyle().Width(imageColWidth).Foreground(lipgloss.Color("#a3be8c"))
		portsColStyle := lipgloss.NewStyle().Width(portsColWidth).Foreground(lipgloss.Color("#ebcb8b"))

		// Render header with proper spacing
		sb.WriteString(tableHeaderStyle.Render(
			fmt.Sprintf("%-*s │ %-*s │ %-*s",
				nameColWidth, "Name",
				imageColWidth, "Image",
				portsColWidth, "Ports")))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", nameColWidth+imageColWidth+portsColWidth+6))
		sb.WriteString("\n")

		// Format each service row
		for _, service := range tmpComposeServices {
			// Truncate image name if too long
			imageName := service.Image
			if imageName == "" {
				imageName = "-"
			} else if len(imageName) > imageColWidth-3 {
				imageName = imageName[:imageColWidth-6] + "..."
			}

			// Format ports as a comma-separated list
			portsText := strings.Join(service.Ports, ", ")
			if portsText == "" {
				portsText = "-"
			} else if len(portsText) > portsColWidth-3 {
				portsText = portsText[:portsColWidth-6] + "..."
			}

			// Truncate name if necessary
			name := service.Name
			if len(name) > nameColWidth-3 {
				name = name[:nameColWidth-6] + "..."
			}

			// Render service row with proper alignment
			sb.WriteString(
				nameColStyle.Render(name) + " │ " +
					imageColStyle.Render(imageName) + " │ " +
					portsColStyle.Render(portsText) + "\n")
		}
	}

	// Define icons
	const (
		IconRunning    = "🟢 "
		IconStopped    = "🔴 "
		IconPaused     = "⏸️  "
		IconRestarting = "🔄 "
	)

	// Container section header if containers were found
	if len(tmpComposeContainers) > 0 {
		sb.WriteString("\n")
		containerHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#b48ead"))
		sb.WriteString(containerHeaderStyle.Render("Containers:"))
		sb.WriteString("\n")

		// Table header for containers
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d8dee9"))
		sb.WriteString(headerStyle.Render("ID │ Name │ Status │ Image"))
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", viewportWidth))
		sb.WriteString("\n")

		// Row styles
		rowStyle := lipgloss.NewStyle()
		idColStyle := lipgloss.NewStyle().Width(12).Foreground(lipgloss.Color("#88c0d0"))
		nameColStyle := lipgloss.NewStyle().Width(30).Foreground(lipgloss.Color("#a3be8c"))
		stateColStyle := lipgloss.NewStyle().Width(12)
		imageColStyle := lipgloss.NewStyle().Width(40).Foreground(lipgloss.Color("#ebcb8b"))

		// Status styles
		runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a3be8c"))
		stoppedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#bf616a"))
		pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ebcb8b"))
		restartingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#b48ead"))

		// Add each container as a row
		for i, container := range tmpComposeContainers {
			// Truncate container name if too long
			name := container.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}

			// Apply state-based styling
			state := container.State
			var stateStyle lipgloss.Style
			switch state {
			case "running":
				stateStyle = runningStyle
				state = IconRunning + state
			case "exited", "stopped":
				stateStyle = stoppedStyle
				state = IconStopped + state
			case "paused":
				stateStyle = pausedStyle
				state = IconPaused + state
			case "restarting":
				stateStyle = restartingStyle
				state = IconRestarting + state
			}

			// Truncate image name if too long
			image := container.Image
			if len(image) > 40 {
				image = image[:37] + "..."
			}

			// Render the row with container details
			row := rowStyle.Render(
				idColStyle.Render(container.ID) + " │ " +
					nameColStyle.Render(name) + " │ " +
					stateColStyle.Render(stateStyle.Render(state)) + " │ " +
					imageColStyle.Render(image))

			// Add a number for selection
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, row))
		}

		// Add container navigation help
		sb.WriteString("\n")
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa")).Italic(true)
		sb.WriteString(helpStyle.Render("💡 Press 'c' + container number (1-9) to switch to Containers tab and focus on that container"))
		sb.WriteString("\n")
	} else if composeContainersLoading {
		// Show loading message if containers are still loading
		sb.WriteString("\n")
		sb.WriteString("Loading containers...")
		sb.WriteString("\n")
	} else {
		// Show message if no containers are found
		sb.WriteString("\n")
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#bf616a"))
		sb.WriteString(errorStyle.Render("No containers found for this compose project."))
		sb.WriteString("\n\n")

		// Add suggestions for troubleshooting
		tipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ebcb8b")).Italic(true)
		sb.WriteString(tipStyle.Render("Tips:"))
		sb.WriteString("\n")
		sb.WriteString("- Check if containers are running with 'docker ps'")
		sb.WriteString("\n")
		sb.WriteString("- Try pressing 'u' to start the Docker Compose project")
		sb.WriteString("\n")
		sb.WriteString("- Verify the container name matches the project name pattern")
		sb.WriteString("\n")
	}

	// YAML content section
	sb.WriteString("\n")
	yamlHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5e81ac"))

	// If we have directly read the file content, display it
	if composeFileContent != "" {
		sb.WriteString(yamlHeaderStyle.Render("Compose File Content:"))
		sb.WriteString("\n")
		sb.WriteString(composeFileContent)
	} else if extractedContainer {
		// Otherwise show inspection content, if extracted container show as Container JSON
		sb.WriteString(yamlHeaderStyle.Render("Container JSON:"))
		sb.WriteString("\n")
		sb.WriteString(inspectContent)
	} else {
		sb.WriteString(yamlHeaderStyle.Render("YAML Content:"))
		sb.WriteString("\n")
		sb.WriteString(inspectContent)
	}

	return sb.String()
}

// FetchComposeContainers finds containers belonging to a compose project
func FetchComposeContainers(
	ctx context.Context,
	dockerService *docker.Service,
	projectName string,
) ([]docker.ContainerInfo, error) {
	if projectName == "" {
		return []docker.ContainerInfo{}, fmt.Errorf("no project selected")
	}

	// Normalize the project name for matching
	normalizedProjectName := strings.ToLower(projectName)

	// Create common variations of the project name
	dashVariant := strings.ReplaceAll(normalizedProjectName, "_", "-")
	underscoreVariant := strings.ReplaceAll(normalizedProjectName, "-", "_")

	// Fetch all containers to find matches
	containers, err := dockerService.ListContainers(ctx, true)
	if err != nil {
		return []docker.ContainerInfo{}, err
	}

	// Filter containers that match the project
	var composeContainers []docker.ContainerInfo

	// First pass: check for exact container names or labels
	for _, container := range containers {
		// Check if container name contains the project name
		matched := false
		containerNameLower := strings.ToLower(container.Name)

		// Try basic name matching first (most common case)
		if strings.Contains(containerNameLower, normalizedProjectName) ||
			strings.Contains(containerNameLower, dashVariant) ||
			strings.Contains(containerNameLower, underscoreVariant) {
			composeContainers = append(composeContainers, container)
			matched = true
		}

		// If not matched by name, try to examine container labels
		if !matched {
			// Get container details to check for labels
			containerInfo, err := dockerService.InspectContainer(ctx, container.ID)
			if err != nil {
				continue
			}

			// Check container labels for project association
			var inspectData map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(containerInfo), &inspectData); jsonErr == nil {
				// Check the container config for labels
				if config, ok := inspectData["Config"].(map[string]interface{}); ok {
					if labels, ok := config["Labels"].(map[string]interface{}); ok {
						// Check for various Docker Compose related labels
						for k, v := range labels {
							labelVal, isString := v.(string)
							if !isString {
								continue
							}

							// Check common compose label patterns
							if k == "com.docker.compose.project" &&
								(strings.EqualFold(labelVal, projectName) ||
									strings.EqualFold(labelVal, dashVariant) ||
									strings.EqualFold(labelVal, underscoreVariant)) {
								composeContainers = append(composeContainers, container)
								matched = true
								break
							}

							// Also check for alternate label patterns
							if (strings.Contains(strings.ToLower(k), "compose") ||
								strings.Contains(strings.ToLower(k), "project")) &&
								(strings.Contains(strings.ToLower(labelVal), normalizedProjectName) ||
									strings.Contains(strings.ToLower(labelVal), dashVariant) ||
									strings.Contains(strings.ToLower(labelVal), underscoreVariant)) {
								composeContainers = append(composeContainers, container)
								matched = true
								break
							}
						}
					}
				}
			}
		}
	}

	// If we still have no containers, try using the Docker Compose CLI
	if len(composeContainers) == 0 {
		// Try using docker compose ps to get containers directly
		cmd := exec.Command("docker", "compose", "--project-name", projectName, "ps", "--format", "json")
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			// Try to parse as JSON
			var containerList []map[string]interface{}
			if jsonErr := json.Unmarshal(output, &containerList); jsonErr == nil {
				for _, c := range containerList {
					if id, ok := c["ID"].(string); ok {
						// Found container ID, try to find it in our main list
						for _, existingContainer := range containers {
							if strings.HasPrefix(existingContainer.ID, id) ||
								(len(id) >= 12 && len(existingContainer.ID) >= 12 &&
									strings.HasPrefix(existingContainer.ID, id[:12])) {
								// Add the container if not already in the list
								alreadyAdded := false
								for _, added := range composeContainers {
									if added.ID == existingContainer.ID {
										alreadyAdded = true
										break
									}
								}
								if !alreadyAdded {
									composeContainers = append(composeContainers, existingContainer)
								}
								break
							}
						}
					}
				}
			}
		}
	}

	// If we still have no matches, consider any container with matching names
	if len(composeContainers) == 0 {
		// Deep name-based search
		for _, container := range containers {
			// In some cases, the path component in the container name is relevant
			parts := strings.Split(container.Name, "_")
			for _, part := range parts {
				if strings.EqualFold(part, normalizedProjectName) ||
					strings.EqualFold(part, dashVariant) ||
					strings.EqualFold(part, underscoreVariant) {
					// Add the container if not already in the list
					alreadyAdded := false
					for _, added := range composeContainers {
						if added.ID == container.ID {
							alreadyAdded = true
							break
						}
					}
					if !alreadyAdded {
						composeContainers = append(composeContainers, container)
					}
					break
				}
			}
		}
	}

	return composeContainers, nil
}
