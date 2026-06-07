package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type AuthController struct {
	service      JwtService
	integrations map[string]AuthProvider
}

func NewAuthController(service JwtService) *AuthController {
	return &AuthController{service: service, integrations: make(map[string]AuthProvider)}
}

type tokenRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Token        string `json:"token"`
}
type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

func (c *AuthController) Token(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	clientID, err := c.authenticate(req)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := c.service.GenerateToken(clientID)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tokenResponse{
		Token:     token,
		ExpiresIn: int(c.service.TokenDuration() / time.Second),
	})
}

// authenticate resolves a clientID from either credentials or an API token.
func (c *AuthController) authenticate(req tokenRequest) (string, error) {
	if req.ClientID != "" && req.ClientSecret != "" {
		if c.service.ValidateCredentials(req.ClientID, req.ClientSecret) {
			return req.ClientID, nil
		}
		return "", errors.New("invalid credentials")
	}

	if req.Token != "" {
		for _, provider := range c.integrations {
			if clientID, err := provider.ValidateToken(req.Token); err == nil {
				return clientID, nil
			}
		}
	}

	return "", errors.New("no valid credentials provided")
}

func (c *AuthController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/token", c.Token)
}

func (c *AuthController) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token == authHeader || token == "" {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		clientID, err := c.service.ValidateJWT(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		_ = clientID // TODO
		next.ServeHTTP(w, r)
	})
}

func (c *AuthController) AddIntegration(name string, provider AuthProvider) {
	c.integrations[name] = provider
}
