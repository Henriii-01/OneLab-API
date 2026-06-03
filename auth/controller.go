package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type AuthController struct {
	service AuthService
}

func NewAuthController(service AuthService) *AuthController {

	return &AuthController{service: service}
}

type tokenRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}
type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

// Token Handler that checks the provided token for the credentials
func (c *AuthController) Token(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !c.service.ValidateCredentials(req.ClientID, req.ClientSecret) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := c.service.GenerateToken(req.ClientID)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse{
		Token:     token,
		ExpiresIn: int(c.service.TokenDuration() / time.Second),
	})
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

		clientID, err := c.service.ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		_ = clientID // TODO
		next.ServeHTTP(w, r)
	})
}
