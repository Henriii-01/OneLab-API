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

	// Initialize Auth Service
	jwtSvc := auth.NewJwtService(
		cfgService.GetAuthTokenExpiry(),
		cfgService.GetJwtSecret(),
	)

	jwtSvc.NewJwtClient(
		cfgService.GetAuthClientID(),
		cfgService.GetAuthClientSecret(),
	)

	authController := auth.NewAuthController(jwtSvc)
	authController.RegisterRoutes(mux)

	// Initialize App Controller and specify exactly what Services are available.
	fController := fileService.NewFileController()

	// Inject integrations securely configured with their base URLs and credentials from the config service
	fController.AddIntegration("nextcloud", fileService.NewNextcloudService(cfgService, secretService))
	fController.AddIntegration("paperless", fileService.NewPaperlessService(cfgService, secretService))

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

	fController.RegisterRoutes(mux)

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
