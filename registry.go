package main

import (
	"log"

	"github.com/HTMLuke/OneLab-API/auth"
	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// registerIntegrations wires every enabled integration into its controller
// Integrations register themselves from their own files via their init()
func registerIntegrations(cfg config.ConfigService, s secretProvider.SecretService, fc *fileService.FileController, ac *auth.AuthController) {
	for name, build := range fileService.Builders() {
		if !cfg.IsServiceEnabled(name) {
			continue
		}
		svc, err := build(cfg.GetServiceURL(name), s)
		if err != nil {
			log.Printf("Warning: service '%s' is enabled but failed to initialize (%v), skipping", name, err)
			continue
		}
		fc.AddIntegration(name, svc)
		log.Printf("service integration enabled: %s", name)
	}

	for name, build := range auth.Builders() {
		if !cfg.IsAuthEnabled(name) {
			continue
		}
		svc, err := build(cfg.GetAuthExpiry(), s)
		if err != nil {
			log.Printf("Warning: auth '%s' is enabled but failed to initialize (%v), skipping", name, err)
			continue
		}
		ac.AddIntegration(name, svc)
		log.Printf("auth integration enabled: %s", name)
	}
}
