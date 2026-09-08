package fileService

import (
	"log/slog"

	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// IntegrationBuilder constructs one service integration from the provided base URL and the shared secret
type IntegrationBuilder func(baseURL string, secrets secretProvider.SecretService, logger *slog.Logger) (IntegrationService, error)

// builders holds every known service integration keyed by name
// each service file adds itself here from its own init()
var builders = map[string]IntegrationBuilder{}

// RegisterIntegration records a service integration
// call from service file init()
func RegisterIntegration(name string, build IntegrationBuilder) {
	builders[name] = build
}

// Builders returns all registered service integrations
func Builders() map[string]IntegrationBuilder { return builders }
