package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"
)

type ConfigService interface {
	GetAuthExpiry() time.Duration
	IsServiceEnabled(name string) bool
	GetServiceURL(name string) string
	IsAuthEnabled(name string) bool
}

// ServiceConfig is the per-service integration config from config.json
type ServiceConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

type fileConfig struct {
	AuthExpiryHours int `json:"authExpiryHours"`
	Integrations    struct {
		Services map[string]ServiceConfig `json:"services"`
		Auth     map[string]bool          `json:"auth"`
	} `json:"integrations"`
}

type AppConfig struct {
	AuthExpiry  time.Duration
	Services    map[string]ServiceConfig
	EnabledAuth map[string]bool
}

type configService struct {
	cfg    *AppConfig
	logger *slog.Logger
}

func NewConfigService(logger *slog.Logger) (ConfigService, error) {
	cfg := &AppConfig{
		AuthExpiry:  6 * time.Hour,
		Services:    map[string]ServiceConfig{},
		EnabledAuth: map[string]bool{},
	}

	if data, err := os.ReadFile("config/config.json"); err == nil {
		var fileCfg fileConfig
		if err := json.Unmarshal(data, &fileCfg); err == nil {
			if fileCfg.AuthExpiryHours > 0 {
				cfg.AuthExpiry = time.Duration(fileCfg.AuthExpiryHours) * time.Hour
			}
			if fileCfg.Integrations.Services != nil {
				cfg.Services = fileCfg.Integrations.Services
			}
			if fileCfg.Integrations.Auth != nil {
				cfg.EnabledAuth = fileCfg.Integrations.Auth
			}
			if logger != nil {
				logger.Info("configuration loaded", "path", "config/config.json")
			}
		} else if logger != nil {
			logger.Warn("configuration file is invalid; using defaults and environment", "path", "config/config.json", "error", err)
		}
	} else if logger != nil {
		logger.Debug("configuration file not found; using defaults and environment", "path", "config/config.json")
	}

	return &configService{cfg: cfg, logger: logger}, nil
}

func (s *configService) GetAuthExpiry() time.Duration { return s.cfg.AuthExpiry }

// IsServiceEnabled tracks whether a service integration is enabled or disabled
// unknown names default to false
func (s *configService) IsServiceEnabled(name string) bool { return s.cfg.Services[name].Enabled }

// GetServiceURL returns the configured base URL for a service integration.
// If the config file omits a URL, we fall back to the matching env var.
// unknown names default to ""
func (s *configService) GetServiceURL(name string) string {
	if service, ok := s.cfg.Services[name]; ok && strings.TrimSpace(service.URL) != "" {
		return service.URL
	}

	envName := strings.ToUpper(strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(name))
	for _, key := range []string{
		"ONELAB_" + envName + "_BASE_URL",
		"ONELAB_" + envName + "_URL",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	if service, ok := s.cfg.Services[name]; ok {
		return service.URL
	}
	return ""
}

// IsAuthEnabled tracks whether an auth integration is enabled or disabled
// unknown names default to false
func (s *configService) IsAuthEnabled(name string) bool { return s.cfg.EnabledAuth[name] }
