package config

import (
	"encoding/json"
	"os"
	"time"
)

type ConfigService interface {
	GetAuthExpiry() time.Duration
	IsServiceEnabled(name string) bool
	IsAuthEnabled(name string) bool
}

type fileConfig struct {
	AuthExpiryHours int `json:"authExpiryHours"`
	Integrations    struct {
		EnabledServices map[string]bool `json:"services"`
		EnabledAuth     map[string]bool `json:"auth"`
	} `json:"integrations"`
}

type AppConfig struct {
	AuthExpiry      time.Duration
	EnabledServices map[string]bool
	EnabledAuth     map[string]bool
}

type configService struct {
	cfg *AppConfig
}

func NewConfigService() (ConfigService, error) {
	cfg := &AppConfig{
		AuthExpiry:      6 * time.Hour,
		EnabledServices: map[string]bool{},
		EnabledAuth:     map[string]bool{},
	}

	if data, err := os.ReadFile("config/config.json"); err == nil {
		var fileCfg fileConfig
		if err := json.Unmarshal(data, &fileCfg); err == nil {
			if fileCfg.AuthExpiryHours > 0 {
				cfg.AuthExpiry = time.Duration(fileCfg.AuthExpiryHours) * time.Hour
			}
			if fileCfg.Integrations.EnabledServices != nil {
				cfg.EnabledServices = fileCfg.Integrations.EnabledServices
			}
			if fileCfg.Integrations.EnabledAuth != nil {
				cfg.EnabledAuth = fileCfg.Integrations.EnabledAuth
			}
		}
	}

	return &configService{cfg: cfg}, nil
}

func (s *configService) GetAuthExpiry() time.Duration { return s.cfg.AuthExpiry }

// IsServiceEnabled tracks whether a service integration is enabled or disabled
// unknown names default to false
func (s *configService) IsServiceEnabled(name string) bool { return s.cfg.EnabledServices[name] }

// IsAuthEnabled tracks whether an auth integration is enabled or disabled
// unknown names default to false
func (s *configService) IsAuthEnabled(name string) bool { return s.cfg.EnabledAuth[name] }
