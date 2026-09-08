package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/HTMLuke/OneLab-API/auth"
	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/logging"
	"github.com/HTMLuke/OneLab-API/secretProvider"
	"github.com/joho/godotenv"
)

type Response struct {
	Message      string            `json:"message"`
	Status       string            `json:"status"`
	Integrations map[string]string `json:"integrations,omitempty"`
}

func main() {
	logger := logging.New()

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			logger.Warn("failed to load environment file", "error", err)
		}
	}

	// Initialize Config Service
	cfgService, err := config.NewConfigService(logger)
	if err != nil {
		logger.Warn("failed to load config file; using defaults and environment", "error", err)
	}
	secretService := secretProvider.NewSecretService(logger)
	mux := http.NewServeMux()

	// Build controllers and wire in every integration enabled in config
	authController := auth.NewAuthController(logger)
	fController := fileService.NewFileController(logger)
	registerIntegrations(cfgService, secretService, fController, authController, logger)
	authController.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		res := Response{
			Message: "Healthy",
			Status:  "UP",
		}
		json.NewEncoder(w).Encode(res)
	})

	fController.RegisterRoutes(mux, authController.Middleware)

	logger.Info("server starting", "address", ":8080")
	if err := http.ListenAndServe(":8080", logging.AccessLog(logger, mux)); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
