package shellService

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// DockerCmdService executes commands inside Docker containers via the Docker CLI.
type DockerCmdService struct {
	logger *slog.Logger
}

func init() {
	RegisterService("dockerCmd", func(logger *slog.Logger) (ShellService, error) {
		return NewDockerCmdService(logger), nil
	})
}

func NewDockerCmdService(logger *slog.Logger) *DockerCmdService {
	return &DockerCmdService{logger: logger}
}

// Execute runs a command inside the given container.
// If a shell expression is passed (e.g. "ls -la"), it is executed through sh -lc.
func (s *DockerCmdService) Execute(ctx context.Context, containerID string, command string) (string, error) {
	return s.ExecuteWithArgs(ctx, containerID, command)
}

// ExecuteWithArgs runs a command plus explicit arguments directly inside the container.
func (s *DockerCmdService) ExecuteWithArgs(ctx context.Context, containerID string, command string, args ...string) (string, error) {
	containerID = strings.TrimSpace(containerID)
	command = strings.TrimSpace(command)
	if containerID == "" {
		return "", fmt.Errorf("container ID is required")
	}
	if command == "" {
		return "", fmt.Errorf("command is required")
	}
	started := time.Now()
	if s.logger != nil {
		s.logger.Debug("docker command started", "container", containerID, "argument_count", len(args))
	}

	dockerArgs := []string{"exec", "-i", containerID}
	if len(args) > 0 {
		dockerArgs = append(dockerArgs, command)
		dockerArgs = append(dockerArgs, args...)
	} else if strings.ContainsAny(command, " \t\n\r") {
		dockerArgs = append(dockerArgs, "sh", "-lc", command)
	} else {
		dockerArgs = append(dockerArgs, command)
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("docker command failed", "container", containerID, "argument_count", len(args), "duration_ms", time.Since(started).Seconds()*1000)
		}
		if len(output) > 0 {
			return string(output), fmt.Errorf("docker command failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("docker command failed: %w", err)
	}

	if s.logger != nil {
		s.logger.Debug("docker command completed", "container", containerID, "argument_count", len(args), "duration_ms", time.Since(started).Seconds()*1000)
	}
	return strings.TrimSpace(string(output)), nil
}
