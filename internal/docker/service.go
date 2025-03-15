package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// Service provides methods for interacting with Docker
type Service struct {
	client *client.Client
}

// ContainerInfo represents the container data we're interested in displaying
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Command string
	Status  string
	State   string
	Created time.Time
	Ports   []types.Port
}

// Port represents a port mapping
type Port struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	Type        string
}

// ImageInfo represents the image data we're interested in displaying
type ImageInfo struct {
	ID          string
	RepoTags    []string
	Size        int64
	CreatedAt   time.Time
	VirtualSize int64
}

// VolumeInfo represents the volume data we're interested in displaying
type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	CreatedAt  time.Time
	Size       int64
}

// NetworkInfo represents the network data we're interested in displaying
type NetworkInfo struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Containers map[string]NetworkContainer
}

// NetworkContainer represents a container connected to a network
type NetworkContainer struct {
	Name       string
	EndpointID string
}

// NetworkDetails contains detailed information about a network
type NetworkDetails struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Attachable bool
	Ingress    bool
	EnableIPv6 bool
	IPAM       map[string]interface{}
	Containers map[string]interface{}
	Options    map[string]string
	Labels     map[string]string
	CreatedAt  time.Time
}

// ContainerCreateConfig represents the configuration for creating a new container
type ContainerCreateConfig struct {
	Name        string
	Image       string
	Command     []string
	Env         []string
	Ports       []string // "hostPort:containerPort/protocol" format (e.g., "8080:80/tcp")
	Volumes     []string // "hostPath:containerPath" format (e.g., "/host:/container")
	Labels      map[string]string
	NetworkMode string
	Restart     string
	Memory      int64
	CPUShares   int64
}

// NewService creates a new Docker service with a given client
func NewService(client *client.Client) *Service {
	return &Service{
		client: client,
	}
}

// NewDockerService creates a new Docker service with the default client
func NewDockerService() (*Service, error) {
	// Initialize Docker client with default options
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return NewService(dockerClient), nil
}

// ListContainers returns a list of all containers
func (s *Service) ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error) {
	containers, err := s.client.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, err
	}

	var containerInfos []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0][1:] // Remove leading slash
		}

		id := c.ID
		if len(id) > 12 {
			id = id[:12] // Short ID
		}

		containerInfos = append(containerInfos, ContainerInfo{
			ID:      id,
			Name:    name,
			Image:   c.Image,
			Command: c.Command,
			Status:  c.Status,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
			Ports:   c.Ports,
		})
	}

	return containerInfos, nil
}

// GetContainerStats returns the stats for a container
func (s *Service) GetContainerStats(ctx context.Context, containerID string) (map[string]interface{}, error) {
	stats, err := s.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var statsJSON map[string]interface{}
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, err
	}

	return statsJSON, nil
}

// GetContainerStatsStream returns a stream of stats for a container
func (s *Service) GetContainerStatsStream(ctx context.Context, containerID string) (io.ReadCloser, error) {
	stats, err := s.client.ContainerStats(ctx, containerID, true)
	if err != nil {
		return nil, err
	}
	return stats.Body, nil
}

// ContainerStats represents processed container stats data
type ContainerStats struct {
	CPUPercentage    float64
	MemoryUsage      int64
	MemoryLimit      int64
	MemoryPercentage float64
	NetworkRx        int64
	NetworkTx        int64
	BlockRead        int64
	BlockWrite       int64
}

// GetProcessedStats returns processed container stats in a more usable format
func (s *Service) GetProcessedStats(ctx context.Context, containerID string) (ContainerStats, error) {
	// Check if context is already done before making the API call
	if ctx.Err() != nil {
		return ContainerStats{}, fmt.Errorf("context error before API call: %w", ctx.Err())
	}

	// Use a default timeout if none provided in the context
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	// Get container stats (non-streaming mode)
	stats, err := s.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return ContainerStats{}, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	// Read with timeout to prevent blocking indefinitely
	var statsJSON map[string]interface{}
	decodeErr := make(chan error, 1)
	decodeJSON := make(chan map[string]interface{}, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				decodeErr <- fmt.Errorf("panic while decoding JSON: %v", r)
			}
		}()

		var result map[string]interface{}
		err := json.NewDecoder(stats.Body).Decode(&result)
		if err != nil {
			decodeErr <- err
			return
		}
		decodeJSON <- result
	}()

	// Wait for decode or timeout
	select {
	case err := <-decodeErr:
		return ContainerStats{}, fmt.Errorf("failed to decode stats JSON: %w", err)
	case statsJSON = <-decodeJSON:
		// Decode successful
	case <-ctx.Done():
		return ContainerStats{}, fmt.Errorf("timeout decoding stats: %w", ctx.Err())
	}

	// Extract CPU data
	cpuPercent := 0.0
	if cpu, ok := statsJSON["cpu_stats"].(map[string]interface{}); ok {
		if preCPU, ok := statsJSON["precpu_stats"].(map[string]interface{}); ok {
			cpuDelta := float64(0)
			if cpuUsage, ok := cpu["cpu_usage"].(map[string]interface{}); ok {
				if preCPUUsage, ok := preCPU["cpu_usage"].(map[string]interface{}); ok {
					if totalUsage, ok := cpuUsage["total_usage"].(float64); ok {
						if preTotalUsage, ok := preCPUUsage["total_usage"].(float64); ok {
							cpuDelta = totalUsage - preTotalUsage
						}
					}
				}
			}

			systemDelta := float64(0)
			if systemUsage, ok := cpu["system_cpu_usage"].(float64); ok {
				if preSystemUsage, ok := preCPU["system_cpu_usage"].(float64); ok {
					systemDelta = systemUsage - preSystemUsage
				}
			}

			if systemDelta > 0.0 && cpuDelta > 0.0 {
				percpuLen := 0
				if cpuUsage, ok := cpu["cpu_usage"].(map[string]interface{}); ok {
					if percpu, ok := cpuUsage["percpu_usage"].([]interface{}); ok {
						percpuLen = len(percpu)
					}
				}
				cpuPercent = (cpuDelta / systemDelta) * float64(percpuLen) * 100.0
			}
		}
	}

	// Extract memory data
	memUsage := int64(0)
	memLimit := int64(0)
	memPercent := 0.0
	if mem, ok := statsJSON["memory_stats"].(map[string]interface{}); ok {
		if usage, ok := mem["usage"].(float64); ok {
			memUsage = int64(usage)
		}
		if limit, ok := mem["limit"].(float64); ok {
			memLimit = int64(limit)
		}
		if memLimit > 0 {
			memPercent = float64(memUsage) / float64(memLimit) * 100.0
		}
	}

	// Extract network data (moved to separate function for safety)
	networkRx, networkTx := extractNetworkStats(statsJSON)

	// Extract block IO data (moved to separate function for safety)
	blockRead, blockWrite := extractBlockIOStats(statsJSON)

	return ContainerStats{
		CPUPercentage:    cpuPercent,
		MemoryUsage:      memUsage,
		MemoryLimit:      memLimit,
		MemoryPercentage: memPercent,
		NetworkRx:        networkRx,
		NetworkTx:        networkTx,
		BlockRead:        blockRead,
		BlockWrite:       blockWrite,
	}, nil
}

// Helper function to safely extract network stats
func extractNetworkStats(statsJSON map[string]interface{}) (int64, int64) {
	networkRx := int64(0)
	networkTx := int64(0)

	defer func() {
		if r := recover(); r != nil {
			// Silently recover without crashing
		}
	}()

	if networks, ok := statsJSON["networks"].(map[string]interface{}); ok {
		for _, network := range networks {
			if networkStats, ok := network.(map[string]interface{}); ok {
				if rx, ok := networkStats["rx_bytes"].(float64); ok {
					networkRx += int64(rx)
				}
				if tx, ok := networkStats["tx_bytes"].(float64); ok {
					networkTx += int64(tx)
				}
			}
		}
	}

	return networkRx, networkTx
}

// Helper function to safely extract block IO stats
func extractBlockIOStats(statsJSON map[string]interface{}) (int64, int64) {
	blockRead := int64(0)
	blockWrite := int64(0)

	defer func() {
		if r := recover(); r != nil {
			// Silently recover without crashing
		}
	}()

	if blkio, ok := statsJSON["blkio_stats"].(map[string]interface{}); ok {
		if ioServiceBytes, ok := blkio["io_service_bytes_recursive"].([]interface{}); ok {
			for _, entry := range ioServiceBytes {
				if stat, ok := entry.(map[string]interface{}); ok {
					if op, ok := stat["op"].(string); ok {
						if op == "Read" {
							if val, ok := stat["value"].(float64); ok {
								blockRead += int64(val)
							}
						} else if op == "Write" {
							if val, ok := stat["value"].(float64); ok {
								blockWrite += int64(val)
							}
						}
					}
				}
			}
		}
	}

	return blockRead, blockWrite
}

// GetContainerLogs retrieves logs for a container
func (s *Service) GetContainerLogs(ctx context.Context, containerID string) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       "100",
	}

	logs, err := s.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ListImages returns a list of all images
func (s *Service) ListImages(ctx context.Context) ([]ImageInfo, error) {
	images, err := s.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}

	var imageInfos []ImageInfo
	for _, img := range images {
		repoTags := img.RepoTags
		if len(repoTags) == 0 {
			repoTags = []string{"<none>:<none>"}
		}

		id := img.ID
		if len(id) > 12 {
			if len(id) > 7 && id[:7] == "sha256:" {
				id = id[7:19] // Remove "sha256:" prefix and shorten
			} else {
				id = id[:12]
			}
		}

		imageInfos = append(imageInfos, ImageInfo{
			ID:          id,
			RepoTags:    repoTags,
			Size:        img.Size,
			CreatedAt:   time.Unix(img.Created, 0),
			VirtualSize: img.VirtualSize,
		})
	}

	return imageInfos, nil
}

// RemoveImage removes an image
func (s *Service) RemoveImage(ctx context.Context, imageID string, force bool) error {
	options := image.RemoveOptions{
		Force: force,
	}
	_, err := s.client.ImageRemove(ctx, imageID, options)
	return err
}

// InspectImage returns detailed info about an image
func (s *Service) InspectImage(ctx context.Context, imageID string) (string, error) {
	info, _, err := s.client.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ListVolumes returns a list of all volumes
func (s *Service) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volumes, err := s.client.VolumeList(ctx, volume.ListOptions{Filters: filters.Args{}})
	if err != nil {
		return nil, err
	}

	var volumeInfos []VolumeInfo
	for _, vol := range volumes.Volumes {
		volumeInfos = append(volumeInfos, VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
		})
	}

	return volumeInfos, nil
}

// RemoveVolume removes a volume
func (s *Service) RemoveVolume(ctx context.Context, volumeName string, force bool) error {
	return s.client.VolumeRemove(ctx, volumeName, force)
}

// InspectVolume returns detailed info about a volume
func (s *Service) InspectVolume(ctx context.Context, volumeName string) (string, error) {
	info, err := s.client.VolumeInspect(ctx, volumeName)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ListNetworks returns a list of all networks
func (s *Service) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	networks, err := s.client.NetworkList(ctx, network.ListOptions{Filters: filters.Args{}})
	if err != nil {
		return nil, err
	}

	var networkInfos []NetworkInfo
	for _, nw := range networks {
		id := nw.ID
		if len(id) > 12 {
			id = id[:12] // Short ID
		}

		// Convert from Docker network containers to our simplified type
		containers := make(map[string]NetworkContainer)
		for k, v := range nw.Containers {
			containers[k] = NetworkContainer{
				Name:       v.Name,
				EndpointID: v.EndpointID,
			}
		}

		networkInfos = append(networkInfos, NetworkInfo{
			ID:         id,
			Name:       nw.Name,
			Driver:     nw.Driver,
			Scope:      nw.Scope,
			Containers: containers,
		})
	}

	return networkInfos, nil
}

// RemoveNetwork removes a network
func (s *Service) RemoveNetwork(ctx context.Context, networkID string) error {
	return s.client.NetworkRemove(ctx, networkID)
}

// InspectNetwork returns detailed info about a network
func (s *Service) InspectNetwork(ctx context.Context, networkID string) (string, error) {
	info, err := s.client.NetworkInspect(ctx, networkID, network.InspectOptions{})
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// PruneContainers removes all stopped containers
func (s *Service) PruneContainers(ctx context.Context) (uint64, error) {
	report, err := s.client.ContainersPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneImages removes all unused images
func (s *Service) PruneImages(ctx context.Context) (uint64, error) {
	report, err := s.client.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneVolumes removes all unused volumes
func (s *Service) PruneVolumes(ctx context.Context) (uint64, error) {
	report, err := s.client.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PauseContainer pauses a container
func (s *Service) PauseContainer(ctx context.Context, containerID string) error {
	return s.client.ContainerPause(ctx, containerID)
}

// UnpauseContainer unpauses a container
func (s *Service) UnpauseContainer(ctx context.Context, containerID string) error {
	return s.client.ContainerUnpause(ctx, containerID)
}

// KillContainer kills a container
func (s *Service) KillContainer(ctx context.Context, containerID string) error {
	return s.client.ContainerKill(ctx, containerID, "SIGKILL")
}

// InspectContainer returns detailed info about a container
func (s *Service) InspectContainer(ctx context.Context, containerID string) (string, error) {
	info, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// CreateContainer creates a new container with the given configuration
func (s *Service) CreateContainer(ctx context.Context, config ContainerCreateConfig) (string, error) {
	// Pull the image if it doesn't exist
	_, err := s.client.ImagePull(ctx, config.Image, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull image: %v", err)
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image:  config.Image,
		Cmd:    config.Command,
		Env:    config.Env,
		Labels: config.Labels,
	}

	// Prepare host configuration
	hostConfig := &container.HostConfig{
		Binds:       config.Volumes,
		NetworkMode: container.NetworkMode(config.NetworkMode),
		Resources: container.Resources{
			Memory:    config.Memory,
			CPUShares: config.CPUShares,
		},
	}

	// Set restart policy if provided
	if config.Restart != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(config.Restart),
		}
	}

	// Create the container
	resp, err := s.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		config.Name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %v", err)
	}

	return resp.ID, nil
}

// StartContainer starts a container
func (s *Service) StartContainer(ctx context.Context, containerID string) error {
	return s.client.ContainerStart(ctx, containerID, container.StartOptions{})
}

// StopContainer stops a container
func (s *Service) StopContainer(ctx context.Context, containerID string) error {
	timeout := int(10)
	return s.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RestartContainer restarts a container
func (s *Service) RestartContainer(ctx context.Context, containerID string) error {
	timeout := int(10)
	return s.client.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RemoveContainer removes a container
func (s *Service) RemoveContainer(ctx context.Context, containerID string) error {
	return s.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

// Ping checks if the Docker daemon is responding
func (s *Service) Ping(ctx context.Context) (types.Ping, error) {
	return s.client.Ping(ctx)
}
