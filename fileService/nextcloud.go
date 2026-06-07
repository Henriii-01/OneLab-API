package fileService

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/HTMLuke/OneLab-API/config"
	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// NextcloudService handles sending files to Nextcloud.
type NextcloudService struct {
	apiLookupUrl  string
	username      string
	password      string
	cnfService    config.ConfigService
	secretService secretProvider.SecretService
}

type NextcloudFileMetadata struct {
	ThumbnailURL string `json:"thumbnailUrl"`
	Title        string `json:"title"`
	Subline      string `json:"subline"`
	ResourceURL  string `json:"resourceUrl"`
	Icon         string `json:"icon"`
	Rounded      bool   `json:"rounded"`
	FileID       string `json:"fileId"`
	Path         string `json:"path"`
}

type NextcloudFileResponse struct {
	ThumbnailURL string `json:"thumbnailUrl"`
	Title        string `json:"title"`
	ResourceURL  string `json:"resourceUrl"`
	FileID       string `json:"fileId"`
	Path         string `json:"path"`
}

func (m *NextcloudFileMetadata) UnmarshalJSON(data []byte) error {
	type Alias NextcloudFileMetadata
	aux := &struct {
		Attributes map[string]string `json:"attributes"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Attributes != nil {
		m.FileID = aux.Attributes["fileId"]
		m.Path = aux.Attributes["path"]
	}
	return nil
}

type NextcloudSearchResponse struct {
	OCS struct {
		Data struct {
			Entries []NextcloudFileMetadata `json:"entries"`
		} `json:"data"`
	} `json:"ocs"`
}

func NewNextcloudService(cnf config.ConfigService, secretService secretProvider.SecretService) *NextcloudService {
	apiLookup, _ := url.JoinPath(cnf.GetNextcloudBaseUrl(), "ocs/v2.php/search/providers/files/search")
	username := secretService.GetNextcloudUser()
	password := secretService.GetNextcloudPassword()
	return &NextcloudService{
		apiLookupUrl:  apiLookup,
		username:      username,
		password:      password,
		cnfService:    cnf,
		secretService: secretService,
	}
}

func (s *NextcloudService) TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error {
	// TODO: Implement Nextcloud WebDAV or API upload logic here.
	fmt.Printf("Transferring file '%s' to Nextcloud at %s\n", header.Filename, s.apiLookupUrl)
	return nil
}
func (s *NextcloudService) LookupFile(ctx context.Context, filename string) (any, error) {
	// TODO: Implement Nextcloud file lookup logic here.
	req, err := http.NewRequestWithContext(ctx, "GET", s.apiLookupUrl, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("term", filename)
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth(s.username, s.password)
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: received status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResp NextcloudSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	var results []NextcloudFileResponse
	for _, entry := range searchResp.OCS.Data.Entries {
		results = append(results, NextcloudFileResponse{
			ThumbnailURL: entry.ThumbnailURL,
			Title:        entry.Title,
			ResourceURL:  entry.ResourceURL,
			FileID:       entry.FileID,
			Path:         entry.Path,
		})
	}

	return results, nil
}
func (s *NextcloudService) CheckStatus(ctx context.Context) error {
	// simple request to the WebDAV endpoint.
	req, err := http.NewRequestWithContext(ctx, "GET", s.apiLookupUrl, nil)
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
