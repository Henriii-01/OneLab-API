package fileService

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

// PaperlessService handles sending files to Paperless-ngx.
type PaperlessService struct {
	apiURL   string
	checkURL string
	token    string
}

func NewPaperlessService(baseURL, token string) *PaperlessService {
	apiURL, _ := url.JoinPath(baseURL, "api/documents/post_document/")
	checkURL, _ := url.JoinPath(baseURL, "api/documents/")
	return &PaperlessService{
		apiURL:   apiURL,
		checkURL: checkURL,
		token:    token,
	}
}

func (s *PaperlessService) TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error {
	// TODO: Implement Paperless-ngx document consumption logic here.
	fmt.Printf("Transferring file '%s' to Paperless-ngx at %s\n", header.Filename, s.apiURL)
	return nil
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
