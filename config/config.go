package config

import (
	"encoding/json"
	"os"
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
	cfg *AppConfig
}

func NewConfigService() (ConfigService, error) {
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
		}
	}

	return &configService{cfg: cfg}, nil
}

func (s *configService) GetAuthExpiry() time.Duration { return s.cfg.AuthExpiry }

// IsServiceEnabled tracks whether a service integration is enabled or disabled
// unknown names default to false
func (s *configService) IsServiceEnabled(name string) bool { return s.cfg.Services[name].Enabled }

// GetServiceURL returns the configured base URL for a service integration
// unknown names default to ""
func (s *configService) GetServiceURL(name string) string { return s.cfg.Services[name].URL }

// IsAuthEnabled tracks whether an auth integration is enabled or disabled
// unknown names default to false
func (s *configService) IsAuthEnabled(name string) bool { return s.cfg.EnabledAuth[name] }
