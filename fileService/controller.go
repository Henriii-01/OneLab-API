package fileService

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type FileController struct {
	// A map of available integration services (e.g., "nextcloud", "paperless")
	integrations map[string]IntegrationService
	logger       *slog.Logger
}

func NewFileController(logger *slog.Logger) *FileController {
	return &FileController{
		integrations: make(map[string]IntegrationService),
		logger:       logger,
	}
}

// AddIntegration allows registering more services dynamically
func (c *FileController) AddIntegration(name string, service IntegrationService) {
	c.integrations[name] = service
}

func (c *FileController) GetValueFromBody(r *http.Request, key string) (string, error) {
	// Is Body even there
	if r.Body == nil {
		return "", fmt.Errorf("request body is nil")
	}

	// Read complete body into bytes
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	// We add the body back to the request so it can be read again later if needed (e.g., for file transfer)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Check if body is empty
	if len(bodyBytes) == 0 {
		return "", fmt.Errorf("request body is empty")
	}

	// Parse body as JSON into a map
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return "", fmt.Errorf("failed to decode JSON: %v", err)
	}

	//return the value for the specified key if it exists and is a string
	if value, exists := bodyMap[key]; exists {
		if str, ok := value.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("key '%s' exists but is not a string", key)
	}

	return "", fmt.Errorf("key '%s' not found in body", key)
}

func (c *FileController) GetValueFromQuery(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("missing '%s' in query parameters", key)
	}
	return value, nil
}
func (c *FileController) HTTPTransferHandler(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	source, err := c.GetValueFromBody(r, "source")
	if err != nil {
		http.Error(w, fmt.Sprintf("source query parameter is required: %v", err), http.StatusBadRequest)
		return
	}
	target, err := c.GetValueFromBody(r, "target")
	if err != nil {
		http.Error(w, fmt.Sprintf("target query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	filename, err := c.GetValueFromBody(r, "filename")
	if err != nil {
		http.Error(w, fmt.Sprintf("filename query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	tvc, exists := c.integrations[target]
	if !exists {
		http.Error(w, fmt.Sprintf("Target software '%s' is not supported", target), http.StatusBadRequest)
		return
	}

	svc, exists := c.integrations[source]
	if !exists {
		http.Error(w, fmt.Sprintf("Source software '%s' is not supported", source), http.StatusBadRequest)
		return
	}
	err, statusCode := c.TransferHandler(svc, tvc, filename, r.Context())
	if err != nil {
		c.log(r.Context(), slog.LevelWarn, "file transfer failed", "source", source, "target", target, "status", statusCode, "duration_ms", time.Since(started).Seconds()*1000)
		http.Error(w, fmt.Sprintf("Error transferring file: %v", err), statusCode)
		return
	} else {
		c.log(r.Context(), slog.LevelInfo, "file transfer completed", "source", source, "target", target, "status", statusCode, "duration_ms", time.Since(started).Seconds()*1000)
		w.WriteHeader(statusCode)
		return
	}
}
func (c *FileController) TransferHandler(svc IntegrationService, tvc IntegrationService, filename string, ctx context.Context) (error, int) {
	var fileID string
	foundFiles, err, statusCode := c.LookupHandler(svc, filename, ctx) // Reuse the lookup handler to validate the file exists before transfer
	if err != nil {
		return err, statusCode
	}

	// Check if foundFiles is a slice and has more than one element using reflection
	switch files := foundFiles.(type) {
	case []NextcloudFileResponse:

		if len(files) == 0 {
			return fmt.Errorf("file '%s' not found in source", filename), http.StatusNotFound
		}
		if len(files) > 1 {
			return fmt.Errorf("multiple files found with name '%s' in source, please specify more precise filename", filename), http.StatusBadRequest
		}
		fileID = files[0].Path
	case []PaperlessFileResponse:
		if len(files) == 0 {
			return fmt.Errorf("file '%s' not found in source", filename), http.StatusNotFound
		}
		if len(files) > 1 {
			return fmt.Errorf("multiple files found with name '%s' in source, please specify more precise filename", filename), http.StatusBadRequest
		}
		fileID = fmt.Sprintf("%d", files[0].ID)
	default:
		return fmt.Errorf("unsupported response type from source lookup"), http.StatusInternalServerError
	}

	file, err := svc.GetFile(ctx, fileID)
	if err != nil {
		return fmt.Errorf("error retrieving file content: %v", err), http.StatusInternalServerError
	}

	err = tvc.AddFile(ctx, file, filename)
	if err != nil {
		return fmt.Errorf("error adding file to target: %v", err), http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

// CheckIntegrationsStatus verifies the connectivity of all registered integrations.
func (c *FileController) CheckIntegrationsStatus(ctx context.Context) map[string]string {
	statusMap := make(map[string]string)
	for name, svc := range c.integrations {
		if err := svc.CheckStatus(ctx); err != nil {
			c.log(ctx, slog.LevelWarn, "integration health check failed", "integration", name)
			statusMap[name] = fmt.Sprintf("DOWN (%v)", err.Error())
		} else {
			c.log(ctx, slog.LevelDebug, "integration health check passed", "integration", name)
			statusMap[name] = "OK"
		}
	}
	return statusMap
}
func (c *FileController) HTTPLookupHandler(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	source, err := c.GetValueFromQuery(r, "source")
	if err != nil {
		http.Error(w, fmt.Sprintf("source query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	filename, err := c.GetValueFromQuery(r, "filename")
	if err != nil {
		http.Error(w, fmt.Sprintf("filename query parameter is required: %v", err), http.StatusBadRequest)
		return
	}

	sourceSoft := strings.ToLower(source)
	svc, exists := c.integrations[sourceSoft]
	if !exists {
		http.Error(w, fmt.Sprintf("source software '%s' is not supported", sourceSoft), http.StatusBadRequest)
		return
	}

	result, err, statusCode := c.LookupHandler(svc, filename, r.Context())
	if err != nil {
		c.log(r.Context(), slog.LevelWarn, "file lookup failed", "source", sourceSoft, "status", statusCode, "duration_ms", time.Since(started).Seconds()*1000)
		http.Error(w, fmt.Sprintf("Error looking up file from %s: %v", sourceSoft, err), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		c.log(r.Context(), slog.LevelError, "file lookup response encoding failed", "source", sourceSoft, "status", http.StatusInternalServerError)
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
	c.log(r.Context(), slog.LevelInfo, "file lookup completed", "source", sourceSoft, "status", statusCode, "duration_ms", time.Since(started).Seconds()*1000)

}
func (c *FileController) LookupHandler(svc IntegrationService, filename string, ctx context.Context) (any, error, int) {

	// Check if the service implements the FileLookuper interface
	if lookuper, ok := svc.(FileLookuper); ok {
		result, err := lookuper.LookupFile(ctx, filename)
		if err != nil {
			return nil, fmt.Errorf("Error looking up file from: %v", err), http.StatusInternalServerError
		}

		return result, nil, http.StatusOK

	} else {
		return nil, fmt.Errorf("lookup not supported for source"), http.StatusBadRequest
	}
}
func (c *FileController) HandlePaperlessBackup(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	paperlessSvc, exists := c.integrations["paperless"]
	if !exists {
		http.Error(w, "paperless integration is not configured", http.StatusNotFound)
		return
	}

	backupSvc, ok := paperlessSvc.(PaperlessBackupService)
	if !ok {
		http.Error(w, "paperless backup is not supported", http.StatusNotImplemented)
		return
	}

	path, err := backupSvc.Backup(r.Context())
	if err != nil {
		c.log(r.Context(), slog.LevelError, "paperless backup failed", "duration_ms", time.Since(started).Seconds()*1000)
		http.Error(w, fmt.Sprintf("failed to create paperless backup: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"path":     path,
		"filename": filepath.Base(path),
	}); err != nil {
		c.log(r.Context(), slog.LevelError, "paperless backup response encoding failed")
		http.Error(w, fmt.Sprintf("failed to encode backup response: %v", err), http.StatusInternalServerError)
		return
	}
	c.log(r.Context(), slog.LevelInfo, "paperless backup completed", "duration_ms", time.Since(started).Seconds()*1000)
}

func (c *FileController) log(ctx context.Context, level slog.Level, message string, args ...any) {
	if c.logger != nil {
		c.logger.Log(ctx, level, message, args...)
	}
}

func (c *FileController) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {

	transferHandler := http.HandlerFunc(c.HTTPTransferHandler)
	lookupHandler := http.HandlerFunc(c.HTTPLookupHandler)
	backupHandler := http.HandlerFunc(c.HandlePaperlessBackup)

	mux.Handle("POST /api/v1/files/transfer/", authMiddleware(transferHandler))
	mux.Handle("GET /api/v1/files/lookup/", authMiddleware(lookupHandler))
	mux.Handle("GET /api/v1/paperless/backup", authMiddleware(backupHandler))
}
