package fileService

import (
	"context"
	"fmt"
	"net/http"
)

type FileController struct {
	// A map of available integration services (e.g., "nextcloud", "paperless")
	integrations map[string]IntegrationService
}

func NewFileController() *FileController {
	return &FileController{
		integrations: make(map[string]IntegrationService),
	}
}

// AddIntegration allows registering more services dynamically
func (c *FileController) AddIntegration(name string, service IntegrationService) {
	c.integrations[name] = service
}

func (c *FileController) TransferHandler(w http.ResponseWriter, r *http.Request) {
	// Using Go 1.22+ PathValue to extract the targeted software
	targetSoft := r.PathValue("target")

	svc, exists := c.integrations[targetSoft]
	if !exists {
		http.Error(w, fmt.Sprintf("Target software '%s' is not supported", targetSoft), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Let the specific integration handle the logic
	if err := svc.TransferFile(r.Context(), file, header); err != nil {
		http.Error(w, fmt.Sprintf("Transfer to %s failed", targetSoft), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("File %s successfully transferred to %s\n", header.Filename, targetSoft)))
}

// CheckIntegrationsStatus verifies the connectivity of all registered integrations.
func (c *FileController) CheckIntegrationsStatus(ctx context.Context) map[string]string {
	statusMap := make(map[string]string)
	for name, svc := range c.integrations {
		if err := svc.CheckStatus(ctx); err != nil {
			statusMap[name] = fmt.Sprintf("DOWN (%v)", err.Error())
		} else {
			statusMap[name] = "OK"
		}
	}
	return statusMap
}

func (c *FileController) RegisterRoutes(mux *http.ServeMux) {
	// e.g. POST /api/v1/files/transfer/nextcloud
	mux.HandleFunc("POST /api/v1/files/transfer/{target}", c.TransferHandler)
}
