package fileService

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

// decodeJSON is a general helper to parse a JSON request body into a provided object
func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func (c *FileController) GetValueFromBody(r *http.Request, key string) (string, error) {
	var bodyMap map[string]interface{}
	if err := decodeJSON(r, &bodyMap); err != nil {
		return "", err
	}
	if value, exists := bodyMap[key]; exists {
		if str, ok := value.(string); ok {
			return str, nil
		}
	}
	return "", fmt.Errorf("key '%s' not found or not a string", key)
}

func (c *FileController) GetValueFromQuery(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("missing '%s' in query parameters", key)
	}
	return value, nil
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
	if transferer, ok := svc.(FileTransferer); ok {
		if err := transferer.TransferFile(r.Context(), file, header); err != nil {
			http.Error(w, fmt.Sprintf("Transfer to %s failed", targetSoft), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("File %s successfully transferred to %s\n", header.Filename, targetSoft)))
	} else {
		http.Error(w, fmt.Sprintf("File transfer not supported for target '%s'", targetSoft), http.StatusBadRequest)
	}
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
func (c *FileController) LookupHandler(w http.ResponseWriter, r *http.Request) {

	source, err := c.GetValueFromQuery(r, "source")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filename, err := c.GetValueFromQuery(r, "filename")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sourceSoft := strings.ToLower(source)
	svc, exists := c.integrations[sourceSoft]
	if !exists {
		http.Error(w, fmt.Sprintf("Source software '%s' is not supported", sourceSoft), http.StatusBadRequest)
		return
	}

	// Check if the service implements the FileLookuper interface
	if lookuper, ok := svc.(FileLookuper); ok {
		result, err := lookuper.LookupFile(r.Context(), filename)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error looking up file from %s: %v", sourceSoft, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, fmt.Sprintf("Lookup not supported for source '%s'", sourceSoft), http.StatusBadRequest)
	}
}
func (c *FileController) RegisterRoutes(mux *http.ServeMux) {
	// e.g. POST /api/v1/files/transfer/nextcloud
	mux.HandleFunc("POST /api/v1/files/transfer/{target}", c.TransferHandler)
	mux.HandleFunc("GET /api/v1/files/lookup/", c.LookupHandler)
}
