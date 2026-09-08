package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

type AuthController struct {
	// A map of available auth services ("jwt")
	integrations map[string]AuthService
	logger       *slog.Logger
}

func NewAuthController(logger *slog.Logger) *AuthController {
	return &AuthController{integrations: make(map[string]AuthService), logger: logger}
}

// AddIntegration allows registering more auth services dynamically
func (c *AuthController) AddIntegration(name string, service AuthService) {
	c.integrations[name] = service
}

type tokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

// IssueToken authenticates the caller via clientId & clientSecret, then issues a signed token type is defined by AddIntegration
func (c *AuthController) IssueToken(w http.ResponseWriter, r *http.Request) {
	if c.logger != nil {
		c.logger.DebugContext(r.Context(), "token request started", "content_type", r.Header.Get("Content-Type"))
	}
	var req tokenRequest

	contentType := r.Header.Get("Content-Type")

	// Parse the request body depending on the Content-Type header
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			c.logOutcome(r, "token request rejected", http.StatusBadRequest)
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		req.Username = r.FormValue("username")
		req.Password = r.FormValue("password")
	} else {
		// Fall back to JSON parsing by default
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			c.logOutcome(r, "token request rejected", http.StatusBadRequest)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	for _, service := range c.integrations {
		token, expiresIn, err := service.GenerateToken(req.Username, req.Password)
		if err != nil {
			if errors.Is(err, errInvalidCredentials) {
				continue
			}
			http.Error(w, "failed to issue token", http.StatusInternalServerError)
			c.logOutcome(r, "token request failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(tokenResponse{
			Token:     token,
			ExpiresIn: expiresIn,
		})
		if err != nil {
			c.logOutcome(r, "token response encoding failed", http.StatusInternalServerError)
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
		c.logOutcome(r, "token issued", http.StatusOK)
		return
	}

	http.Error(w, "invalid credentials", http.StatusUnauthorized)
	c.logOutcome(r, "token request rejected", http.StatusUnauthorized)
}

func (c *AuthController) logOutcome(r *http.Request, message string, status int) {
	if c.logger == nil {
		return
	}
	level := slog.LevelInfo
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	} else if status >= http.StatusBadRequest {
		level = slog.LevelWarn
	}
	c.logger.Log(r.Context(), level, message, "status", status)
}

var errInvalidCredentials = errors.New("invalid credentials")

// Middleware validates the bearer token against all registered services.
func (c *AuthController) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token == authHeader || token == "" {
			c.logOutcome(r, "request rejected: missing authorization", http.StatusUnauthorized)
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		for _, service := range c.integrations {
			if _, err := service.ValidateToken(token); err == nil {
				next.ServeHTTP(w, r)
				return
			}
		}

		c.logOutcome(r, "request rejected: invalid authorization", http.StatusUnauthorized)
		http.Error(w, "invalid token", http.StatusUnauthorized)
	})
}

func (c *AuthController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/token", c.IssueToken)
}
