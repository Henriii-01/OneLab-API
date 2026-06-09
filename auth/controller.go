package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

type AuthController struct {
	// A map of available auth services ("jwt")
	integrations map[string]AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{integrations: make(map[string]AuthService)}
}

// AddIntegration allows registering more auth services dynamically
func (c *AuthController) AddIntegration(name string, service AuthService) {
	c.integrations[name] = service
}

type tokenRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}
type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

// Token authenticates the caller via clientId & clientSecret, then issues a signed token
func (c *AuthController) IssueToken(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Find the service that recognizes these credentials and let it issue the token.
	for _, service := range c.integrations {
		if !service.ValidateCredentials(req.ClientID, req.ClientSecret) {
			continue
		}

		token, expiresIn, err := service.GenerateToken(req.ClientID)
		if err != nil {
			http.Error(w, "failed to issue token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(tokenResponse{
			Token:     token,
			ExpiresIn: expiresIn,
		})
		if err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "invalid credentials", http.StatusUnauthorized)
}

// Middleware validates the bearer token against all registered services.
func (c *AuthController) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token == authHeader || token == "" {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		for _, service := range c.integrations {
			if _, err := service.ValidateToken(token); err == nil {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "invalid token", http.StatusUnauthorized)
	})
}

func (c *AuthController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/token", c.IssueToken)
}
