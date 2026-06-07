package fileService

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

// NextcloudService handles sending files to Nextcloud.
type NextcloudService struct {
	apiURL   string
	username string
	password string
}

func NewNextcloudService(baseURL, username, password string) *NextcloudService {
	apiURL, _ := url.JoinPath(baseURL, "remote.php/webdav/")
	return &NextcloudService{
		apiURL:   apiURL,
		username: username,
		password: password,
	}
}

func (s *NextcloudService) TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error {
	// TODO: Implement Nextcloud WebDAV or API upload logic here.
	fmt.Printf("Transferring file '%s' to Nextcloud at %s\n", header.Filename, s.apiURL)
	return nil
}

func (s *NextcloudService) CheckStatus(ctx context.Context) error {
	// simple request to the WebDAV endpoint.
	req, err := http.NewRequestWithContext(ctx, "GET", s.apiURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.username, s.password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("nextcloud authentication failed with status: %d", resp.StatusCode)
	}

	return nil
}
