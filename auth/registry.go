package auth

import (
	"log/slog"
	"time"

	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// AuthBuilder constructs one auth integration from the configured auth expiry and shared secret
type AuthBuilder func(expiry time.Duration, secrets secretProvider.SecretService, logger *slog.Logger) (AuthService, error)

// builders holds every known auth integration keyed by name
// each service file adds itself here from its own init()
var builders = map[string]AuthBuilder{}

// RegisterAuth records an auth integration
// call from service file init()
func RegisterAuth(name string, build AuthBuilder) {
	builders[name] = build
}

// Builders returns all registered auth integrations
func Builders() map[string]AuthBuilder { return builders }
