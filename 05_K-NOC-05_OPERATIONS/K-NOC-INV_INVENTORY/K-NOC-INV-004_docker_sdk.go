// Package noc provides NOC operations tooling.
// K-NOC-INV-004 — Docker Inventory: enumerate Docker containers, images, and volumes
// using the docker CLI via exec, since docker/docker SDK is not in go.mod as a direct dep.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	nats "github.com/nats-io/nats.go"
)

// PortBinding pairs a host port to a container port.
type PortBinding struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ContainerInfo holds summary data about a running or stopped container.
type ContainerInfo struct {
	ID      string            `json:"ID"`
	Name    string            `json:"Name"`
	Image   string            `json:"Image"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Created string            `json:"Created"`
	Ports   []PortBinding     `json:"Ports"`
	Labels  map[string]string `json:"Labels"`
}

// ImageInfo holds summary data about a Docker image.
type ImageInfo struct {
	ID       string   `json:"ID"`
	RepoTags []string `json:"RepoTags"`
	Size     int64    `json:"Size"`
	Created  string   `json:"Created"`
}

// ContainerStats holds resource usage metrics for a container.
type ContainerStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsage int64   `json:"memory_usage"`
	MemoryLimit int64   `json:"memory_limit"`
	NetworkRx   int64   `json:"network_rx"`
	NetworkTx   int64   `json:"network_tx"`
}

// ContainerDetail holds detailed inspect output for a single container.
type ContainerDetail struct {
	ID          string            `json:"Id"`
	Name        string            `json:"Name"`
	Image       string            `json:"Image"`
	Status      string            `json:"Status"`
	IPAddress   string            `json:"IPAddress"`
	Hostname    string            `json:"Hostname"`
	Env         []string          `json:"Env"`
	Labels      map[string]string `json:"Labels"`
	Mounts      []string          `json:"Mounts"`
	RestartCount int              `json:"RestartCount"`
}

// DockerInventory enumerates Docker resources on the local host.
type DockerInventory struct {
	dockerHost string
}

// NewDockerInventory reads DOCKER_HOST from the environment.
func NewDockerInventory() *DockerInventory {
	return &DockerInventory{
		dockerHost: os.Getenv("DOCKER_HOST"),
	}
}

func (d *DockerInventory) dockerCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "docker", args...)
	if d.dockerHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+d.dockerHost)
	}
	return cmd
}

// ListContainers returns all containers (running and stopped).
func (d *DockerInventory) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	// docker ps -a --format json outputs one JSON object per line.
	cmd := d.dockerCmd(ctx, "ps", "-a", "--format", "{{json .}}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker ps: %w — stderr: %s", err, stderr.String())
	}

	var containers []ContainerInfo
	dec := json.NewDecoder(&stdout)
	for dec.More() {
		var c ContainerInfo
		if err := dec.Decode(&c); err != nil {
			break
		}
		containers = append(containers, c)
	}
	return containers, nil
}

// ListImages returns all locally available Docker images.
func (d *DockerInventory) ListImages(ctx context.Context) ([]ImageInfo, error) {
	cmd := d.dockerCmd(ctx, "images", "--format", "{{json .}}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker images: %w — stderr: %s", err, stderr.String())
	}

	var images []ImageInfo
	dec := json.NewDecoder(&stdout)
	for dec.More() {
		var img ImageInfo
		if err := dec.Decode(&img); err != nil {
			break
		}
		images = append(images, img)
	}
	return images, nil
}

// GetContainerStats returns resource usage stats for a running container.
func (d *DockerInventory) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	cmd := d.dockerCmd(ctx, "stats", "--no-stream", "--format", "{{json .}}", containerID)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats %s: %w", containerID, err)
	}

	var raw struct {
		CPUPerc string `json:"CPUPerc"`
		MemUsage string `json:"MemUsage"`
	}
	if parseErr := json.Unmarshal(out, &raw); parseErr != nil {
		return nil, fmt.Errorf("parse docker stats: %w", parseErr)
	}

	return &ContainerStats{
		CPUPercent: parseDockerPercent(raw.CPUPerc),
	}, nil
}

// InspectContainer returns the full inspect output for a container.
func (d *DockerInventory) InspectContainer(ctx context.Context, containerID string) (*ContainerDetail, error) {
	cmd := d.dockerCmd(ctx, "inspect", "--format", "{{json .}}", containerID)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect %s: %w", containerID, err)
	}

	// docker inspect of a single container wraps it in an array.
	var details []struct {
		ID   string `json:"Id"`
		Name string `json:"Name"`
		Config struct {
			Image    string            `json:"Image"`
			Hostname string            `json:"Hostname"`
			Env      []string          `json:"Env"`
			Labels   map[string]string `json:"Labels"`
		} `json:"Config"`
		State struct {
			Status string `json:"Status"`
		} `json:"State"`
		RestartCount int `json:"RestartCount"`
		NetworkSettings struct {
			IPAddress string `json:"IPAddress"`
		} `json:"NetworkSettings"`
	}

	if parseErr := json.Unmarshal(out, &details); parseErr != nil {
		return nil, fmt.Errorf("parse docker inspect: %w", parseErr)
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("no inspect data for %s", containerID)
	}

	det := details[0]
	return &ContainerDetail{
		ID:           det.ID,
		Name:         det.Name,
		Image:        det.Config.Image,
		Status:       det.State.Status,
		IPAddress:    det.NetworkSettings.IPAddress,
		Hostname:     det.Config.Hostname,
		Env:          det.Config.Env,
		Labels:       det.Config.Labels,
		RestartCount: det.RestartCount,
	}, nil
}

// PublishInventory publishes container inventory events to NATS.
func (d *DockerInventory) PublishInventory(ctx context.Context, tenantID string, nc *nats.Conn) {
	containers, err := d.ListContainers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[docker-inv] list containers error: %v\n", err)
		return
	}

	type inventoryEvent struct {
		TenantID   string          `json:"tenant_id"`
		Containers []ContainerInfo `json:"containers"`
		SampledAt  time.Time       `json:"sampled_at"`
	}

	evt := inventoryEvent{
		TenantID:   tenantID,
		Containers: containers,
		SampledAt:  time.Now().UTC(),
	}

	data, err := json.Marshal(evt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[docker-inv] marshal error: %v\n", err)
		return
	}

	subject := fmt.Sprintf("kubric.%s.asset.provisioned.v1", tenantID)
	if pubErr := nc.Publish(subject, data); pubErr != nil {
		fmt.Fprintf(os.Stderr, "[docker-inv] nats publish error subject=%s err=%v\n", subject, pubErr)
	}
}

// parseDockerPercent converts a docker stats CPU percentage string like "12.34%" to float64.
func parseDockerPercent(s string) float64 {
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			f := 0.0
			fmt.Sscanf(s[:i], "%f", &f)
			return f
		}
	}
	return 0
}
