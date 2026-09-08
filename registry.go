package main

import (
	"log/slog"

	"github.com/HTMLuke/OneLab-API/auth"
	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// registerIntegrations wires every enabled integration into its controller
// Integrations register themselves from their own files via their init()
func registerIntegrations(cfg config.ConfigService, s secretProvider.SecretService, fc *fileService.FileController, ac *auth.AuthController, logger *slog.Logger) {
	for name, build := range fileService.Builders() {
		if !cfg.IsServiceEnabled(name) {
			continue
		}
		svc, err := build(cfg.GetServiceURL(name), s, logger)
		if err != nil {
			logger.Warn("service integration failed to initialize", "integration", name, "error", err)
			continue
		}
		fc.AddIntegration(name, svc)
		logger.Info("service integration enabled", "integration", name)
	}

	for name, build := range auth.Builders() {
		if !cfg.IsAuthEnabled(name) {
			continue
		}
		svc, err := build(cfg.GetAuthExpiry(), s, logger)
		if err != nil {
			logger.Warn("auth integration failed to initialize", "integration", name, "error", err)
			continue
		}
		ac.AddIntegration(name, svc)
		logger.Info("auth integration enabled", "integration", name)
	}
}
