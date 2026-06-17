package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/HTMLuke/OneLab-API/auth"
	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/fileService"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

type Response struct {
	Message      string            `json:"message"`
	Status       string            `json:"status"`
	Integrations map[string]string `json:"integrations,omitempty"`
}

func main() {
	// Initialize Config Service
	cfgService, err := config.NewConfigService()
	if err != nil {
		log.Printf("Warning: Failed to load config file (falling back to defaults & env): %v", err)
	}
	secretService := secretProvider.NewSecretService()
	mux := http.NewServeMux()

	// Build controllers and wire in every integration enabled in config
	authController := auth.NewAuthController()
	fController := fileService.NewFileController()
	registerIntegrations(cfgService, secretService, fController, authController)
	authController.RegisterRoutes(mux)

	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check the status of all registered integrations
		integrationStatuses := fController.CheckIntegrationsStatus(r.Context())

		res := Response{
			Message:      "OneAPI running!",
			Status:       "OK",
			Integrations: integrationStatuses,
		}
		json.NewEncoder(w).Encode(res)
	})

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

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
