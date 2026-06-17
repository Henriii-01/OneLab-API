package main

import (
	"log"

	"github.com/HTMLuke/OneLab-API/auth"
	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// entry is one toggleable integration.
type entry[T any] struct {
	name  string
	build func() (T, error)
}

// register wires every enabled entry into the controller via add skipping disabled or failed entries and prints a warning
func register[T any](kind string, enabled func(string) bool, add func(string, T), entries []entry[T]) {
	for _, e := range entries {
		if !enabled(e.name) {
			continue
		}
		svc, err := e.build()
		if err != nil {
			log.Printf("Warning: %s '%s' is enabled but failed to initialize (%v), skipping", kind, e.name, err)
			continue
		}
		add(e.name, svc)
		log.Printf("%s integration enabled: %s", kind, e.name)
	}
}

// registerIntegrations is the single source of truth for all known integrations
func registerIntegrations(cfg config.ConfigService, s secretProvider.SecretService, fc *fileService.FileController, ac *auth.AuthController) {
	register("service", cfg.IsServiceEnabled, fc.AddIntegration, []entry[fileService.IntegrationService]{
		{"nextcloud", func() (fileService.IntegrationService, error) {
			return fileService.NewNextcloudService(cfg.GetServiceURL("nextcloud"), s)
		}},
		{"paperless", func() (fileService.IntegrationService, error) {
			return fileService.NewPaperlessService(cfg.GetServiceURL("paperless"), s)
		}},
	})
	register("auth", cfg.IsAuthEnabled, ac.AddIntegration, []entry[auth.AuthService]{
		{"jwt", func() (auth.AuthService, error) { return auth.NewJwtService(cfg.GetAuthExpiry(), s) }},
	})
}
