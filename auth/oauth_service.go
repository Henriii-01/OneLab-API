package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/HTMLuke/OneLab-API/secretProvider"
)

type oauthClient struct {
	clientSecret string
}

type oauthService struct {
	authClientID     string
	authClientSecret string
	tokenURL         string
	introspectionURL string
	scope            string
	httpClient       *http.Client
	logger           *slog.Logger
}

type oidcDiscoveryResponse struct {
	Issuer        string `json:"issuer"`
	TokenEndpoint string `json:"token_endpoint"`
}

type introspectionResponse struct {
	Active   bool   `json:"active"`
	Sub      string `json:"sub"`
	Username string `json:"username"`
	ClientID string `json:"client_id"`
}

func init() {
	RegisterAuth("oauth", NewOAuthService)
}

func NewOAuthService(_ time.Duration, secretService secretProvider.SecretService, logger *slog.Logger) (AuthService, error) {
	clientID, err := secretService.GetSecret("ONELAB_OAUTH_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	clientSecret, err := secretService.GetSecret("ONELAB_OAUTH_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}

	issuer := strings.TrimRight(os.Getenv("ONELAB_OAUTH_ISSUER_URL"), "/")
	tokenURL := strings.TrimSpace(os.Getenv("ONELAB_OAUTH_TOKEN_URL"))
	introspectionURL := strings.TrimSpace(os.Getenv("ONELAB_OAUTH_INTROSPECTION_URL"))
	scope := strings.TrimSpace(os.Getenv("ONELAB_OAUTH_SCOPE"))

	if issuer == "" && tokenURL == "" {
		return nil, errors.New("oauth auth requires ONELAB_OAUTH_ISSUER_URL or ONELAB_OAUTH_TOKEN_URL")
	}

	if issuer != "" && tokenURL == "" {
		discovery, err := discoverOIDC(issuer)
		if err != nil {
			return nil, err
		}
		tokenURL = discovery.TokenEndpoint
	}

	if tokenURL == "" {
		return nil, errors.New("oauth auth is missing token endpoint configuration")
	}

	if introspectionURL == "" {
		introspectionURL, err = deriveSiblingEndpoint(tokenURL, "introspect")
		if err != nil {
			return nil, fmt.Errorf("oauth auth is missing introspection endpoint configuration: %w", err)
		}
	}

	return &oauthService{
		authClientID:     clientID,
		authClientSecret: clientSecret,
		tokenURL:         tokenURL,
		introspectionURL: introspectionURL,
		scope:            scope,
		httpClient:       &http.Client{Timeout: 10 * time.Second},
		logger:           logger,
	}, nil
}

func discoverOIDC(issuer string) (*oidcDiscoveryResponse, error) {
	requestURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	response, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch oidc discovery document: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc discovery returned %s", response.Status)
	}

	var discovery oidcDiscoveryResponse
	if err := json.NewDecoder(response.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("failed to decode oidc discovery document: %w", err)
	}
	if discovery.TokenEndpoint == "" {
		return nil, errors.New("oidc discovery document is missing token_endpoint")
	}
	return &discovery, nil
}

func (s *oauthService) ValidateCredentials(clientID, clientSecret string) bool {
	return clientID != "" && clientSecret != ""
}

func (s *oauthService) GenerateToken(username, password string) (string, int, error) {
	if username == "" || password == "" {
		return "", 0, errInvalidCredentials
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", s.authClientID)
	form.Set("username", username)
	form.Set("password", password)

	request, err := http.NewRequest(http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", 0, fmt.Errorf("oauth token request failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read oauth token response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return "", 0, errInvalidCredentials
		}
		return "", 0, fmt.Errorf("oauth token endpoint returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", 0, fmt.Errorf("failed to decode oauth token response: %w", err)
	}
	if tokenResponse.AccessToken == "" {
		return "", 0, errors.New("oauth token response did not include an access_token")
	}

	return tokenResponse.AccessToken, tokenResponse.ExpiresIn, nil
}

func (s *oauthService) ValidateToken(tokenString string) (string, error) {
	clientID, clientSecret, err := s.clientCredentials()
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("token", tokenString)
	form.Set("token_type_hint", "access_token")

	request, err := http.NewRequest(http.MethodPost, s.introspectionURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, clientSecret)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("oauth introspection request failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read oauth introspection response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth introspection endpoint returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var introspection introspectionResponse
	if err := json.Unmarshal(body, &introspection); err != nil {
		return "", fmt.Errorf("failed to decode oauth introspection response: %w", err)
	}
	if !introspection.Active {
		return "", errors.New("invalid token")
	}

	if introspection.Sub != "" {
		return introspection.Sub, nil
	}
	if introspection.Username != "" {
		return introspection.Username, nil
	}
	if introspection.ClientID != "" {
		return introspection.ClientID, nil
	}

	return "", errors.New("valid token did not include a subject")
}

func (s *oauthService) clientCredentials() (string, string, error) {
	if s.authClientID == "" || s.authClientSecret == "" {
		return "", "", errors.New("oauth client credentials are not configured")
	}
	return s.authClientID, s.authClientSecret, nil
}

func deriveSiblingEndpoint(baseURL, sibling string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	trimmedPath := strings.TrimRight(parsedURL.Path, "/")
	lastSlash := strings.LastIndex(trimmedPath, "/")
	if lastSlash <= 0 {
		return "", fmt.Errorf("cannot derive %s endpoint from %q", sibling, baseURL)
	}

	parsedURL.Path = trimmedPath[:lastSlash+1] + sibling + "/"
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String(), nil
}
