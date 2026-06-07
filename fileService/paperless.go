package fileService

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// PaperlessService handles sending files to Paperless-ngx.
type PaperlessService struct {
	apiURL        string
	checkURL      string
	token         string
	cnfService    config.ConfigService
	secretService secretProvider.SecretService
}

func NewPaperlessService(cfgService config.ConfigService, secretService secretProvider.SecretService) *PaperlessService {
	baseURL := cfgService.GetPaperlessBaseUrl()
	token := secretService.GetPaperlessToken()
	apiURL, _ := url.JoinPath(baseURL, "api/documents/post_document/")
	checkURL, _ := url.JoinPath(baseURL, "api/documents/")
	return &PaperlessService{
		apiURL:        apiURL,
		checkURL:      checkURL,
		token:         token,
		cnfService:    cfgService,
		secretService: secretService,
	}
}

func (s *PaperlessService) TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error {
	// TODO: Implement Paperless-ngx document consumption logic here.
	fmt.Printf("Transferring file '%s' to Paperless-ngx at %s\n", header.Filename, s.apiURL)
	return nil
}

func (s *PaperlessService) LookupFile(ctx context.Context, filename string) (any, error) {
	// TODO: Implement Paperless-ngx file lookup logic here.
	return nil, fmt.Errorf("LookupFile not implemented yet for Paperless")
}

func (s *PaperlessService) CheckStatus(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.checkURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("paperless authentication failed with status: %d", resp.StatusCode)
	}

	return nil
}
