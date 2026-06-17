package fileService

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/HTMLuke/OneLab-API/secretProvider"
)

// NextcloudService handles sending files to Nextcloud.
type NextcloudService struct {
	apiLookupUrl string
	apiAddUrl    string
	apiGetUrl    string
	username     string
	password     string
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

func NewNextcloudService(baseURL string, secretService secretProvider.SecretService) (*NextcloudService, error) {
	username, err := secretService.GetSecret("ONELAB_NEXTCLOUD_USER")
	if err != nil {
		return nil, err
	}
	password, err := secretService.GetSecret("ONELAB_NEXTCLOUD_PASSWORD")
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		baseURL = "http://nextcloud.local/"
	}
	apiLookup, _ := url.JoinPath(baseURL, "ocs/v2.php/search/providers/files/search")
	apiAdd, _ := url.JoinPath(baseURL, "remote.php/dav/files/")
	apiGet, _ := url.JoinPath(baseURL, "remote.php/dav/files/")
	return &NextcloudService{
		apiLookupUrl: apiLookup,
		apiAddUrl:    apiAdd,
		apiGetUrl:    apiGet,
		username:     username,
		password:     password,
	}, nil
}

func (s *NextcloudService) AddFile(ctx context.Context, file []byte, filename string) error {
	// Build the full WebDAV destination URL: .../remote.php/dav/files/USERNAME/filename
	cleanFilename, err := url.PathUnescape(filename)
	if err != nil {
		return fmt.Errorf("failed to unescape filename: %v", err)
	}

	// 2. CRITICAL: Replace slashes with a filesystem-safe character (e.g., a hyphen)
	// Nextcloud/Linux filesystems cannot have "/" in a filename.
	cleanFilename = strings.ReplaceAll(cleanFilename, "/", "∕")  // Slash -> Division Slash
	cleanFilename = strings.ReplaceAll(cleanFilename, "\\", "⧵") // Backslash -> Reverse Solidus
	cleanFilename = strings.ReplaceAll(cleanFilename, ":", "∶")  // Colon -> Ratio Symbol
	cleanFilename = strings.ReplaceAll(cleanFilename, "?", "？")  // Question mark -> Fullwidth Question Mark
	cleanFilename = strings.ReplaceAll(cleanFilename, "*", "⁎")  // Asterisk -> Low Asterisk

	// 3. Build the URL cleanly using url.JoinPath
	targetURL, err := url.JoinPath(s.apiAddUrl, s.username, cleanFilename)
	if err != nil {
		return fmt.Errorf("failed to construct WebDAV URL: %v", err)
	}
	// Create a PUT request, passing the file byte slice as an io.Reader
	req, err := http.NewRequestWithContext(ctx, "PUT", targetURL, bytes.NewReader(file))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %v", err)
	}

	//  Set basic authentication headers
	req.SetBasicAuth(s.username, s.password)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute PUT request: %v", err)
	}
	defer resp.Body.Close()

	// Handle Nextcloud/WebDAV responses
	// WebDAV usually returns 201 (Created) for new files or 204 (No Content) if overwriting
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: received status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
func (s *NextcloudService) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	// Build url to: .../remote.php/dav/files/USERNAME/?X-File-Id=12345
	userURL, err := url.JoinPath(s.apiGetUrl, s.username)
	fullURL, err := url.JoinPath(userURL, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %v", err)
	}

	// add quersy parameter to URL
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	u.RawQuery = q.Encode()

	// create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// set basic auth header
	req.SetBasicAuth(s.username, s.password)

	// send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// error handling for non-success status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: received status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// save file content into bytes
	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %v", err)
	}

	// return file content as bytes (or you could return an io.Reader or any other format depending on your needs)
	return fileData, nil
}
func (s *NextcloudService) LookupFile(ctx context.Context, filename string) (any, error) {
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
