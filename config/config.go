package config

import (
	"encoding/json"
	"os"
	"time"
)

type ConfigService interface {
	GetNextcloudBaseUrl() string
	GetPaperlessBaseUrl() string
	GetNextcloudUser() string
	GetNextcloudPassword() string
	GetPaperlessToken() string
	GetAuthTokenExpiry() time.Duration
}

type fileConfig struct {
	TokenExpiryHours int `json:"tokenExpiryHours"`
}

type AppConfig struct {
	NextcloudBaseUrl  string
	PaperlessBaseUrl  string
	NextcloudUser     string
	NextcloudPassword string
	PaperlessToken    string
	AuthTokenExpiry   time.Duration
}

type configService struct {
	cfg *AppConfig
}

func NewConfigService() (ConfigService, error) {
	cfg := &AppConfig{
		NextcloudBaseUrl: "http://nextcloud.local/",
		PaperlessBaseUrl: "http://paperless.local/",
		AuthTokenExpiry:  6 * time.Hour,
	}

	if data, err := os.ReadFile("config/config.json"); err == nil {
		var fileCfg fileConfig
		if err := json.Unmarshal(data, &fileCfg); err == nil && fileCfg.TokenExpiryHours > 0 {
			cfg.AuthTokenExpiry = time.Duration(fileCfg.TokenExpiryHours) * time.Hour
		}
	}

	if url := os.Getenv("ONELAB_NEXTCLOUD_BASE_URL"); url != "" {
		cfg.NextcloudBaseUrl = url
	}
	if url := os.Getenv("ONELAB_PAPERLESS_BASE_URL"); url != "" {
		cfg.PaperlessBaseUrl = url
	}

	cfg.NextcloudUser = os.Getenv("ONELAB_NEXTCLOUD_USER")
	cfg.NextcloudPassword = os.Getenv("ONELAB_NEXTCLOUD_PASSWORD")
	cfg.PaperlessToken = os.Getenv("ONELAB_PAPERLESS_TOKEN")

	return &configService{cfg: cfg}, nil
}

func (s *configService) GetNextcloudBaseUrl() string       { return s.cfg.NextcloudBaseUrl }
func (s *configService) GetPaperlessBaseUrl() string       { return s.cfg.PaperlessBaseUrl }
func (s *configService) GetNextcloudUser() string          { return s.cfg.NextcloudUser }
func (s *configService) GetNextcloudPassword() string      { return s.cfg.NextcloudPassword }
func (s *configService) GetPaperlessToken() string         { return s.cfg.PaperlessToken }
func (s *configService) GetAuthTokenExpiry() time.Duration { return s.cfg.AuthTokenExpiry }
