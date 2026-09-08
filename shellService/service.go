package shellService

import "context"

// ShellService describes shell-like command execution backends.
// This package intentionally does not expose HTTP routes; it only provides the service functions.
type ShellService interface {
	Execute(ctx context.Context, containerID string, command string) (string, error)
	ExecuteWithArgs(ctx context.Context, containerID string, command string, args ...string) (string, error)
}
