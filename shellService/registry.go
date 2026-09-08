package shellService

import "log/slog"

// ServiceBuilder builds a shell service implementation.
type ServiceBuilder func(logger *slog.Logger) (ShellService, error)

// builders holds every known shell service keyed by name.
var builders = map[string]ServiceBuilder{}

// RegisterService records a shell service implementation.
func RegisterService(name string, build ServiceBuilder) {
	builders[name] = build
}

// Builders returns all registered shell services.
func Builders() map[string]ServiceBuilder { return builders }
