package auth

import "time"

// AuthProvider validates an opaque token and returns the clientID.
// Implement this for any external credential system (static API tokens, OAuth, etc.).
type AuthProvider interface {
	ValidateToken(token string) (string, error)
}

// JwtService handles OneLab-signed JWT issuance and validation.
type JwtService interface {
	NewJwtClient(clientID string, clientSecret string)
	ValidateCredentials(clientID string, clientSecret string) bool
	GenerateToken(clientID string) (string, error)
	ValidateJWT(token string) (string, error)
	TokenDuration() time.Duration
}
